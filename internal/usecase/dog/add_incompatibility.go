package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// AddDogIncompatibilityInput is the validated command to attach an
// incompatibility to a dog. Only NewAddDogIncompatibilityInput can
// construct one.
type AddDogIncompatibilityInput struct {
	dogID             int
	incompatibilityID int
}

func (in AddDogIncompatibilityInput) DogID() int             { return in.dogID }
func (in AddDogIncompatibilityInput) IncompatibilityID() int { return in.incompatibilityID }

// NewAddDogIncompatibilityInput validates both ids > 0.
func NewAddDogIncompatibilityInput(dogID, incompatibilityID int) (AddDogIncompatibilityInput, error) {
	if err := validateTwoIDs(dogID, incompatibilityID); err != nil {
		return AddDogIncompatibilityInput{}, err
	}
	return AddDogIncompatibilityInput{dogID: dogID, incompatibilityID: incompatibilityID}, nil
}

// MustNewAddDogIncompatibilityInput panics on validation error. For tests.
func MustNewAddDogIncompatibilityInput(dogID, incompatibilityID int) AddDogIncompatibilityInput {
	in, err := NewAddDogIncompatibilityInput(dogID, incompatibilityID)
	if err != nil {
		panic(err)
	}
	return in
}

// AddDogIncompatibilityOutput reports the post-mutation state of the
// dog (id, full traits and incompatibility lists) and whether the new
// link was freshly added (false means the dog already had it:
// idempotent).
type AddDogIncompatibilityOutput struct {
	ID                int
	Incompatibilities []domain.Incompatibility
	Traits            []domain.Incompatibility
	Added             bool
}

// AddDogIncompatibilityUseCase attaches an incompatibility to a dog.
// Idempotent: a duplicate add returns Added=false and skips the DB
// write. The whole read-modify-write cycle runs inside a single
// transaction with FOR UPDATE on the dog row, preventing lost
// updates (B4).
type AddDogIncompatibilityUseCase struct {
	transactor   Transactor
	dogRepo      domain.DogRepository
	incompatRepo domain.IncompatibilityRepository
}

func NewAddDogIncompatibilityUseCase(
	transactor Transactor,
	dogRepo domain.DogRepository,
	incompatRepo domain.IncompatibilityRepository,
) *AddDogIncompatibilityUseCase {
	return &AddDogIncompatibilityUseCase{transactor: transactor, dogRepo: dogRepo, incompatRepo: incompatRepo}
}

func (uc *AddDogIncompatibilityUseCase) Execute(ctx context.Context, input AddDogIncompatibilityInput) (AddDogIncompatibilityOutput, error) {
	incompat, err := uc.incompatRepo.GetIncompatibilityByID(ctx, input.IncompatibilityID())
	if err != nil {
		return AddDogIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.IncompatibilityID(), err)
	}
	if incompat == nil {
		return AddDogIncompatibilityOutput{}, ErrNotFound
	}

	var out AddDogIncompatibilityOutput
	err = uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.dogRepo.GetByIDForUpdate(txCtx, input.DogID())
		if err != nil {
			return fmt.Errorf("get dog %d: %w", input.DogID(), err)
		}
		if dog == nil {
			return ErrNotFound
		}

		var added bool
		if incompat.IsTrait() {
			added, err = dog.AddTrait(incompat)
		} else {
			added, err = dog.AddIncompatibility(incompat)
		}
		if err != nil {
			return err
		}
		if added {
			if err := uc.dogRepo.Update(txCtx, dog); err != nil {
				return fmt.Errorf("update dog %d: %w", input.DogID(), err)
			}
		}
		out = AddDogIncompatibilityOutput{
			ID:                dog.ID(),
			Incompatibilities: dog.Incompatibilities(),
			Traits:            dog.Traits(),
			Added:             added,
		}
		return nil
	})
	return out, err
}
