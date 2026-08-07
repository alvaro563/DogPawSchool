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

func TestReservationRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	dog := insertBaseDog(t, db, user.ID())
	activity := insertBaseActivity(t, db)
	pass := insertBasePass(t, db, user.ID())

	repo := NewReservationRepository(db)
	res, err := domain.NewReservation(0, activity.ID(), dog.ID(), pass.ID(), time.Now().UTC())
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), res)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, activity.ID(), got.ActivityID())
	assert.Equal(t, dog.ID(), got.DogID())
	assert.Equal(t, pass.ID(), got.PassID())
}

func TestReservationRepository_ListByActivity(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	dog := insertBaseDog(t, db, user.ID())
	act1 := insertBaseActivity(t, db)
	act2, err := domain.NewActivity(0, "Segunda Actividad", "", "Otro Lugar",
		domain.TypeIndividual, 5, 2, time.Now().Add(21*24*time.Hour))
	require.NoError(t, err)
	actRepo := NewActivityRepository(db)
	id2, err := actRepo.Create(context.Background(), act2)
	require.NoError(t, err)
	act2got, err := actRepo.GetByID(context.Background(), id2)
	require.NoError(t, err)
	pass := insertBasePass(t, db, user.ID())

	repo := NewReservationRepository(db)
	r1, _ := domain.NewReservation(0, act1.ID(), dog.ID(), pass.ID(), time.Now().UTC())
	r2, _ := domain.NewReservation(0, act2got.ID(), dog.ID(), pass.ID(), time.Now().UTC())
	_, _ = repo.Create(context.Background(), r1)
	_, _ = repo.Create(context.Background(), r2)

	list, err := repo.ListByActivity(context.Background(), act1.ID())
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestReservationRepository_ListByDog(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	dog1 := insertBaseDog(t, db, user.ID())
	dog2, err := domain.NewDog(0, "Max", "Mix", "ES-RES2", 12, domain.SexMale, 10, user.ID())
	require.NoError(t, err)
	dogRepo := NewDogRepository(db)
	dog2id, err := dogRepo.Create(context.Background(), dog2)
	require.NoError(t, err)
	dog2got, err := dogRepo.GetByID(context.Background(), dog2id)
	require.NoError(t, err)

	activity := insertBaseActivity(t, db)
	pass := insertBasePass(t, db, user.ID())

	repo := NewReservationRepository(db)
	r1, _ := domain.NewReservation(0, activity.ID(), dog1.ID(), pass.ID(), time.Now().UTC())
	r2, _ := domain.NewReservation(0, activity.ID(), dog2got.ID(), pass.ID(), time.Now().UTC())
	_, _ = repo.Create(context.Background(), r1)
	_, _ = repo.Create(context.Background(), r2)

	list, err := repo.ListByDog(context.Background(), dog1.ID())
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestReservationRepository_NotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewReservationRepository(db)
	_, err := repo.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestConcurrency_ActivityCapacity(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	ctx := context.Background()
	now := time.Now().UTC()

	user := insertBaseUser(t, db)

	actRepo := NewActivityRepository(db)
	activity, err := domain.NewActivity(0, "Concurrency Capacity Test", "", "Test Location",
		domain.TypeRoute, 1, 1, now.Add(7*24*time.Hour))
	require.NoError(t, err)
	actID, err := actRepo.Create(ctx, activity)
	require.NoError(t, err)

	numGoroutines := 5
	dogRepo := NewDogRepository(db)
	passRepo := NewPassRepository(db)
	dogIDs := make([]int, numGoroutines)
	passIDs := make([]int, numGoroutines)

	for i := range numGoroutines {
		dog, err := domain.NewDog(0, fmt.Sprintf("ConcDog-%d", i), "Mix",
			fmt.Sprintf("ES-CONC-%d", i), 12, domain.SexMale, 10.0, user.ID())
		require.NoError(t, err)
		did, err := dogRepo.Create(ctx, dog)
		require.NoError(t, err)
		dogIDs[i] = did

		pass, err := domain.NewPass(0, 5, 5, 2500, domain.PassGeneric, user.ID(), now, now, nil)
		require.NoError(t, err)
		pid, err := passRepo.Create(ctx, pass)
		require.NoError(t, err)
		passIDs[i] = pid
	}

	var (
		wg        sync.WaitGroup
		successes atomic.Int32
		failures  atomic.Int32
	)
	errActivityFull := errors.New("activity full")
	transactor := NewTransactor(db)
	resRepo := NewReservationRepository(db)

	for i := range numGoroutines {
		wg.Add(1)
		go func(dogID, passID int) {
			defer wg.Done()
			err := transactor.WithinTx(ctx, func(txCtx context.Context) error {
				act, err := actRepo.GetByIDForUpdate(txCtx, actID)
				if err != nil {
					return fmt.Errorf("get activity: %w", err)
				}
				existing, err := resRepo.ListByActivity(txCtx, actID)
				if err != nil {
					return fmt.Errorf("list reservations: %w", err)
				}
				confirmed := 0
				for _, r := range existing {
					if r.IsConfirmed() {
						confirmed++
					}
				}
				if act.IsFull(confirmed) {
					return errActivityFull
				}
				reservation, err := domain.NewReservation(0, actID, dogID, passID, now)
				if err != nil {
					return fmt.Errorf("build reservation: %w", err)
				}
				_, err = resRepo.Create(txCtx, reservation)
				return err
			})
			if errors.Is(err, errActivityFull) {
				failures.Add(1)
			} else if err != nil {
				failures.Add(1)
			} else {
				successes.Add(1)
			}
		}(dogIDs[i], passIDs[i])
	}
	wg.Wait()

	assert.Equal(t, int32(1), successes.Load(), "exactly one reservation must succeed (capacity=1)")
	assert.Equal(t, int32(4), failures.Load(), "the remaining 4 must fail (capacity full)")

	reservations, err := resRepo.ListByActivity(ctx, actID)
	require.NoError(t, err)
	assert.Len(t, reservations, 1)
	assert.True(t, reservations[0].IsConfirmed())
}
