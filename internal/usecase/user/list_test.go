package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestNewListUsersInput(t *testing.T) {
	t.Run("normalizes_zero_limit", func(t *testing.T) {
		in, err := NewListUsersInput(0, 0)
		require.NoError(t, err)
		assert.Equal(t, defaultPageLimit, in.Limit())
		assert.Equal(t, 0, in.Offset())
	})

	t.Run("normalizes_negative_limit", func(t *testing.T) {
		in, err := NewListUsersInput(-5, 0)
		require.NoError(t, err)
		assert.Equal(t, defaultPageLimit, in.Limit())
	})

	t.Run("clamps_overmax_limit", func(t *testing.T) {
		in, err := NewListUsersInput(500, 0)
		require.NoError(t, err)
		assert.Equal(t, maxPageLimit, in.Limit())
	})

	t.Run("keeps_valid_limit", func(t *testing.T) {
		in, err := NewListUsersInput(25, 0)
		require.NoError(t, err)
		assert.Equal(t, 25, in.Limit())
	})

	t.Run("normalizes_negative_offset", func(t *testing.T) {
		in, err := NewListUsersInput(10, -7)
		require.NoError(t, err)
		assert.Equal(t, 0, in.Offset())
	})

	t.Run("keeps_valid_offset", func(t *testing.T) {
		in, err := NewListUsersInput(10, 30)
		require.NoError(t, err)
		assert.Equal(t, 30, in.Offset())
	})
}

func TestListUsersUseCase_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		users := []*domain.User{newTestUser(1), newTestUser(2)}
		repo := &mockUserRepository{
			listAllPaged: func(_ context.Context, limit, offset int) ([]*domain.User, error) {
				assert.Equal(t, 10, limit)
				assert.Equal(t, 20, offset)
				return users, nil
			},
		}
		uc := NewListUsersUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListUsersInput(10, 20))
		require.NoError(t, err)
		assert.Equal(t, users, out.Users)
	})

	t.Run("empty_result", func(t *testing.T) {
		repo := &mockUserRepository{
			listAllPaged: func(_ context.Context, limit, offset int) ([]*domain.User, error) {
				return []*domain.User{}, nil
			},
		}
		uc := NewListUsersUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListUsersInput(10, 0))
		require.NoError(t, err)
		assert.Empty(t, out.Users)
	})

	t.Run("repo_error_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("connection lost")
		repo := &mockUserRepository{
			listAllPaged: func(_ context.Context, limit, offset int) ([]*domain.User, error) {
				return nil, repoErr
			},
		}
		uc := NewListUsersUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewListUsersInput(10, 0))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
		assert.Contains(t, err.Error(), "list users:")
	})
}
