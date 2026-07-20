package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// dbRunner is the minimal subset of *sql.DB / *sql.Tx that the
// repositories need. Both types satisfy it natively, so the helper
// runner() can transparently route repository calls to the live
// connection pool or to an in-flight transaction.
type dbRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// txKey is the unexported context key under which a *sql.Tx is stored
// while a Transactor.WithinTx closure is running. The key is an empty
// struct so it never collides with keys defined in other packages.
type txKey struct{}

// Transactor runs a function inside a single database transaction.
// If the function returns nil the transaction is committed; otherwise
// it is rolled back. Panics are propagated after a rollback.
type Transactor struct {
	db *sql.DB
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

// WithinTx begins a transaction, attaches the *sql.Tx to the derived
// context, and runs fn. Repository methods that read the context via
// runner() will automatically use the transaction; methods that do
// not will continue to use the live *sql.DB. Use case implementations
// that need to return a value from inside the closure should capture
// it in a variable that lives in the enclosing scope.
func (transactor *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := transactor.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if panicValue := recover(); panicValue != nil {
			_ = tx.Rollback()
			panic(panicValue)
		}
	}()
	txCtx := withTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w (rollback: %v)", err, rollbackErr)
		}
		return err
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit tx: %w", commitErr)
	}
	return nil
}

// withTx returns a child context carrying the given *sql.Tx. Repository
// methods call TxFrom(ctx) to retrieve the transaction (or nil when
// running outside a transactor).
func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFrom extracts the *sql.Tx stored in ctx by WithinTx, or nil when
// no transaction is in flight. Repository methods call this via
// runner().
func TxFrom(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

// runner returns the dbRunner a repository method should use for the
// given context: the live *sql.DB by default, or the *sql.Tx
// attached by Transactor.WithinTx when one is in flight. This is the
// only place where the "in or out of transaction" decision is made.
func runner(ctx context.Context, db *sql.DB) dbRunner {
	if tx := TxFrom(ctx); tx != nil {
		return tx
	}
	return db
}
