package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// SetDogNeuteredInput is the validated command to flip the neutered
// flag of a dog. Only NewSetDogNeuteredInput can construct one.
type SetDogNeuteredInput struct {
	id       int
	neutered bool
}

func (in SetDogNeuteredInput) ID() int        { return in.id }
func (in SetDogNeuteredInput) Neutered() bool { return in.neutered }

// NewSetDogNeuteredInput validates id > 0.
func NewSetDogNeuteredInput(id int, neutered bool) (SetDogNeuteredInput, error) {
	if id <= 0 {
		return SetDogNeuteredInput{}, &ValidationError{Field: "id"}
	}
	return SetDogNeuteredInput{id: id, neutered: neutered}, nil
}

// MustNewSetDogNeuteredInput panics on validation error. For tests.
func MustNewSetDogNeuteredInput(id int, neutered bool) SetDogNeuteredInput {
	in, err := NewSetDogNeuteredInput(id, neutered)
	if err != nil {
		panic(err)
	}
	return in
}

// SetDogNeuteredOutput is the post-mutation snapshot of the dog.
type SetDogNeuteredOutput struct {
	ID       int
	Neutered bool
	Sex      domain.Sex
}

// SetDogNeuteredUseCase toggles the neutered flag through the
// aggregate: load the Dog, mutate it via its domain method, then
// persist the whole aggregate.
type SetDogNeuteredUseCase struct {
	repo domain.DogRepository
}

func NewSetDogNeuteredUseCase(repo domain.DogRepository) *SetDogNeuteredUseCase {
	return &SetDogNeuteredUseCase{repo: repo}
}

func (uc *SetDogNeuteredUseCase) Execute(ctx context.Context, input SetDogNeuteredInput) (SetDogNeuteredOutput, error) {
	dog, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return SetDogNeuteredOutput{}, ErrNotFound
		}
		return SetDogNeuteredOutput{}, fmt.Errorf("set dog neutered: %w", err)
	}
	dog.SetNeutered(input.Neutered())
	if err := uc.repo.Update(ctx, dog); err != nil {
		return SetDogNeuteredOutput{}, fmt.Errorf("set dog neutered: %w", err)
	}
	return SetDogNeuteredOutput{
		ID:       dog.ID(),
		Neutered: dog.Neutered(),
		Sex:      dog.Sex(),
	}, nil
}
