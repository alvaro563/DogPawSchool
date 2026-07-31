package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUserEmailsUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		emails := []string{"one@example.com", "two@example.com"}
		repo := &mockUserRepository{
			listAllEmails: func(_ context.Context) ([]string, error) {
				return emails, nil
			},
		}
		uc := NewListUserEmailsUseCase(repo)
		out, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, emails, out.Emails)
	})

	t.Run("empty_result", func(t *testing.T) {
		repo := &mockUserRepository{
			listAllEmails: func(_ context.Context) ([]string, error) {
				return []string{}, nil
			},
		}
		uc := NewListUserEmailsUseCase(repo)
		out, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Empty(t, out.Emails)
	})

	t.Run("repo_error_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("connection lost")
		repo := &mockUserRepository{
			listAllEmails: func(_ context.Context) ([]string, error) {
				return nil, repoErr
			},
		}
		uc := NewListUserEmailsUseCase(repo)
		_, err := uc.Execute(context.Background())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
		assert.Contains(t, err.Error(), "list user emails:")
	})
}
