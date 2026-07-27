package pass

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyPassInput is the validated command to apply a partial
// update to a pass. The patch *values* are validated by
// domain.Pass.ApplyPatch — defense in depth.
type ModifyPassInput struct {
	id    int
	patch domain.PassPatch
}

func (in ModifyPassInput) ID() int                 { return in.id }
func (in ModifyPassInput) Patch() domain.PassPatch { return in.patch }

// NewModifyPassInput validates id > 0. Empty patch is allowed
// (Execute short-circuits to a no-op, preserving the current
// behavior).
func NewModifyPassInput(id int, patch domain.PassPatch) (ModifyPassInput, error) {
	if id <= 0 {
		return ModifyPassInput{}, &ValidationError{Field: "id"}
	}
	return ModifyPassInput{id: id, patch: patch}, nil
}

// MustNewModifyPassInput panics on validation error. For tests.
func MustNewModifyPassInput(id int, patch domain.PassPatch) ModifyPassInput {
	in, err := NewModifyPassInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
}

// ModifyPassOutput carries the post-mutation pass. The full
// domain object is returned so the handler can serialize it
// directly.
type ModifyPassOutput struct {
	Pass *domain.Pass
}

// ModifyPassUseCase applies a partial update to a pass. An
// empty patch is a no-op and returns the unmodified pass without
// touching the database.
type ModifyPassUseCase struct {
	repo domain.PassRepository
}

func NewModifyPassUseCase(repo domain.PassRepository) *ModifyPassUseCase {
	return &ModifyPassUseCase{repo: repo}
}

func (uc *ModifyPassUseCase) Execute(ctx context.Context, input ModifyPassInput) (ModifyPassOutput, error) {
	pass, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return ModifyPassOutput{}, fmt.Errorf("get pass %d: %w", input.ID(), err)
	}
	if pass == nil {
		return ModifyPassOutput{}, ErrNotFound
	}

	patch := input.Patch()
	if err := pass.ApplyPatch(patch); err != nil {
		var passValidationErr *domain.PassValidationError
		if errors.As(err, &passValidationErr) {
			return ModifyPassOutput{}, &ValidationError{Field: passValidationErr.Field}
		}
		return ModifyPassOutput{}, err
	}

	if isEmptyPassPatch(patch) {
		return ModifyPassOutput{Pass: pass}, nil
	}

	if err := uc.repo.Update(ctx, pass); err != nil {
		return ModifyPassOutput{}, fmt.Errorf("update pass %d: %w", input.ID(), err)
	}
	// Re-fetch to surface the post-update updatedAt (set by the DB
	// trigger on every UPDATE). Without this, the response would
	// carry the pre-update updatedAt from the in-memory pass.
	updatedPass, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return ModifyPassOutput{}, fmt.Errorf("get updated pass %d: %w", input.ID(), err)
	}
	return ModifyPassOutput{Pass: updatedPass}, nil
}

func isEmptyPassPatch(patch domain.PassPatch) bool {
	return patch.Price == nil && patch.PassType == nil && patch.ExpiresAt == nil
}
