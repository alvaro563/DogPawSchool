package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation_zero_id", func(t *testing.T) {
		uc := NewDeleteIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("validation_negative_id", func(t *testing.T) {
		uc := NewDeleteIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{ID: -1})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("not_found", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error { return ErrNotFound },
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{ID: 999})
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("in_use", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			delete: func(ctx context.Context, id int) error { return ErrInUse },
		}
		uc := NewDeleteIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{ID: 3})
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
		out, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{ID: 5})
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
		_, err := uc.Execute(context.Background(), DeleteIncompatibilityInput{ID: 5})
		assert.True(t, errors.Is(err, repoErr))
	})
}
