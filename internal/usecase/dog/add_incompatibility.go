package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type AddDogIncompatibilityInput struct {
	DogID             int
	IncompatibilityID int
}

type AddDogIncompatibilityOutput struct {
	ID                int
	Incompatibilities []domain.Incompatibility
	Added             bool
}

type AddDogIncompatibilityUseCase struct {
	dogRepo      domain.DogRepository
	incompatRepo domain.IncompatibilityRepository
}

func NewAddDogIncompatibilityUseCase(
	dogRepo domain.DogRepository,
	incompatRepo domain.IncompatibilityRepository,
) *AddDogIncompatibilityUseCase {
	return &AddDogIncompatibilityUseCase{dogRepo: dogRepo, incompatRepo: incompatRepo}
}

func (uc *AddDogIncompatibilityUseCase) Execute(ctx context.Context, input AddDogIncompatibilityInput) (AddDogIncompatibilityOutput, error) {
	if err := input.validate(); err != nil {
		return AddDogIncompatibilityOutput{}, err
	}

	incompat, err := uc.incompatRepo.GetIncompatibilityByID(ctx, input.IncompatibilityID)
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.IncompatibilityID, err)
	}
	if incompat == nil {
		return AddDogIncompatibilityOutput{}, ErrNotFound
	}

	dog, err := uc.dogRepo.GetByID(ctx, input.DogID)
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", input.DogID, err)
	}
	if dog == nil {
		return AddDogIncompatibilityOutput{}, ErrNotFound
	}

	added, err := dog.AddIncompatibility(incompat)
	if err != nil {
		return AddDogIncompatibilityOutput{}, err
	}
	if added {
		if err := uc.dogRepo.Update(ctx, dog); err != nil {
			return AddDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", input.DogID, err)
		}
	}

	return AddDogIncompatibilityOutput{
		ID:                dog.ID(),
		Incompatibilities: dog.Incompatibilities(),
		Added:             added,
	}, nil
}

func (input AddDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(input.DogID, input.IncompatibilityID)
}
