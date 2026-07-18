package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type SetDogHeatInput struct {
	ID   int
	Heat bool
}

type SetDogHeatOutput struct {
	ID   int
	Heat bool
	Sex  domain.Sex
}

type SetDogHeatUseCase struct {
	repo domain.DogRepository
}

func NewSetDogHeatUseCase(repo domain.DogRepository) *SetDogHeatUseCase {
	return &SetDogHeatUseCase{repo: repo}
}

func (uc *SetDogHeatUseCase) Execute(ctx context.Context, input SetDogHeatInput) (SetDogHeatOutput, error) {
	if input.ID <= 0 {
		return SetDogHeatOutput{}, &ValidationError{Field: "id"}
	}
	dog, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return SetDogHeatOutput{}, fmt.Errorf("set dog heat: %w", err)
	}
	// Business rule: heat=true is only valid for female dogs. The DB does
	// not enforce this (the column is just a bool) so the use case is
	// the gate. heat=false is always allowed.
	if input.Heat && dog.Sex() != domain.SexFemale {
		return SetDogHeatOutput{}, ErrInvalidHeatForSex
	}
	if err := uc.repo.SetDogHeat(ctx, input.ID, input.Heat); err != nil {
		return SetDogHeatOutput{}, fmt.Errorf("set dog heat: %w", err)
	}
	return SetDogHeatOutput{
		ID:   input.ID,
		Heat: input.Heat,
		Sex:  dog.Sex(),
	}, nil
}

// Compile-time check that ErrInvalidHeatForSex is the kind of error the
// handler can map to 400 via errors.Is.
var _ = errors.Is(ErrInvalidHeatForSex, ErrInvalidHeatForSex)
