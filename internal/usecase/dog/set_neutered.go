package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type SetDogNeuteredInput struct {
	ID       int
	Neutered bool
}

type SetDogNeuteredOutput struct {
	ID       int
	Neutered bool
	Sex      domain.Sex
}

type SetDogNeuteredUseCase struct {
	repo domain.DogRepository
}

func NewSetDogNeuteredUseCase(repo domain.DogRepository) *SetDogNeuteredUseCase {
	return &SetDogNeuteredUseCase{repo: repo}
}

func (uc *SetDogNeuteredUseCase) Execute(ctx context.Context, input SetDogNeuteredInput) (SetDogNeuteredOutput, error) {
	if input.ID <= 0 {
		return SetDogNeuteredOutput{}, &ValidationError{Field: "id"}
	}
	dog, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return SetDogNeuteredOutput{}, fmt.Errorf("set dog neutered: %w", err)
	}
	if err := uc.repo.SetDogNeutered(ctx, input.ID, input.Neutered); err != nil {
		return SetDogNeuteredOutput{}, fmt.Errorf("set dog neutered: %w", err)
	}
	return SetDogNeuteredOutput{
		ID:       input.ID,
		Neutered: input.Neutered,
		Sex:      dog.Sex(),
	}, nil
}
