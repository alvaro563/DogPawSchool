package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type AddDogTriggerInput struct {
	dogID     int
	triggerID int
}

func (in AddDogTriggerInput) DogID() int     { return in.dogID }
func (in AddDogTriggerInput) TriggerID() int { return in.triggerID }

func NewAddDogTriggerInput(dogID, triggerID int) (AddDogTriggerInput, error) {
	if err := validateTwoIDs(dogID, triggerID); err != nil {
		return AddDogTriggerInput{}, err
	}
	return AddDogTriggerInput{dogID: dogID, triggerID: triggerID}, nil
}

func MustNewAddDogTriggerInput(dogID, triggerID int) AddDogTriggerInput {
	in, err := NewAddDogTriggerInput(dogID, triggerID)
	if err != nil {
		panic(err)
	}
	return in
}

type AddDogTriggerOutput struct {
	ID               int
	Incompatibilities []domain.Incompatibility
	Added            bool
}

type AddDogTriggerUseCase struct {
	transactor   Transactor
	dogRepo      domain.DogRepository
	incompatRepo domain.IncompatibilityRepository
}

func NewAddDogTriggerUseCase(
	transactor Transactor,
	dogRepo domain.DogRepository,
	incompatRepo domain.IncompatibilityRepository,
) *AddDogTriggerUseCase {
	return &AddDogTriggerUseCase{transactor: transactor, dogRepo: dogRepo, incompatRepo: incompatRepo}
}

func (uc *AddDogTriggerUseCase) Execute(ctx context.Context, input AddDogTriggerInput) (AddDogTriggerOutput, error) {
	trigger, err := uc.incompatRepo.GetIncompatibilityByID(ctx, input.TriggerID())
	if err != nil {
		return AddDogTriggerOutput{}, fmt.Errorf("get trigger %d: %w", input.TriggerID(), err)
	}
	if trigger == nil || trigger.TargetTraitCode() == "" {
		return AddDogTriggerOutput{}, ErrNotATrigger
	}

	var out AddDogTriggerOutput
	err = uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.dogRepo.GetByIDForUpdate(txCtx, input.DogID())
		if err != nil {
			return fmt.Errorf("get dog %d: %w", input.DogID(), err)
		}
		if dog == nil {
			return ErrNotFound
		}

		added, err := dog.AddIncompatibility(trigger)
		if err != nil {
			return err
		}
		if added {
			if err := uc.dogRepo.Update(txCtx, dog); err != nil {
				return fmt.Errorf("update dog %d: %w", input.DogID(), err)
			}
		}
		out = AddDogTriggerOutput{
			ID:                dog.ID(),
			Incompatibilities: dog.Incompatibilities(),
			Added:             added,
		}
		return nil
	})
	return out, err
}
