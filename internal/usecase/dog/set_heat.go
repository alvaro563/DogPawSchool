package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// SetDogHeatInput is the validated command to flip the heat flag of a
// dog. Only NewSetDogHeatInput can construct one.
type SetDogHeatInput struct {
	id   int
	heat bool
}

func (in SetDogHeatInput) ID() int    { return in.id }
func (in SetDogHeatInput) Heat() bool { return in.heat }

// NewSetDogHeatInput validates id > 0. The "heat only on female dogs"
// invariant is enforced inside domain.Dog.SetHeat, not here.
func NewSetDogHeatInput(id int, heat bool) (SetDogHeatInput, error) {
	if id <= 0 {
		return SetDogHeatInput{}, &ValidationError{Field: "id"}
	}
	return SetDogHeatInput{id: id, heat: heat}, nil
}

// MustNewSetDogHeatInput panics on validation error. For tests.
func MustNewSetDogHeatInput(id int, heat bool) SetDogHeatInput {
	in, err := NewSetDogHeatInput(id, heat)
	if err != nil {
		panic(err)
	}
	return in
}

// SetDogHeatOutput is the post-mutation snapshot of the dog.
type SetDogHeatOutput struct {
	ID   int
	Heat bool
	Sex  domain.Sex
}

// SetDogHeatUseCase toggles the heat flag through the aggregate. The
// "heat only on female dogs" invariant lives in the entity
// (dog.SetHeat); a DogValidationError from the entity is translated
// into the use-case-facing ErrInvalidHeatForSex sentinel. Runs inside
// a single transaction with FOR UPDATE on the dog row, preventing
// lost updates (B4).
type SetDogHeatUseCase struct {
	transactor Transactor
	repo       domain.DogRepository
}

func NewSetDogHeatUseCase(transactor Transactor, repo domain.DogRepository) *SetDogHeatUseCase {
	return &SetDogHeatUseCase{transactor: transactor, repo: repo}
}

func (uc *SetDogHeatUseCase) Execute(ctx context.Context, input SetDogHeatInput) (SetDogHeatOutput, error) {
	var out SetDogHeatOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.repo.GetByIDForUpdate(txCtx, input.ID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("set dog heat: %w", err)
		}

		if err := dog.SetHeat(input.Heat()); err != nil {
			var validationErr *domain.DogValidationError
			if errors.As(err, &validationErr) && validationErr.Field == "heat" {
				return ErrInvalidHeatForSex
			}
			return fmt.Errorf("set dog heat: %w", err)
		}

		if err := uc.repo.Update(txCtx, dog); err != nil {
			return fmt.Errorf("set dog heat: %w", err)
		}
		out = SetDogHeatOutput{
			ID:   dog.ID(),
			Heat: dog.Heat(),
			Sex:  dog.Sex(),
		}
		return nil
	})
	return out, err
}
