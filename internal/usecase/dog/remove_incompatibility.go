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

func (uc *RemoveDogIncompatibilityUseCase) Execute(ctx context.Context, input RemoveDogIncompatibilityInput) (RemoveDogIncompatibilityOutput, error) {
	if err := input.validate(); err != nil {
		return RemoveDogIncompatibilityOutput{}, err
	}

	dog, err := uc.repo.GetByID(ctx, input.DogID)
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", input.DogID, err)
	}
	if dog == nil {
		return RemoveDogIncompatibilityOutput{}, ErrNotFound
	}

	removed, err := dog.RemoveIncompatibility(input.IncompatibilityID)
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, err
	}
	if removed {
		if err := uc.repo.Update(ctx, dog); err != nil {
			return RemoveDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", input.DogID, err)
		}
	}

	return RemoveDogIncompatibilityOutput{
		ID:                dog.ID(),
		Incompatibilities: dog.Incompatibilities(),
		Removed:           removed,
	}, nil
}

func (input RemoveDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(input.DogID, input.IncompatibilityID)
}
