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

func (uc *AddDogIncompatibilityUseCase) Execute(ctx context.Context, in AddDogIncompatibilityInput) (AddDogIncompatibilityOutput, error) {
	if err := in.validate(); err != nil {
		return AddDogIncompatibilityOutput{}, err
	}

	incompat, err := uc.incompatRepo.GetIncompatibilityByID(ctx, in.IncompatibilityID)
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", in.IncompatibilityID, err)
	}
	if incompat == nil {
		return AddDogIncompatibilityOutput{}, ErrNotFound
	}

	d, err := uc.dogRepo.GetByID(ctx, in.DogID)
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", in.DogID, err)
	}
	if d == nil {
		return AddDogIncompatibilityOutput{}, ErrNotFound
	}

	added, err := d.AddIncompatibility(incompat)
	if err != nil {
		return AddDogIncompatibilityOutput{}, err
	}
	if added {
		if err := uc.dogRepo.Update(ctx, d); err != nil {
			return AddDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", in.DogID, err)
		}
	}

	return AddDogIncompatibilityOutput{
		ID:                d.ID(),
		Incompatibilities: d.Incompatibilities(),
		Added:             added,
	}, nil
}

func (in AddDogIncompatibilityInput) validate() error {
	return validateIncompatibilityInput(in.DogID, in.IncompatibilityID)
}
