package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type ModifyDogInput struct {
	ID    int
	Patch domain.DogPatch
}

type ModifyDogOutput struct {
	ID int
}

type ModifyDogUseCase struct {
	repo domain.DogRepository
}

func NewModifyDogUseCase(repo domain.DogRepository) *ModifyDogUseCase {
	return &ModifyDogUseCase{repo: repo}
}

func (uc *ModifyDogUseCase) Execute(ctx context.Context, input ModifyDogInput) (ModifyDogOutput, error) {
	if input.ID <= 0 {
		return ModifyDogOutput{}, &ValidationError{Field: "id"}
	}

	dog, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return ModifyDogOutput{}, fmt.Errorf("get dog %d: %w", input.ID, err)
	}
	if dog == nil {
		return ModifyDogOutput{}, ErrNotFound
	}

	if err := dog.ApplyPatch(input.Patch); err != nil {
		var validationErr *domain.DogValidationError
		if errors.As(err, &validationErr) {
			return ModifyDogOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyDogOutput{}, err
	}

	if isEmptyPatch(input.Patch) {
		return ModifyDogOutput{ID: dog.ID()}, nil
	}

	if err := uc.repo.Update(ctx, dog); err != nil {
		return ModifyDogOutput{}, fmt.Errorf("update dog %d: %w", input.ID, err)
	}

	return ModifyDogOutput{ID: dog.ID()}, nil
}

func isEmptyPatch(patch domain.DogPatch) bool {
	return patch.Name == nil && patch.Breed == nil && patch.AgeInMonths == nil &&
		patch.Sex == nil && patch.Passport == nil && patch.WeightKg == nil &&
		patch.Neutered == nil && patch.Heat == nil && patch.PhotoURL == nil &&
		patch.MedicalNotes == nil && patch.EducatorNotes == nil && patch.IsActive == nil
}
