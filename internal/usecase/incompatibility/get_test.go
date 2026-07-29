package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewGetIncompatibilityInput(t *testing.T) {
	t.Parallel()
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewGetIncompatibilityInput(0)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("negative_id", func(t *testing.T) {
		_, err := NewGetIncompatibilityInput(-1)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestGetIncompatibilityUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("not_found", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return nil, nil },
		}
		uc := NewGetIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewGetIncompatibilityInput(999))
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
		out, err := uc.Execute(context.Background(), MustNewGetIncompatibilityInput(3))
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
		_, err := uc.Execute(context.Background(), MustNewGetIncompatibilityInput(1))
		assert.True(t, errors.Is(err, repoErr))
	})
}
