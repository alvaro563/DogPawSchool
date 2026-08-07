package postgres

import (
	"context"
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
		domain.TypeRoute, 8, 3, date)
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id)
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
		domain.TypeSocialization, 10, 2, date.Add(1*24*time.Hour))
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), activity2)
	require.NoError(t, err)

	list, err := repo.List(context.Background(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	list2, err := repo.List(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Len(t, list2, 1)

	upcoming, err := repo.ListUpcoming(context.Background(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, upcoming, 2)
}

func TestActivityRepository_Update(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewActivityRepository(db)
	activity := insertBaseActivity(t, db)

	patched, err := domain.NewActivity(activity.ID(), "Paseo Actualizado", "", "Nuevo Lugar",
		domain.TypeIndividual, 5, 2, activity.Date().Add(1*time.Hour))
	require.NoError(t, err)

	err = repo.Update(context.Background(), patched)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), activity.ID())
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

	got, err := repo.GetByIDForUpdate(context.Background(), activity.ID())
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
	_, err := repo.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	nonexistent, err := domain.NewActivity(9999, "Ghost", "", "Nowhere",
		domain.TypeRoute, 5, 1, time.Now().Add(30*24*time.Hour))
	require.NoError(t, err)
	err = repo.Update(context.Background(), nonexistent)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	err = repo.Delete(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
