package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type ModifyDogInput struct {
	ID            int
	Neutered      bool
	Heat          bool
	WeightKg      float64
	PhotoURL      string
	MedicalNotes  string
	EducatorNotes string
	IsActive      bool
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
	if err := in.validate(); err != nil {
		return ModifyDogOutput{}, err
	}

	d, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		return ModifyDogOutput{}, fmt.Errorf("get dog %d: %w", in.ID, err)
	}
	if d == nil {
		return ModifyDogOutput{}, fmt.Errorf("dog %d not found", in.ID)
	}

	d.Neutered = in.Neutered
	d.Heat = in.Heat
	d.WeightKg = in.WeightKg
	d.PhotoURL = in.PhotoURL
	d.MedicalNotes = in.MedicalNotes
	d.EducatorNotes = in.EducatorNotes
	d.IsActive = in.IsActive

	if err := uc.repo.Update(ctx, d); err != nil {
		return ModifyDogOutput{}, fmt.Errorf("update dog %d: %w", in.ID, err)
	}

	return ModifyDogOutput{ID: d.ID}, nil
}

func (in ModifyDogInput) validate() error {
	if in.ID <= 0 {
		return &ValidationError{Field: "id"}
	}
	if in.WeightKg <= 0 {
		return &ValidationError{Field: "weight_kg"}
	}
	return nil
}
