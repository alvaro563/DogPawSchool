package activity

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyActivityInput is the input for a partial update of an
// existing activity.
type ModifyActivityInput struct {
	ID    int
	Patch domain.ActivityPatch
}

// ModifyActivityOutput carries the post-mutation activity. The full
// domain object is returned so the handler can serialize it
// directly.
type ModifyActivityOutput struct {
	Activity *domain.Activity
}

// ModifyActivityUseCase applies a partial update to an activity. An
// empty patch is a no-op and returns the unmodified activity.
type ModifyActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewModifyActivityUseCase(repo domain.ActivityRepository) *ModifyActivityUseCase {
	return &ModifyActivityUseCase{repo: repo}
}

func (uc *ModifyActivityUseCase) Execute(ctx context.Context, input ModifyActivityInput) (ModifyActivityOutput, error) {
	if input.ID <= 0 {
		return ModifyActivityOutput{}, &ValidationError{Field: "id"}
	}

	activity, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return ModifyActivityOutput{}, fmt.Errorf("get activity %d: %w", input.ID, err)
	}
	if activity == nil {
		return ModifyActivityOutput{}, ErrNotFound
	}

	if err := activity.ApplyPatch(input.Patch); err != nil {
		var validationErr *domain.ActivityValidationError
		if errors.As(err, &validationErr) {
			return ModifyActivityOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyActivityOutput{}, err
	}

	if isEmptyActivityPatch(input.Patch) {
		return ModifyActivityOutput{Activity: activity}, nil
	}

	if err := uc.repo.Update(ctx, activity); err != nil {
		return ModifyActivityOutput{}, fmt.Errorf("update activity %d: %w", input.ID, err)
	}
	return ModifyActivityOutput{Activity: activity}, nil
}

// isEmptyActivityPatch reports whether the patch contains no fields
// to mutate. An empty patch is a no-op and the use case short-circuits
// before touching the database.
func isEmptyActivityPatch(patch domain.ActivityPatch) bool {
	return patch.Name == nil &&
		patch.Location == nil &&
		patch.ActivityType == nil &&
		patch.MaxCapacity == nil &&
		patch.DurationInHours == nil &&
		patch.Date == nil
}
