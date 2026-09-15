package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestActivityRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)
	date := time.Now().Add(14 * 24 * time.Hour)
	activity, err := domain.NewActivity(0, "Ruta por la montaña", "", "Parque Natural",
		domain.TypeRoute, 8, 3, date, nil, nil)
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id, 0, true)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Ruta por la montaña", got.Name())
	assert.Equal(t, domain.TypeRoute, got.Type())
	assert.Equal(t, 8, got.MaxCapacity())
	assert.Equal(t, "Parque Natural", got.Location())
	assert.Equal(t, 3, got.DurationInHours())
	assert.WithinDuration(t, date, got.Date(), time.Microsecond)
	assert.False(t, got.IsClosed())

	activity2, err := domain.NewActivity(0, "Socialización grupal", "", "Centro",
		domain.TypeSocialization, 10, 2, date.Add(1*24*time.Hour), nil, nil)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), activity2)
	require.NoError(t, err)

	list, err := repo.List(context.Background(), 0, true, 50, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	list2, err := repo.List(context.Background(), 0, true, 1, 0)
	require.NoError(t, err)
	assert.Len(t, list2, 1)

	upcoming, err := repo.ListUpcoming(context.Background(), 0, true, 50, 0)
	require.NoError(t, err)
	assert.Len(t, upcoming, 2)
}

func TestActivityRepository_Update(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)
	activity := insertBaseActivity(t, db)
	// Insert a target dog so the FK on activities.dog_id resolves.
	insertDogForIndividualClassTest(t, db)

	individualTargetDogID := 1 // first dog id after cleanTables (RESTART IDENTITY)
	patched, err := domain.NewActivity(activity.ID(), "Paseo Actualizado", "", "Nuevo Lugar",
		domain.TypeIndividual, 5, 2, activity.Date().Add(1*time.Hour), &individualTargetDogID, nil)
	require.NoError(t, err)

	err = repo.Update(context.Background(), patched)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), activity.ID(), 0, true)
	require.NoError(t, err)
	assert.Equal(t, "Paseo Actualizado", got.Name())
	assert.Equal(t, "Nuevo Lugar", got.Location())
	assert.Equal(t, 5, got.MaxCapacity())
}

func TestActivityRepository_GetByIDForUpdate(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	activity := insertBaseActivity(t, db)
	repo := NewActivityRepository(db)

	got, err := repo.GetByIDForUpdate(context.Background(), activity.ID(), 0, true)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, activity.ID(), got.ID())
	assert.Equal(t, activity.Name(), got.Name())
	assert.Equal(t, activity.MaxCapacity(), got.MaxCapacity())
}

func TestActivityRepository_NotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)
	_, err := repo.GetByID(context.Background(), 9999, 0, true)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	nonexistent, err := domain.NewActivity(9999, "Ghost", "", "Nowhere",
		domain.TypeRoute, 5, 1, time.Now().Add(30*24*time.Hour), nil, nil)
	require.NoError(t, err)
	err = repo.Update(context.Background(), nonexistent)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	err = repo.Delete(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// TestActivityRepository_ListByClosed pins down the critical SQL
// edge case: when a bound is nil, the COALESCE-style predicate
// `$N::timestamptz IS NULL OR date >= $N` must short-circuit (not
// fall through to "date >= 0001-01-01"). A regression here would
// silently exclude every row in the open/closed-only listings.
func TestActivityRepository_ListByClosed(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)

	now := time.Now().UTC().Truncate(time.Second)
	// Three open in range + one closed in range + one closed outside.
	open1, err := domain.NewActivity(0, "Open 1", "", "L", domain.TypeRoute, 5, 1, now.Add(-48*time.Hour), nil, nil)
	require.NoError(t, err)
	open2, err := domain.NewActivity(0, "Open 2", "", "L", domain.TypeRoute, 5, 1, now.Add(-24*time.Hour), nil, nil)
	require.NoError(t, err)
	open3, err := domain.NewActivity(0, "Open 3", "", "L", domain.TypeRoute, 5, 1, now.Add(-72*time.Hour), nil, nil)
	require.NoError(t, err)
	closedIn, err := domain.NewActivity(0, "Closed In", "", "L", domain.TypeRoute, 5, 1, now.Add(-12*time.Hour), nil, nil)
	require.NoError(t, err)
	require.NoError(t, closedIn.Close())
	closedOut, err := domain.NewActivity(0, "Closed Out", "", "L", domain.TypeRoute, 5, 1, now.Add(48*time.Hour), nil, nil)
	require.NoError(t, err)
	require.NoError(t, closedOut.Close())

	for _, a := range []*domain.Activity{open1, open2, open3, closedIn, closedOut} {
		_, err := repo.Create(context.Background(), a)
		require.NoError(t, err)
	}

	// Persist the Close() state for closedIn and closedOut (Create
	// sets closed=false; we need to Update to flip it). The
	// Activity entity doesn't auto-set its id, so reload each by
	// name after Create.
	for _, target := range []*domain.Activity{closedIn, closedOut} {
		persisted, err := findActivityByName(repo, target.Name())
		require.NoError(t, err)
		require.NoError(t, persisted.Close())
		require.True(t, persisted.IsClosed(), "Close() must flip the entity flag in memory")
		require.NoError(t, repo.Update(context.Background(), persisted))

		// Confirm the DB persisted the flip: a fresh ListByClosed(true)
		// should return this name.
		out, err := repo.ListByClosed(context.Background(), 0, true, true, nil, nil, 100, 0)
		require.NoError(t, err)
		names := make(map[string]bool)
		for _, a := range out {
			names[a.Name()] = true
		}
		require.True(t, names[target.Name()], "%q must be persisted as closed=true", target.Name())
	}

	t.Run("closed_true_no_bounds", func(t *testing.T) {
		out, err := repo.ListByClosed(context.Background(), 0, true, true, nil, nil, 100, 0)
		require.NoError(t, err)
		assert.Len(t, out, 2)
		// Date ASC: closedIn (-12h, pasado) first, then closedOut (+48h, futuro).
		assert.Equal(t, "Closed In", out[0].Name())
		assert.Equal(t, "Closed Out", out[1].Name())
	})

	t.Run("closed_false_no_bounds", func(t *testing.T) {
		out, err := repo.ListByClosed(context.Background(), 0, true, false, nil, nil, 100, 0)
		require.NoError(t, err)
		assert.Len(t, out, 3)
		for _, a := range out {
			assert.False(t, a.IsClosed())
		}
	})

	t.Run("closed_true_with_bounds_filters_date", func(t *testing.T) {
		from := now.Add(-30 * time.Hour)
		to := now.Add(-1 * time.Hour)
		out, err := repo.ListByClosed(context.Background(), 0, true, true, &from, &to, 100, 0)
		require.NoError(t, err)
		assert.Len(t, out, 1)
		assert.Equal(t, "Closed In", out[0].Name())
	})

	t.Run("closed_true_bounds_inclusive", func(t *testing.T) {
		// closedIn is at now-12h. Bounds now-12h exactly to now-12h exactly should include it.
		exact := now.Add(-12 * time.Hour)
		out, err := repo.ListByClosed(context.Background(), 0, true, true, &exact, &exact, 100, 0)
		require.NoError(t, err)
		assert.Len(t, out, 1)
	})

	t.Run("closed_true_with_only_from", func(t *testing.T) {
		from := now.Add(-6 * time.Hour)
		out, err := repo.ListByClosed(context.Background(), 0, true, true, &from, nil, 100, 0)
		require.NoError(t, err)
		// closedIn (-12h) is BEFORE from → excluded; closedOut (+48h)
		// is AFTER from → included.
		assert.Len(t, out, 1)
		assert.Equal(t, "Closed Out", out[0].Name())
	})

	t.Run("closed_true_with_only_to", func(t *testing.T) {
		to := now.Add(24 * time.Hour)
		out, err := repo.ListByClosed(context.Background(), 0, true, true, nil, &to, 100, 0)
		require.NoError(t, err)
		// closedIn (-12h) is BEFORE to → included; closedOut (+48h)
		// is AFTER to → excluded.
		assert.Len(t, out, 1)
		assert.Equal(t, "Closed In", out[0].Name())
	})

	t.Run("nil_bounds_do_not_match_zero_time", func(t *testing.T) {
		// If a regression turns nil into time.Time{} zero value, this
		// query would return 0 rows because every date is "after
		// 0001-01-01".
		out, err := repo.ListByClosed(context.Background(), 0, true, true, nil, nil, 100, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, out, "nil bounds must not silently filter every row out")
	})

	t.Run("pagination", func(t *testing.T) {
		page1, err := repo.ListByClosed(context.Background(), 0, true, false, nil, nil, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1, 2)
		page2, err := repo.ListByClosed(context.Background(), 0, true, false, nil, nil, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2, 1)
	})
}

// findActivityByName scans all activities looking for one whose
// Name() matches. Used to recover the DB-assigned ID after Create
// (which returns the id as a separate int, not on the entity).
func findActivityByName(repo *ActivityRepository, name string) (*domain.Activity, error) {
	all, err := repo.List(context.Background(), 0, true, 100, 0)
	if err != nil {
		return nil, err
	}
	for _, a := range all {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, errors.New("not found: " + name)
}

// insertDogForIndividualClassTest inserts a fresh dog so tests can
// flip an activity's type to INDIVIDUAL_CLASS without the FK on
// activities.dog_id rejecting the update.
func insertDogForIndividualClassTest(t *testing.T, db *sql.DB) {
	t.Helper()
	dog, err := domain.NewDog(0, "IndividualTarget", "Mix", "ES-IND-TGT", 12,
		domain.SexFemale, 10.0, insertBaseUser(t, db).ID())
	require.NoError(t, err)
	dogRepo := NewDogRepository(db)
	_, err = dogRepo.Create(context.Background(), dog)
	require.NoError(t, err)
}

func TestActivityRepository_RoundTripWithSizeTarget(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)
	date := time.Now().Add(14 * 24 * time.Hour)

	mini := domain.SizeBracketMini
	activity, err := domain.NewActivity(0, "Paseo Minis", "", "Parque",
		domain.TypeRoute, 6, 1, date, nil, &mini)
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), id, 0, true)
	require.NoError(t, err)
	require.NotNil(t, got.SizeTarget(), "size_target must be persisted")
	assert.Equal(t, domain.SizeBracketMini, *got.SizeTarget())

	// Clearing via Update.
	cleared, err := domain.NewActivity(id, "Paseo Minis", "", "Parque",
		domain.TypeRoute, 6, 1, date, nil, nil)
	require.NoError(t, err)
	require.NoError(t, repo.Update(context.Background(), cleared))

	got2, err := repo.GetByID(context.Background(), id, 0, true)
	require.NoError(t, err)
	assert.Nil(t, got2.SizeTarget(), "size_target must be cleared on Update")
}
