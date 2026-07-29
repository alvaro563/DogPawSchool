package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestDogRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewDogRepository(db)

	dog, err := domain.NewDog(0, "Luna", "Labrador", "ES-001", 24,
		domain.SexFemale, 22.5, user.ID())
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Luna", got.Name())
	assert.Equal(t, "Labrador", got.Breed())
	assert.Equal(t, 24, got.AgeInMonths())
	assert.Equal(t, domain.SexFemale, got.Sex())
	assert.Equal(t, 22.5, got.WeightKg())
	assert.Equal(t, "ES-001", got.Passport())
	assert.Equal(t, user.ID(), got.UserID())
	assert.True(t, got.IsActive())
	assert.Empty(t, got.Incompatibilities())
}

func TestDogRepository_UpdateWithIncompatibilities(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	incomp := insertBaseIncompatibility(t, db)
	repo := NewDogRepository(db)

	dog, err := domain.NewDog(0, "Toby", "Beagle", "ES-002", 36,
		domain.SexMale, 12.0, user.ID())
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)

	dog2, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	added, err := dog2.AddIncompatibility(incomp)
	require.NoError(t, err)
	assert.True(t, added)

	err = repo.Update(context.Background(), dog2)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, got.Incompatibilities(), 1)
	assert.Equal(t, incomp.ID(), got.Incompatibilities()[0].ID())
}

func TestDogRepository_ListByOwner(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user1 := insertBaseUser(t, db)
	user2, err := domain.NewUser(0, "Second Owner", "second@test.com",
		repeatedString("s", 60), domain.RoleRegular)
	require.NoError(t, err)
	userRepo := NewUserRepository(db)
	_, err = userRepo.Create(context.Background(), user2)
	require.NoError(t, err)
	user2got, err := userRepo.GetByEmail(context.Background(), "second@test.com")
	require.NoError(t, err)

	repo := NewDogRepository(db)
	d1, _ := domain.NewDog(0, "Dog1", "Lab", "ES-A1", 12, domain.SexMale, 10, user1.ID())
	d2, _ := domain.NewDog(0, "Dog2", "Lab", "ES-A2", 12, domain.SexMale, 10, user1.ID())
	d3, _ := domain.NewDog(0, "Dog3", "Lab", "ES-A3", 12, domain.SexMale, 10, user2got.ID())
	_, _ = repo.Create(context.Background(), d1)
	_, _ = repo.Create(context.Background(), d2)
	_, _ = repo.Create(context.Background(), d3)

	list, err := repo.ListByOwner(context.Background(), user1.ID(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	list2, err := repo.ListByOwner(context.Background(), user2got.ID(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, list2, 1)
}

func TestDogRepository_Lists(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewDogRepository(db)

	// Luna: age 24 → SemiAdult, weight 22.5 → Large
	d, _ := domain.NewDog(0, "Luna", "Labrador", "ES-L1", 24, domain.SexFemale, 22.5, user.ID())
	_, _ = repo.Create(context.Background(), d)

	// Toby: age 6 → Children, weight 8.0 → Medium
	d2, _ := domain.NewDog(0, "Toby", "Beagle", "ES-T1", 6, domain.SexMale, 8.0, user.ID())
	id2, _ := repo.Create(context.Background(), d2)

	// Max: age 48 → Adult, weight 35.0 → Large
	d3, _ := domain.NewDog(0, "Max", "Pastor Alemán", "ES-M1", 48, domain.SexMale, 35.0, user.ID())
	_, _ = repo.Create(context.Background(), d3)

	all, err := repo.ListAll(context.Background(), false, 50, 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	active, err := repo.ListAll(context.Background(), true, 50, 0)
	require.NoError(t, err)
	assert.Len(t, active, 3)

	byBreed, err := repo.ListByBreed(context.Background(), "Beagle", 50, 0)
	require.NoError(t, err)
	assert.Len(t, byBreed, 1)
	assert.Equal(t, "Toby", byBreed[0].Name())

	bySex, err := repo.ListBySex(context.Background(), domain.SexMale, 50, 0)
	require.NoError(t, err)
	assert.Len(t, bySex, 2)

	byNeutered, err := repo.ListByNeutered(context.Background(), false, 50, 0)
	require.NoError(t, err)
	assert.Len(t, byNeutered, 3)

	byHeat, err := repo.ListByHeat(context.Background(), false, 50, 0)
	require.NoError(t, err)
	assert.Len(t, byHeat, 3)

	byActive, err := repo.ListByIsActive(context.Background(), true, 50, 0)
	require.NoError(t, err)
	assert.Len(t, byActive, 3)

	// age 24 → SemiAdult, 48 → Adult
	byAge, err := repo.ListByAgeBracket(context.Background(), domain.AgeBracketSemiAdult, 50, 0)
	require.NoError(t, err)
	assert.Len(t, byAge, 1)

	// weight 8.0 → Medium
	bySize, err := repo.ListBySizeBracket(context.Background(), domain.SizeBracketLarge, 50, 0)
	require.NoError(t, err)
	assert.Len(t, bySize, 2)

	repo2 := NewDogRepository(db)
	dog2, err := repo2.GetByID(context.Background(), id2)
	require.NoError(t, err)
	dog2.Deactivate()
	err = repo2.Update(context.Background(), dog2)
	require.NoError(t, err)

	active2, err := repo.ListByIsActive(context.Background(), true, 50, 0)
	require.NoError(t, err)
	assert.Len(t, active2, 2)

	active3, err := repo.ListByIsActive(context.Background(), false, 50, 0)
	require.NoError(t, err)
	assert.Len(t, active3, 1)
}

func TestDogRepository_GetByIDForUpdate(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	incomp := insertBaseIncompatibility(t, db)
	repo := NewDogRepository(db)

	dog, err := domain.NewDog(0, "ForUpdate Dog", "Mixed", "ES-FU1", 18,
		domain.SexFemale, 15.0, user.ID())
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)

	dog2, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	dog2.AddIncompatibility(incomp)
	repo.Update(context.Background(), dog2)

	got, err := repo.GetByIDForUpdate(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, dog.Name(), got.Name())
	require.Len(t, got.Incompatibilities(), 1)
	assert.Equal(t, incomp.ID(), got.Incompatibilities()[0].ID())
}

func TestDogRepository_ErrDuplicatePassport(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewDogRepository(db)

	dog, _ := domain.NewDog(0, "First", "Lab", "ES-DUP", 12, domain.SexMale, 10, user.ID())
	_, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)

	dup, _ := domain.NewDog(0, "Second", "Lab", "ES-DUP", 12, domain.SexMale, 10, user.ID())
	_, err = repo.Create(context.Background(), dup)
	assert.ErrorIs(t, err, domain.ErrDuplicatePassport)
}

func TestDogRepository_ErrInvalidUser(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewDogRepository(db)
	dog, _ := domain.NewDog(0, "Orphan", "Mix", "ES-ORPHAN", 12, domain.SexMale, 10, 99999)
	_, err := repo.Create(context.Background(), dog)
	assert.ErrorIs(t, err, domain.ErrInvalidUserReference)
}

func TestDogRepository_GetByIDNotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewDogRepository(db)
	_, err := repo.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestConcurrency_DogIncompatibilities(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	ctx := context.Background()
	user := insertBaseUser(t, db)

	dog := insertBaseDog(t, db, user.ID())

	incompA, err := domain.NewIncompatibility(0, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta)
	require.NoError(t, err)
	incompRepo := NewIncompatibilityRepository(db)
	idA, err := incompRepo.Create(ctx, incompA)
	require.NoError(t, err)
	incompA, err = incompRepo.GetIncompatibilityByID(ctx, idA)
	require.NoError(t, err)

	incompB, err := domain.NewIncompatibility(0, "Miedo a perros grandes", domain.IncompatibilityLevelMedia)
	require.NoError(t, err)
	idB, err := incompRepo.Create(ctx, incompB)
	require.NoError(t, err)
	incompB, err = incompRepo.GetIncompatibilityByID(ctx, idB)
	require.NoError(t, err)

	var (
		wg   sync.WaitGroup
		errs [2]error
	)
	transactor := NewTransactor(db)
	dogRepo := NewDogRepository(db)

	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			chosen := incompA
			if idx == 1 {
				chosen = incompB
			}
			errs[idx] = transactor.WithinTx(ctx, func(txCtx context.Context) error {
				d, err := dogRepo.GetByIDForUpdate(txCtx, dog.ID())
				if err != nil {
					return fmt.Errorf("get dog: %w", err)
				}
				if _, err := d.AddIncompatibility(chosen); err != nil {
					return fmt.Errorf("add incompatibility: %w", err)
				}
				return dogRepo.Update(txCtx, d)
			})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d should succeed with FOR UPDATE", i)
	}

	finalDog, err := dogRepo.GetByID(ctx, dog.ID())
	require.NoError(t, err)
	require.Len(t, finalDog.Incompatibilities(), 2,
		"both incompatibilities must be present — no lost update")

	ids := make(map[int]bool)
	for _, inc := range finalDog.Incompatibilities() {
		ids[inc.ID()] = true
	}
	assert.True(t, ids[idA], "incompatibility A must be present")
	assert.True(t, ids[idB], "incompatibility B must be present")
}
