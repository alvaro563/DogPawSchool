package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type RemoveDogIncompatibilityInput struct {
	DogID           int
	Incompatibility string
}

type RemoveDogIncompatibilityOutput struct {
	ID                int
	Incompatibilities []domain.Incompatibility
	Removed           bool
}

type RemoveDogIncompatibilityUseCase struct {
	repo domain.DogRepository
}

func NewRemoveDogIncompatibilityUseCase(repo domain.DogRepository) *RemoveDogIncompatibilityUseCase {
	return &RemoveDogIncompatibilityUseCase{repo: repo}
}

func (uc *RemoveDogIncompatibilityUseCase) Execute(ctx context.Context, in RemoveDogIncompatibilityInput) (RemoveDogIncompatibilityOutput, error) {
	if err := in.validate(); err != nil {
		return RemoveDogIncompatibilityOutput{}, err
	}

	d, err := uc.repo.GetByID(ctx, in.DogID)
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", in.DogID, err)
	}
	if d == nil {
		return RemoveDogIncompatibilityOutput{}, fmt.Errorf("dog %d not found", in.DogID)
	}

	target := domain.Incompatibility(in.Incompatibility)
	removed := containsIncompatibility(d.Incompatibilities, target)
	if removed {
		d.Incompatibilities = removeIncompatibility(d.Incompatibilities, target)
		if err := uc.repo.Update(ctx, d); err != nil {
			return RemoveDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", in.DogID, err)
		}
	}

	return RemoveDogIncompatibilityOutput{
		ID:                d.ID,
		Incompatibilities: d.Incompatibilities,
		Removed:           removed,
	}, nil
}

func (in RemoveDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(in.DogID, in.Incompatibility)
}
