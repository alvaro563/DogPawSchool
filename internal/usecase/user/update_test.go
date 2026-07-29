package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestNewUpdateUserInput(t *testing.T) {
	t.Parallel()
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewUpdateUserInput(0, domain.UserPatch{})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("negative_id", func(t *testing.T) {
		_, err := NewUpdateUserInput(-3, domain.UserPatch{})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("valid_empty_patch", func(t *testing.T) {
		in, err := NewUpdateUserInput(1, domain.UserPatch{})
		require.NoError(t, err)
		assert.Equal(t, 1, in.ID())
		assert.True(t, in.Patch().IsEmpty())
	})
}

func TestUpdateUserUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("success_name_only", func(t *testing.T) {
		existing := newTestUser(42)
		var updated *domain.User
		var updateCalled bool
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) {
				assert.Equal(t, 42, id)
				return existing, nil
			},
			update: func(_ context.Context, u *domain.User) error {
				updateCalled = true
				updated = u
				return nil
			},
		}
		uc := NewUpdateUserUseCase(repo)
		newName := "Luna"
		out, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Name: &newName}))
		require.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.True(t, updateCalled)
		assert.Equal(t, "Luna", updated.Name())
		assert.Equal(t, "test@example.com", updated.Email(), "email preserved")
	})

	t.Run("success_email_only", func(t *testing.T) {
		existing := newTestUser(42)
		var updated *domain.User
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { updated = u; return nil },
		}
		uc := NewUpdateUserUseCase(repo)
		newEmail := "luna@example.com"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Email: &newEmail}))
		require.NoError(t, err)
		assert.Equal(t, "luna@example.com", updated.Email())
		assert.Equal(t, "Test User", updated.Name(), "name preserved")
	})

	t.Run("success_both_fields", func(t *testing.T) {
		existing := newTestUser(42)
		var updated *domain.User
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { updated = u; return nil },
		}
		uc := NewUpdateUserUseCase(repo)
		newName := "Luna"
		newEmail := "luna@example.com"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{
			Name:  &newName,
			Email: &newEmail,
		}))
		require.NoError(t, err)
		assert.Equal(t, "Luna", updated.Name())
		assert.Equal(t, "luna@example.com", updated.Email())
	})

	t.Run("empty_patch_is_noop", func(t *testing.T) {
		existing := newTestUser(42)
		var updateCalled bool
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { updateCalled = true; return nil },
		}
		uc := NewUpdateUserUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{}))
		require.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.False(t, updateCalled, "empty patch must not call repo.Update")
	})

	t.Run("invalid_name_returns_validation_error", func(t *testing.T) {
		existing := newTestUser(42)
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
		}
		uc := NewUpdateUserUseCase(repo)
		empty := ""
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Name: &empty}))
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr), "expected *ValidationError, got %T", err)
		assert.Equal(t, "name", verr.Field)
	})

	t.Run("invalid_email_returns_validation_error", func(t *testing.T) {
		existing := newTestUser(42)
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
		}
		uc := NewUpdateUserUseCase(repo)
		malformed := "not-an-email"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Email: &malformed}))
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr), "expected *ValidationError, got %T", err)
		assert.Equal(t, "email", verr.Field)
	})

	t.Run("not_found", func(t *testing.T) {
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return nil, nil },
		}
		uc := NewUpdateUserUseCase(repo)
		newName := "Luna"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(99, domain.UserPatch{Name: &newName}))
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("repo_error_on_get_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("database timeout")
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return nil, repoErr },
		}
		uc := NewUpdateUserUseCase(repo)
		newName := "Luna"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Name: &newName}))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("repo_error_on_update_is_wrapped", func(t *testing.T) {
		existing := newTestUser(42)
		repoErr := errors.New("concurrent modification")
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { return repoErr },
		}
		uc := NewUpdateUserUseCase(repo)
		newName := "Luna"
		_, err := uc.Execute(context.Background(), MustNewUpdateUserInput(42, domain.UserPatch{Name: &newName}))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
		assert.Contains(t, err.Error(), "update user 42:")
	})
}
