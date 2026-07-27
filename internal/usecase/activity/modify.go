package activity

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyActivityInput is the validated command to apply a partial
// update to an existing activity. The patch *values* are validated
// by domain.Activity.ApplyPatch — defense in depth.
type ModifyActivityInput struct {
	id    int
	patch domain.ActivityPatch
}

func (in ModifyActivityInput) ID() int                     { return in.id }
func (in ModifyActivityInput) Patch() domain.ActivityPatch { return in.patch }

// NewModifyActivityInput validates id > 0. Empty patch is allowed
// (Execute short-circuits to a no-op, preserving the current
// behavior).
func NewModifyActivityInput(id int, patch domain.ActivityPatch) (ModifyActivityInput, error) {
	if id <= 0 {
		return ModifyActivityInput{}, &ValidationError{Field: "id"}
	}
	return ModifyActivityInput{id: id, patch: patch}, nil
}

// MustNewModifyActivityInput panics on validation error. For tests.
func MustNewModifyActivityInput(id int, patch domain.ActivityPatch) ModifyActivityInput {
	in, err := NewModifyActivityInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
}

// ModifyActivityOutput carries the post-mutation activity.
type ModifyActivityOutput struct {
	Activity *domain.Activity
}

// ModifyActivityUseCase applies a partial update to an activity.
type ModifyActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewModifyActivityUseCase(repo domain.ActivityRepository) *ModifyActivityUseCase {
	return &ModifyActivityUseCase{repo: repo}
}

func (uc *ModifyActivityUseCase) Execute(ctx context.Context, input ModifyActivityInput) (ModifyActivityOutput, error) {
	activity, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return ModifyActivityOutput{}, fmt.Errorf("get activity %d: %w", input.ID(), err)
	}
	if activity == nil {
		return ModifyActivityOutput{}, ErrNotFound
	}

	patch := input.Patch()
	if err := activity.ApplyPatch(patch); err != nil {
		var validationErr *domain.ActivityValidationError
		if errors.As(err, &validationErr) {
			return ModifyActivityOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyActivityOutput{}, err
	}

	if isEmptyActivityPatch(patch) {
		return ModifyActivityOutput{Activity: activity}, nil
	}

	if err := uc.repo.Update(ctx, activity); err != nil {
		return ModifyActivityOutput{}, fmt.Errorf("update activity %d: %w", input.ID(), err)
	}
	return ModifyActivityOutput{Activity: activity}, nil
}

func isEmptyActivityPatch(patch domain.ActivityPatch) bool {
	return patch.Name == nil &&
		patch.Location == nil &&
		patch.ActivityType == nil &&
		patch.MaxCapacity == nil &&
		patch.DurationInHours == nil &&
		patch.Date == nil
}
