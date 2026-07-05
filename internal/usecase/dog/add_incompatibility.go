package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type AddDogIncompatibilityInput struct {
	DogID           int
	Incompatibility string
}

type AddDogIncompatibilityOutput struct {
	ID                int
	Incompatibilities []domain.Incompatibility
	Added             bool
}

type AddDogIncompatibilityUseCase struct {
	repo domain.DogRepository
}

func NewAddDogIncompatibilityUseCase(repo domain.DogRepository) *AddDogIncompatibilityUseCase {
	return &AddDogIncompatibilityUseCase{repo: repo}
}

func (uc *AddDogIncompatibilityUseCase) Execute(ctx context.Context, in AddDogIncompatibilityInput) (AddDogIncompatibilityOutput, error) {
	if err := in.validate(); err != nil {
		return AddDogIncompatibilityOutput{}, err
	}

	d, err := uc.repo.GetByID(ctx, in.DogID)
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", in.DogID, err)
	}
	if d == nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("dog %d not found", in.DogID)
	}

	target := domain.Incompatibility(in.Incompatibility)
	added := !containsIncompatibility(d.Incompatibilities, target)
	if added {
		d.Incompatibilities = append(d.Incompatibilities, target)
		if err := uc.repo.Update(ctx, d); err != nil {
			return AddDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", in.DogID, err)
		}
	}

	return AddDogIncompatibilityOutput{
		ID:                d.ID,
		Incompatibilities: d.Incompatibilities,
		Added:             added,
	}, nil
}

func (in AddDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(in.DogID, in.Incompatibility)
}
