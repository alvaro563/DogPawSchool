package dog

import "context"

// Transactor runs a function inside a single database transaction.
// If the function returns nil the transaction is committed; otherwise
// it is rolled back. Panics are propagated after a rollback.
//
// When a transaction is already in flight on the context (attached by
// an outer Transactor), implementations must simply run the closure
// within that existing transaction rather than opening a nested one.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
