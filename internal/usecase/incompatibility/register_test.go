package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validTriggerInput() RegisterIncompatibilityInput {
	return MustNewRegisterIncompatibilityInput("Reacciona mal al transportin", domain.IncompatibilityLevelMedia,
		 "", "MIEDOSO")
}

func validTraitInput() RegisterIncompatibilityInput {
	return MustNewRegisterIncompatibilityInput("Miedoso", domain.IncompatibilityLevelBaja,
		 "MIEDOSO", "")
}

func TestNewRegisterIncompatibilityInput(t *testing.T) {
	t.Parallel()
	t.Run("empty_name", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("", domain.IncompatibilityLevelMedia,
			 "", "MIEDOSO")
		assertValidationError(t, err, "name")
	})

	t.Run("invalid_level", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("x", domain.IncompatibilityLevel("OTHER"),
			 "", "MIEDOSO")
		assertValidationError(t, err, "level")
	})

	t.Run("trait_without_code", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("x", domain.IncompatibilityLevelMedia,
			 "", "")
		assertValidationError(t, err, "code")
	})

	t.Run("trigger_without_target", func(t *testing.T) {
		_, err := NewRegisterIncompatibilityInput("x", domain.IncompatibilityLevelMedia,
			 "", "")
		assertValidationError(t, err, "code")
	})
}

func TestRegisterIncompatibilityUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_trigger", func(t *testing.T) {
		var captured *domain.Incompatibility
		mock := &mockIncompatibilityRepository{
			getByCode: func(ctx context.Context, code string) (*domain.Incompatibility, error) {
				assert.Equal(t, "MIEDOSO", code)
				return mustNewTrait(9, "MIEDOSO", "Miedoso", domain.IncompatibilityLevelBaja), nil
			},
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				captured = incomp
				return 5, nil
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), validTriggerInput())
		assert.NoError(t, err)
		assert.Equal(t, 5, out.ID)
		assert.NotNil(t, captured)
		assert.Equal(t, 0, captured.ID(), "id must be 0 (will be set by repo)")
		assert.Equal(t, "Reacciona mal al transportin", captured.Name())
		assert.Equal(t, domain.IncompatibilityLevelMedia, captured.Type())
		assert.True(t, captured.TargetTraitCode() != "")
		assert.Equal(t, "MIEDOSO", captured.TargetTraitCode())
	})

	t.Run("happy_path_trait", func(t *testing.T) {
		var captured *domain.Incompatibility
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				captured = incomp
				return 6, nil
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), validTraitInput())
		assert.NoError(t, err)
		assert.Equal(t, 6, out.ID)
		assert.NotNil(t, captured)
		assert.True(t, captured.Code() != "")
		assert.Equal(t, "MIEDOSO", captured.Code())
	})

	t.Run("trigger_target_does_not_exist", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			getByCode: func(ctx context.Context, code string) (*domain.Incompatibility, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validTriggerInput())
		assertValidationError(t, err, "target_trait_code")
	})

	t.Run("trigger_target_is_not_a_trait", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			getByCode: func(ctx context.Context, code string) (*domain.Incompatibility, error) {
				return mustNewTrigger(2, "Otro trigger", domain.IncompatibilityLevelMedia, "OTRO"), nil
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validTriggerInput())
		assertValidationError(t, err, "target_trait_code")
	})

	t.Run("duplicate_name", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				return 0, domain.ErrDuplicateIncompatibilityName
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validTraitInput())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateName))
	})

	t.Run("duplicate_code", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				return 0, domain.ErrDuplicateIncompatibilityCode
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validTraitInput())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateCode))
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			create: func(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
				return 0, repoErr
			},
		}
		uc := NewRegisterIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), validTraitInput())
		assert.True(t, errors.Is(err, repoErr))
	})
}
