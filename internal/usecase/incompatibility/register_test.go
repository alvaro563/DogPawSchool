package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterIncompatibilityInput {
	return MustNewRegisterIncompatibilityInput("Reacciona mal al transportin", domain.IncompatibilityLevelMedia)
}

func TestNewRegisterIncompatibilityInput(t *testing.T) {
	t.Parallel()
	t.Run("empty_name", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("", domain.IncompatibilityLevelMedia)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "name", verr.Field)
	})

	t.Run("invalid_level", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("x", domain.IncompatibilityLevel("OTHER"))
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "level", verr.Field)
	})
}

func TestRegisterIncompatibilityUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("factory_blocks_invalid_before_use_case_runs", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("", domain.IncompatibilityLevelMedia)
		assert.Error(t, err)
	})

	t.Run("happy_path", func(t *testing.T) {
		var captured *domain.Incompatibility
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				captured = incomp
				return 5, nil
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), validRegisterInput())
		assert.NoError(t, err)
		assert.Equal(t, 5, out.ID)
		assert.NotNil(t, captured)
		assert.Equal(t, 0, captured.ID(), "id must be 0 (will be set by repo)")
		assert.Equal(t, "Reacciona mal al transportin", captured.Name())
		assert.Equal(t, domain.IncompatibilityLevelMedia, captured.Type())
	})

	t.Run("duplicate_name", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				return 0, domain.ErrDuplicateIncompatibilityName
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validRegisterInput())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateName))
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				return 0, repoErr
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validRegisterInput())
		assert.True(t, errors.Is(err, repoErr))
	})
}
