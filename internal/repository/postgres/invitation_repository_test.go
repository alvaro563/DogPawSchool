package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestInvitationRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewInvitationRepository(db)
	now := time.Now().UTC()
	expiresAt := now.Add(48 * time.Hour)

	inv1, err := domain.NewPendingInvitation(user.ID(), "invite@test.com", "tok-abc-1",
		domain.RoleRegular, expiresAt, now)
	require.NoError(t, err)

	id, err := repo.Create(context.Background(), inv1)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "invite@test.com", got.Email())
	assert.Equal(t, "tok-abc-1", got.Token())
	assert.Equal(t, domain.RoleRegular, got.Role())
	assert.Equal(t, domain.InvitationPending, got.Status())
	assert.Equal(t, expiresAt.Unix(), got.ExpiresAt().Unix())

	got2, err := repo.GetByToken(context.Background(), "tok-abc-1")
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, "invite@test.com", got2.Email())

	emails, err := repo.ListByEmail(context.Background(), "invite@test.com")
	require.NoError(t, err)
	assert.Len(t, emails, 1)

	inv2, err := domain.NewPendingInvitation(user.ID(), "other@test.com", "tok-xyz-2",
		domain.RoleRegular, expiresAt, now)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), inv2)
	require.NoError(t, err)

	emails2, err := repo.ListByEmail(context.Background(), "other@test.com")
	require.NoError(t, err)
	assert.Len(t, emails2, 1)

	pending, err := repo.ListPending(context.Background(), 50, 0)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
}

func TestInvitationRepository_Update(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewInvitationRepository(db)
	now := time.Now().UTC()
	expiresAt := now.Add(48 * time.Hour)

	inv, err := domain.NewPendingInvitation(user.ID(), "update@test.com", "tok-upd",
		domain.RoleRegular, expiresAt, now)
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	err = got.Accept()
	require.NoError(t, err)

	err = repo.Update(context.Background(), got)
	require.NoError(t, err)

	got2, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.InvitationAccepted, got2.Status())
}

func TestInvitationRepository_ErrDuplicateToken(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewInvitationRepository(db)
	now := time.Now().UTC()
	expiresAt := now.Add(48 * time.Hour)

	i1, err := domain.NewPendingInvitation(user.ID(), "first@test.com", "dup-tok",
		domain.RoleRegular, expiresAt, now)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), i1)
	require.NoError(t, err)

	i2, err := domain.NewPendingInvitation(user.ID(), "second@test.com", "dup-tok",
		domain.RoleAdmin, expiresAt, now)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), i2)
	assert.ErrorIs(t, err, domain.ErrDuplicateToken)
}

func TestInvitationRepository_GetByTokenNotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewInvitationRepository(db)
	_, err := repo.GetByToken(context.Background(), "non-existent-token")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestInvitationRepository_GetByIDNotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewInvitationRepository(db)
	_, err := repo.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
