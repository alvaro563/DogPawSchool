package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestActivateUserInput_Factory(t *testing.T) {
	t.Parallel()
	t.Run("valid_id", func(t *testing.T) {
		in, err := NewActivateUserInput(1)
		require.NoError(t, err)
		assert.Equal(t, 1, in.ID())
	})

	t.Run("zero_id", func(t *testing.T) {
		_, err := NewActivateUserInput(0)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("negative_id", func(t *testing.T) {
		_, err := NewActivateUserInput(-2)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestActivateUserUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("success_flips_flag_and_persists", func(t *testing.T) {
		existing := newTestUser(42)
		existing.Deactivate()
		require.False(t, existing.IsActive(), "precondition: user must start inactive")

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
				assert.True(t, u.IsActive(), "user must be active when Update is called")
				return nil
			},
		}
		uc := NewActivateUserUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewActivateUserInput(42))
		require.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.True(t, updateCalled)
		assert.True(t, existing.IsActive(), "in-memory user must now be active")
		assert.Same(t, existing, updated, "the same domain object must be passed to Update")
	})

	t.Run("already_active_is_noop", func(t *testing.T) {
		existing := newTestUser(42)
		require.True(t, existing.IsActive(), "precondition")

		var updateCalled bool
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { updateCalled = true; return nil },
		}
		uc := NewActivateUserUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewActivateUserInput(42))
		require.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.False(t, updateCalled, "activating an already-active user must not call Update")
	})

	t.Run("not_found", func(t *testing.T) {
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return nil, nil },
		}
		uc := NewActivateUserUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewActivateUserInput(99))
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("repo_error_on_get_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return nil, repoErr },
		}
		uc := NewActivateUserUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewActivateUserInput(42))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
		assert.Contains(t, err.Error(), "get user 42:")
	})

	t.Run("repo_error_on_update_is_wrapped", func(t *testing.T) {
		existing := newTestUser(42)
		existing.Deactivate()
		repoErr := errors.New("concurrent modification")
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) { return existing, nil },
			update:  func(_ context.Context, u *domain.User) error { return repoErr },
		}
		uc := NewActivateUserUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewActivateUserInput(42))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
		assert.Contains(t, err.Error(), "activate user 42:")
	})
}
