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

func (uc *ModifyDogUseCase) Execute(ctx context.Context, in ModifyDogInput) (ModifyDogOutput, error) {
	if in.ID <= 0 {
		return ModifyDogOutput{}, &ValidationError{Field: "id"}
	}

	d, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		return ModifyDogOutput{}, fmt.Errorf("get dog %d: %w", in.ID, err)
	}
	if d == nil {
		return ModifyDogOutput{}, ErrNotFound
	}

	if err := d.ApplyPatch(in.Patch); err != nil {
		var dverr *domain.DogValidationError
		if errors.As(err, &dverr) {
			return ModifyDogOutput{}, &ValidationError{Field: dverr.Field}
		}
		return ModifyDogOutput{}, err
	}

	if isEmptyPatch(in.Patch) {
		return ModifyDogOutput{ID: d.ID()}, nil
	}

	if err := uc.repo.Update(ctx, d); err != nil {
		return ModifyDogOutput{}, fmt.Errorf("update dog %d: %w", in.ID, err)
	}

	return ModifyDogOutput{ID: d.ID()}, nil
}

func isEmptyPatch(p domain.DogPatch) bool {
	return p.Name == nil && p.Breed == nil && p.AgeInMonths == nil &&
		p.Sex == nil && p.Passport == nil && p.WeightKg == nil &&
		p.Neutered == nil && p.Heat == nil && p.PhotoURL == nil &&
		p.MedicalNotes == nil && p.EducatorNotes == nil && p.IsActive == nil
}
