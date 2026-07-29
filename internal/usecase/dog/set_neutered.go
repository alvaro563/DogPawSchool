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
// persist the whole aggregate. Runs inside a single transaction with
// FOR UPDATE on the dog row, preventing lost updates (B4).
type SetDogNeuteredUseCase struct {
	transactor Transactor
	repo       domain.DogRepository
}

func NewSetDogNeuteredUseCase(transactor Transactor, repo domain.DogRepository) *SetDogNeuteredUseCase {
	return &SetDogNeuteredUseCase{transactor: transactor, repo: repo}
}

func (uc *SetDogNeuteredUseCase) Execute(ctx context.Context, input SetDogNeuteredInput) (SetDogNeuteredOutput, error) {
	var out SetDogNeuteredOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.repo.GetByIDForUpdate(txCtx, input.ID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("set dog neutered: %w", err)
		}

		dog.SetNeutered(input.Neutered())
		if err := uc.repo.Update(txCtx, dog); err != nil {
			return fmt.Errorf("set dog neutered: %w", err)
		}
		out = SetDogNeuteredOutput{
			ID:       dog.ID(),
			Neutered: dog.Neutered(),
			Sex:      dog.Sex(),
		}
		return nil
	})
	return out, err
}
