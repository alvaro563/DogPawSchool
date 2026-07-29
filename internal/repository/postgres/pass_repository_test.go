package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestPassRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewPassRepository(db)
	now := time.Now().UTC()

	pass, err := domain.NewPass(0, 10, 8, 5000, domain.PassGeneric, user.ID(), now, now, nil)
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), pass)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 10, got.NumOfSessions())
	assert.Equal(t, 8, got.RemainingSessions())
	assert.Equal(t, 5000, got.Price())
	assert.Equal(t, domain.PassGeneric, got.Type())
	assert.Equal(t, user.ID(), got.UserID())
	assert.Greater(t, got.RemainingSessions(), 0)

	list, err := repo.ListByOwner(context.Background(), user.ID(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestPassRepository_UpdateWithMovement(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewPassRepository(db)

	pass := insertBasePass(t, db, user.ID())

	movement, err := pass.ConsumeSession("Paseo grupal", time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, movement)

	err = repo.Update(context.Background(), pass)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	assert.Equal(t, pass.NumOfSessions()-1, got.RemainingSessions())
}

func TestPassRepository_GetByIDForUpdate(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	pass := insertBasePass(t, db, user.ID())
	repo := NewPassRepository(db)

	got, err := repo.GetByIDForUpdate(context.Background(), pass.ID())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, pass.ID(), got.ID())
	assert.Equal(t, pass.RemainingSessions(), got.RemainingSessions())
}

func TestPassRepository_ListAll(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewPassRepository(db)
	now := time.Now().UTC()

	p1, _ := domain.NewPass(0, 5, 5, 2500, domain.PassGeneric, user.ID(), now, now, nil)
	p2, _ := domain.NewPass(0, 1, 1, 1200, domain.PassSpecial, user.ID(), now, now, nil)
	_, _ = repo.Create(context.Background(), p1)
	_, _ = repo.Create(context.Background(), p2)

	all, err := repo.ListAll(context.Background(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestPassRepository_NotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewPassRepository(db)
	_, err := repo.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestPassRepository_ErrInvalidUser(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewPassRepository(db)
	now := time.Now().UTC()
	p, _ := domain.NewPass(0, 5, 5, 1000, domain.PassGeneric, 99999, now, now, nil)
	_, err := repo.Create(context.Background(), p)
	assert.ErrorIs(t, err, domain.ErrInvalidUserReference)
}

func TestConcurrency_PassSession(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	ctx := context.Background()
	now := time.Now().UTC()

	user := insertBaseUser(t, db)

	pass, err := domain.NewPass(0, 1, 1, 2500, domain.PassGeneric, user.ID(), now, now, nil)
	require.NoError(t, err)
	passRepo := NewPassRepository(db)
	passID, err := passRepo.Create(ctx, pass)
	require.NoError(t, err)

	var (
		wg        sync.WaitGroup
		successes atomic.Int32
		failures  atomic.Int32
	)
	errPassExhausted := errors.New("pass exhausted")
	transactor := NewTransactor(db)
	numGoroutines := 5

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := transactor.WithinTx(ctx, func(txCtx context.Context) error {
				p, err := passRepo.GetByIDForUpdate(txCtx, passID)
				if err != nil {
					return fmt.Errorf("get pass: %w", err)
				}
				if p.IsExhausted() {
					return errPassExhausted
				}
				if _, err := p.ConsumeSession("concurrent test", now); err != nil {
					return fmt.Errorf("consume session: %w", err)
				}
				return passRepo.Update(txCtx, p)
			})
			if errors.Is(err, errPassExhausted) {
				failures.Add(1)
			} else if err != nil {
				failures.Add(1)
			} else {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successes.Load(), "exactly one goroutine must consume the single session")
	assert.Equal(t, int32(4), failures.Load(), "the remaining 4 must find the pass exhausted")

	finalPass, err := passRepo.GetByID(ctx, passID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalPass.NumOfSessions())
	assert.Equal(t, 0, finalPass.RemainingSessions(), "remaining must be 0 after one consumption")
}
