package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestUserRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewUserRepository(db)
	password := repeatedString("z", 60)
	user, err := domain.NewUser(0, "Alice", "alice@test.com", password, domain.RoleRegular)
	require.NoError(t, err)

	_, err = repo.Create(context.Background(), user)
	require.NoError(t, err)

	got, err := repo.GetByEmail(context.Background(), "alice@test.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Alice", got.Name())
	assert.Equal(t, "alice@test.com", got.Email())
	assert.Equal(t, password, got.Password())
	assert.Equal(t, domain.RoleRegular, got.Role())
	assert.True(t, got.IsActive())

	got2, err := repo.GetByID(context.Background(), got.ID())
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, got.Name(), got2.Name())
	assert.Equal(t, got.Email(), got2.Email())

	user2, err := domain.NewUser(0, "Bob", "bob@test.com", repeatedString("x", 60), domain.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), user2)
	require.NoError(t, err)

	list, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 2)

	paged, err := repo.ListAllPaged(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Len(t, paged, 1)
}

func TestUserRepository_ListAllEmails(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewUserRepository(db)
	for _, tc := range []struct {
		name, email string
	}{
		{"Alice", "alice@test.com"},
		{"Bob", "bob@test.com"},
		{"Carla", "carla@test.com"},
	} {
		user, err := domain.NewUser(0, tc.name, tc.email, repeatedString("z", 60), domain.RoleRegular)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), user)
		require.NoError(t, err)
	}

	emails, err := repo.ListAllEmails(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"alice@test.com", "bob@test.com", "carla@test.com"}, emails)
}

func TestUserRepository_ListAllEmails_Empty(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewUserRepository(db)
	emails, err := repo.ListAllEmails(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, emails, "empty result must be a non-nil slice")
	assert.Empty(t, emails)
}

func TestUserRepository_Update(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewUserRepository(db)

	updated, err := domain.NewUser(user.ID(), "Alice Updated", "alice.new@test.com", user.Password(), domain.RoleAdmin)
	require.NoError(t, err)

	err = repo.Update(context.Background(), updated)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), user.ID())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Alice Updated", got.Name())
	assert.Equal(t, "alice.new@test.com", got.Email())
}

func TestUserRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	user := insertBaseUser(t, db)
	repo := NewUserRepository(db)

	err := repo.Delete(context.Background(), user.ID())
	require.NoError(t, err)

	_, err = repo.GetByID(context.Background(), user.ID())
	assert.ErrorIs(t, err, domain.ErrNotFound)

	err = repo.Delete(context.Background(), user.ID())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepository_ErrDuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewUserRepository(db)
	user, err := domain.NewUser(0, "First", "dup@test.com", repeatedString("a", 60), domain.RoleRegular)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), user)
	require.NoError(t, err)

	dup, err := domain.NewUser(0, "Second", "dup@test.com", repeatedString("b", 60), domain.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), dup)
	assert.ErrorIs(t, err, domain.ErrDuplicateEmail)
}

func TestUserRepository_GetByEmailNotFound(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(func() { cleanTables(t, db) })

	repo := NewUserRepository(db)
	_, err := repo.GetByEmail(context.Background(), "nobody@test.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
