package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type AddDogTraitInput struct {
	dogID  int
	traitID int
}

func (in AddDogTraitInput) DogID() int   { return in.dogID }
func (in AddDogTraitInput) TraitID() int { return in.traitID }

func NewAddDogTraitInput(dogID, traitID int) (AddDogTraitInput, error) {
	if err := validateTwoIDs(dogID, traitID); err != nil {
		return AddDogTraitInput{}, err
	}
	return AddDogTraitInput{dogID: dogID, traitID: traitID}, nil
}

func MustNewAddDogTraitInput(dogID, traitID int) AddDogTraitInput {
	in, err := NewAddDogTraitInput(dogID, traitID)
	if err != nil {
		panic(err)
	}
	return in
}

type AddDogTraitOutput struct {
	ID     int
	Traits []domain.Incompatibility
	Added  bool
}

type AddDogTraitUseCase struct {
	transactor   Transactor
	dogRepo      domain.DogRepository
	incompatRepo domain.IncompatibilityRepository
}

func NewAddDogTraitUseCase(
	transactor Transactor,
	dogRepo domain.DogRepository,
	incompatRepo domain.IncompatibilityRepository,
) *AddDogTraitUseCase {
	return &AddDogTraitUseCase{transactor: transactor, dogRepo: dogRepo, incompatRepo: incompatRepo}
}

func (uc *AddDogTraitUseCase) Execute(ctx context.Context, input AddDogTraitInput) (AddDogTraitOutput, error) {
	trait, err := uc.incompatRepo.GetIncompatibilityByID(ctx, input.TraitID())
	if err != nil {
		return AddDogTraitOutput{}, fmt.Errorf("get trait %d: %w", input.TraitID(), err)
	}
	if trait == nil || trait.Code() == "" {
		return AddDogTraitOutput{}, ErrNotATrait
	}

	var out AddDogTraitOutput
	err = uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.dogRepo.GetByIDForUpdate(txCtx, input.DogID())
		if err != nil {
			return fmt.Errorf("get dog %d: %w", input.DogID(), err)
		}
		if dog == nil {
			return ErrNotFound
		}

		added, err := dog.AddTrait(trait)
		if err != nil {
			return err
		}
		if added {
			if err := uc.dogRepo.Update(txCtx, dog); err != nil {
				return fmt.Errorf("update dog %d: %w", input.DogID(), err)
			}
		}
		out = AddDogTraitOutput{
			ID:     dog.ID(),
			Traits: dog.Traits(),
			Added:  added,
		}
		return nil
	})
	return out, err
}
