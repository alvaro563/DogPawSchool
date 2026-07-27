package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewDeleteIncompatibilityInput(t *testing.T) {
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewDeleteIncompatibilityInput(0)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("negative_id", func(t *testing.T) {
		_, err := NewDeleteIncompatibilityInput(-1)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestDeleteIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error { return domain.ErrNotFound },
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewDeleteIncompatibilityInput(999))
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("in_use", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error { return domain.ErrIncompatibilityInUse },
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewDeleteIncompatibilityInput(3))
		assert.True(t, errors.Is(err, ErrInUse))
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedID int
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error {
				capturedID = id
				return nil
			},
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewDeleteIncompatibilityInput(5))
		assert.NoError(t, err)
		assert.Equal(t, 5, out.ID)
		assert.Equal(t, 5, capturedID)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error { return repoErr },
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewDeleteIncompatibilityInput(5))
		assert.True(t, errors.Is(err, repoErr))
	})
}
