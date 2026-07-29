package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactor_Commit(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	transactor := NewTransactor(db)
	err := transactor.WithinTx(context.Background(), func(txCtx context.Context) error {
		_, execErr := runner(txCtx, db).ExecContext(txCtx,
			`INSERT INTO users (name, email, password, role, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			"Committed User", "commit@test.com", repeatedString("b", 60), "ADMIN", true)
		return execErr
	})
	require.NoError(t, err)

	assertRowCount(t, db, "users", 1)
}

func TestTransactor_RollbackOnError(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	transactor := NewTransactor(db)
	sentinel := errors.New("rollback reason")
	err := transactor.WithinTx(context.Background(), func(txCtx context.Context) error {
		_, execErr := runner(txCtx, db).ExecContext(txCtx,
			`INSERT INTO users (name, email, password, role, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			"Rollback User", "rollback@test.com", repeatedString("c", 60), "ADMIN", true)
		if execErr != nil {
			return execErr
		}
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
	assertRowCount(t, db, "users", 0)
}

func TestTransactor_RollbackOnPanic(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	transactor := NewTransactor(db)
	assert.Panics(t, func() {
		_ = transactor.WithinTx(context.Background(), func(txCtx context.Context) error {
			_, execErr := runner(txCtx, db).ExecContext(txCtx,
				`INSERT INTO users (name, email, password, role, is_active)
				 VALUES ($1, $2, $3, $4, $5)`,
				"Panic User", "panic@test.com", repeatedString("d", 60), "ADMIN", true)
			if execErr != nil {
				panic(execErr)
			}
			panic("boom")
		})
	})
	assertRowCount(t, db, "users", 0)
}

func TestTransactor_NestedWithinTxJoinsOuter(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	transactor := NewTransactor(db)
	var innerID int

	err := transactor.WithinTx(context.Background(), func(txCtx context.Context) error {
		_, execErr := runner(txCtx, db).ExecContext(txCtx,
			`INSERT INTO users (name, email, password, role, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			"Outer User", "outer@test.com", repeatedString("e", 60), "ADMIN", true)
		if execErr != nil {
			return execErr
		}

		err2 := transactor.WithinTx(txCtx, func(innerCtx context.Context) error {
			_, execErr2 := runner(innerCtx, db).ExecContext(innerCtx,
				`INSERT INTO users (name, email, password, role, is_active)
				 VALUES ($1, $2, $3, $4, $5)`,
				"Inner User", "inner@test.com", repeatedString("f", 60), "ADMIN", true)
			return execErr2
		})
		return err2
	})
	require.NoError(t, err)
	assert.Equal(t, 0, innerID)
	assertRowCount(t, db, "users", 2)
}
