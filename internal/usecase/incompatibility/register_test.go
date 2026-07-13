package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterIncompatibilityInput {
	return RegisterIncompatibilityInput{
		Name:  "Reacciona mal al transportin",
		Level: domain.IncompatibilityLevelMedia,
	}
}

func TestRegisterIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation_empty_name", func(t *testing.T) {
		uc := NewRegisterIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), RegisterIncompatibilityInput{Level: domain.IncompatibilityLevelMedia})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "name", verr.Field)
	})

	t.Run("validation_invalid_level", func(t *testing.T) {
		uc := NewRegisterIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), RegisterIncompatibilityInput{Name: "x", Level: "OTHER"})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "level", verr.Field)
	})

	t.Run("validation_does_not_call_repo", func(t *testing.T) {
		called := false
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				called = true
				return 0, nil
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, _ = uc.Execute(context.Background(), RegisterIncompatibilityInput{})
		assert.False(t, called)
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
				return 0, ErrDuplicateName
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
