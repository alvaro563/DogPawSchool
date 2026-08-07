package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestIncompatibilityRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewIncompatibilityRepository(db)

	incomp, err := domain.NewTriggerIncompatibility(0, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO")
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), incomp)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetIncompatibilityByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Reactivo a machos enteros", got.Name())
	assert.Equal(t, domain.IncompatibilityLevelAbsoluta, got.Type())

	incomp2, err := domain.NewTriggerIncompatibility(0, "Miedo a perros grandes", domain.IncompatibilityLevelMedia, "OTRO_PERRO")
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), incomp2)
	require.NoError(t, err)

	list, err := repo.List(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestIncompatibilityRepository_Update(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewIncompatibilityRepository(db)
	incomp := insertBaseIncompatibility(t, db)

	updated, err := domain.NewTriggerIncompatibility(incomp.ID(), "Descripción actualizada", domain.IncompatibilityLevelMedia, "MACHO_ENTERO")
	require.NoError(t, err)

	err = repo.Update(context.Background(), updated)
	require.NoError(t, err)

	got, err := repo.GetIncompatibilityByID(context.Background(), incomp.ID())
	require.NoError(t, err)
	assert.Equal(t, "Descripción actualizada", got.Name())
	assert.Equal(t, domain.IncompatibilityLevelMedia, got.Type())
}

func TestIncompatibilityRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewIncompatibilityRepository(db)
	incomp := insertBaseIncompatibility(t, db)

	err := repo.Delete(context.Background(), incomp.ID())
	require.NoError(t, err)

	_, err = repo.GetIncompatibilityByID(context.Background(), incomp.ID())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestIncompatibilityRepository_NotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewIncompatibilityRepository(db)
	_, err := repo.GetIncompatibilityByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	nonexistent, err := domain.NewTriggerIncompatibility(9999, "Nonexistent", domain.IncompatibilityLevelBaja, "MACHO_ENTERO")
	require.NoError(t, err)
	err = repo.Update(context.Background(), nonexistent)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	err = repo.Delete(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
