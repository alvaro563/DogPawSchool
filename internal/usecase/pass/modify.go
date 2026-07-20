package pass

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyPassInput is the input for a partial update of an existing
// pass. Only the editable fields (price, pass_type, expires_at) can
// be patched; all other fields are intentionally omitted from the
// patch type.
type ModifyPassInput struct {
	ID    int
	Patch domain.PassPatch
}

// ModifyPassOutput carries the post-mutation pass. The full domain
// object is returned so the handler can serialize it directly.
type ModifyPassOutput struct {
	Pass *domain.Pass
}

// ModifyPassUseCase applies a partial update to a pass. An empty
// patch is a no-op and returns the unmodified pass without touching
// the database.
type ModifyPassUseCase struct {
	repo domain.PassRepository
}

func NewModifyPassUseCase(repo domain.PassRepository) *ModifyPassUseCase {
	return &ModifyPassUseCase{repo: repo}
}

func (uc *ModifyPassUseCase) Execute(ctx context.Context, input ModifyPassInput) (ModifyPassOutput, error) {
	if input.ID <= 0 {
		return ModifyPassOutput{}, &ValidationError{Field: "id"}
	}

	pass, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return ModifyPassOutput{}, fmt.Errorf("get pass %d: %w", input.ID, err)
	}
	if pass == nil {
		return ModifyPassOutput{}, ErrNotFound
	}

	if err := pass.ApplyPatch(input.Patch); err != nil {
		var passValidationErr *domain.PassValidationError
		if errors.As(err, &passValidationErr) {
			return ModifyPassOutput{}, &ValidationError{Field: passValidationErr.Field}
		}
		return ModifyPassOutput{}, err
	}

	if isEmptyPassPatch(input.Patch) {
		return ModifyPassOutput{Pass: pass}, nil
	}

	if err := uc.repo.Update(ctx, pass); err != nil {
		return ModifyPassOutput{}, fmt.Errorf("update pass %d: %w", input.ID, err)
	}
	// Re-fetch to surface the post-update updatedAt (set by the DB
	// trigger on every UPDATE). Without this, the response would
	// carry the pre-update updatedAt from the in-memory pass.
	updatedPass, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return ModifyPassOutput{}, fmt.Errorf("get updated pass %d: %w", input.ID, err)
	}
	return ModifyPassOutput{Pass: updatedPass}, nil
}

// isEmptyPassPatch reports whether the patch contains no fields to
// mutate. An empty patch is a no-op and the use case short-circuits
// before touching the database.
func isEmptyPassPatch(patch domain.PassPatch) bool {
	return patch.Price == nil && patch.PassType == nil && patch.ExpiresAt == nil
}
