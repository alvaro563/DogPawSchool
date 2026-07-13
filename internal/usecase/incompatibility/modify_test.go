package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestModifyIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation_zero_id", func(t *testing.T) {
		uc := NewModifyIncompatibilityUseCase(&mockIncompatibilityRepository{})
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("not_found", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return nil, nil },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		newName := "x"
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 999, Patch: domain.IncompatibilityPatch{Name: &newName}})
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("empty_patch_is_noop", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
		updateCalled := false
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
			update:                 func(ctx context.Context, incomp *domain.Incompatibility) error { updateCalled = true; return nil },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{}})
		assert.NoError(t, err)
		assert.Equal(t, existing, out.Incompatibility)
		assert.False(t, updateCalled)
	})

	t.Run("partial_patch_renames_only", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
		var updated *domain.Incompatibility
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
			update:                 func(ctx context.Context, incomp *domain.Incompatibility) error { updated = incomp; return nil },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		newName := "Miedo a petardos y cohetes"
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Name: &newName}})
		assert.NoError(t, err)
		assert.Equal(t, "Miedo a petardos y cohetes", updated.Name(), "name changed")
		assert.Equal(t, domain.IncompatibilityLevelBaja, updated.Type(), "level preserved")
	})

	t.Run("partial_patch_changes_level_only", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
			update: func(ctx context.Context, incomp *domain.Incompatibility) error {
				assert.Equal(t, domain.IncompatibilityLevelAbsoluta, incomp.Type())
				return nil
			},
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		newLevel := domain.IncompatibilityLevelAbsoluta
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Level: &newLevel}})
		assert.NoError(t, err)
	})

	t.Run("patch_with_empty_name_returns_validation_error", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "x", domain.IncompatibilityLevelBaja)
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		empty := ""
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Name: &empty}})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "name", verr.Field)
	})

	t.Run("patch_with_invalid_level_returns_validation_error", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "x", domain.IncompatibilityLevelBaja)
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		invalid := domain.IncompatibilityLevel("OTHER")
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Level: &invalid}})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "level", verr.Field)
	})

	t.Run("duplicate_name", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "x", domain.IncompatibilityLevelBaja)
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
			update:                 func(ctx context.Context, incomp *domain.Incompatibility) error { return ErrDuplicateName },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		newName := "y"
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Name: &newName}})
		assert.True(t, errors.Is(err, ErrDuplicateName))
	})

	t.Run("update_returns_error", func(t *testing.T) {
		existing := mustNewIncompatibility(3, "x", domain.IncompatibilityLevelBaja)
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) { return existing, nil },
			update:                 func(ctx context.Context, incomp *domain.Incompatibility) error { return repoErr },
		}
		uc := NewModifyIncompatibilityUseCase(mock)
		newName := "y"
		_, err := uc.Execute(context.Background(), ModifyIncompatibilityInput{ID: 3, Patch: domain.IncompatibilityPatch{Name: &newName}})
		assert.True(t, errors.Is(err, repoErr))
	})
}
