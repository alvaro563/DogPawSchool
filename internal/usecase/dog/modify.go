package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyDogInput is the validated command to apply a partial update
// to an existing dog. Fields are private: only NewModifyDogInput
// can construct one. The patch *values* are validated by the domain
// (domain.Dog.ApplyPatch) — defense in depth.
type ModifyDogInput struct {
	id    int
	patch domain.DogPatch
}

func (in ModifyDogInput) ID() int                { return in.id }
func (in ModifyDogInput) Patch() domain.DogPatch { return in.patch }

// NewModifyDogInput validates id > 0 and stores the patch. An empty
// patch is allowed (Execute short-circuits to a no-op, preserving
// the current behavior).
func NewModifyDogInput(id int, patch domain.DogPatch) (ModifyDogInput, error) {
	if id <= 0 {
		return ModifyDogInput{}, &ValidationError{Field: "id"}
	}
	return ModifyDogInput{id: id, patch: patch}, nil
}

// MustNewModifyDogInput panics on validation error. For tests.
func MustNewModifyDogInput(id int, patch domain.DogPatch) ModifyDogInput {
	in, err := NewModifyDogInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
}

// ModifyDogOutput carries the post-mutation dog id.
type ModifyDogOutput struct {
	ID int
}

// ModifyDogUseCase applies a partial update to a dog. An empty patch
// is a no-op and returns the unmodified dog without touching the DB.
type ModifyDogUseCase struct {
	repo domain.DogRepository
}

func NewModifyDogUseCase(repo domain.DogRepository) *ModifyDogUseCase {
	return &ModifyDogUseCase{repo: repo}
}

func (uc *ModifyDogUseCase) Execute(ctx context.Context, input ModifyDogInput) (ModifyDogOutput, error) {
	dog, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return ModifyDogOutput{}, fmt.Errorf("get dog %d: %w", input.ID(), err)
	}
	if dog == nil {
		return ModifyDogOutput{}, ErrNotFound
	}

	patch := input.Patch()
	if err := dog.ApplyPatch(patch); err != nil {
		var validationErr *domain.DogValidationError
		if errors.As(err, &validationErr) {
			return ModifyDogOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyDogOutput{}, err
	}

	if isEmptyPatch(patch) {
		return ModifyDogOutput{ID: dog.ID()}, nil
	}

	if err := uc.repo.Update(ctx, dog); err != nil {
		return ModifyDogOutput{}, fmt.Errorf("update dog %d: %w", input.ID(), err)
	}
	return ModifyDogOutput{ID: dog.ID()}, nil
}

func isEmptyPatch(patch domain.DogPatch) bool {
	return patch.Name == nil && patch.Breed == nil && patch.AgeInMonths == nil &&
		patch.Sex == nil && patch.Passport == nil && patch.WeightKg == nil &&
		patch.Neutered == nil && patch.Heat == nil && patch.PhotoURL == nil &&
		patch.MedicalNotes == nil && patch.EducatorNotes == nil && patch.IsActive == nil
}
