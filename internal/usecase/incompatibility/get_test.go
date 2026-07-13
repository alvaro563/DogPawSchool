package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestGetIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation_zero_id", func(t *testing.T) {
		uc := NewGetIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), GetIncompatibilityInput{})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("validation_negative_id", func(t *testing.T) {
		uc := NewGetIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), GetIncompatibilityInput{ID: -1})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("not_found", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return nil, nil },
		}
		uc := NewGetIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), GetIncompatibilityInput{ID: 999})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("happy_path", func(t *testing.T) {
		want := mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				assert.Equal(t, 3, id)
				return want, nil
			},
		}
		uc := NewGetIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), GetIncompatibilityInput{ID: 3})
		assert.NoError(t, err)
		assert.Same(t, want, out.Incompatibility)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return nil, repoErr
			},
		}
		uc := NewGetIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), GetIncompatibilityInput{ID: 1})
		assert.True(t, errors.Is(err, repoErr))
	})
}
