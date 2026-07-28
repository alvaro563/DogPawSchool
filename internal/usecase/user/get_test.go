package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestNewGetUserInput(t *testing.T) {
	t.Run("valid_id", func(t *testing.T) {
		in, err := NewGetUserInput(5)
		require.NoError(t, err)
		assert.Equal(t, 5, in.ID())
	})

	t.Run("zero_id", func(t *testing.T) {
		_, err := NewGetUserInput(0)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr), "expected *ValidationError, got %T", err)
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("negative_id", func(t *testing.T) {
		_, err := NewGetUserInput(-1)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestGetUserUseCase_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := newTestUser(7)
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) {
				assert.Equal(t, 7, id)
				return expected, nil
			},
		}
		uc := NewGetUserUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewGetUserInput(7))
		require.NoError(t, err)
		assert.Equal(t, expected, out.User)
	})

	t.Run("not_found", func(t *testing.T) {
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) {
				return nil, nil
			},
		}
		uc := NewGetUserUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewGetUserInput(99))
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("repo_error_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &mockUserRepository{
			getByID: func(_ context.Context, id int) (*domain.User, error) {
				return nil, repoErr
			},
		}
		uc := NewGetUserUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewGetUserInput(1))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr), "expected wrapped repoErr")
		assert.Contains(t, err.Error(), "get user 1:")
	})
}
