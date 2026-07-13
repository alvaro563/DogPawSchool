package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type RemoveDogIncompatibilityInput struct {
	DogID             int
	IncompatibilityID int
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
		return RemoveDogIncompatibilityOutput{}, ErrNotFound
	}

	removed, err := d.RemoveIncompatibility(in.IncompatibilityID)
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, err
	}
	if removed {
		if err := uc.repo.Update(ctx, d); err != nil {
			return RemoveDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", in.DogID, err)
		}
	}

	return RemoveDogIncompatibilityOutput{
		ID:                d.ID(),
		Incompatibilities: d.Incompatibilities(),
		Removed:           removed,
	}, nil
}

func (in RemoveDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(in.DogID, in.IncompatibilityID)
}
