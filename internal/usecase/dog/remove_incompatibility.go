package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// RemoveDogIncompatibilityInput is the validated command to detach an
// incompatibility from a dog. Only NewRemoveDogIncompatibilityInput
// can construct one.
type RemoveDogIncompatibilityInput struct {
	dogID             int
	incompatibilityID int
}

func (in RemoveDogIncompatibilityInput) DogID() int             { return in.dogID }
func (in RemoveDogIncompatibilityInput) IncompatibilityID() int { return in.incompatibilityID }

// NewRemoveDogIncompatibilityInput validates both ids > 0.
func NewRemoveDogIncompatibilityInput(dogID, incompatibilityID int) (RemoveDogIncompatibilityInput, error) {
	if err := validateTwoIDs(dogID, incompatibilityID); err != nil {
		return RemoveDogIncompatibilityInput{}, err
	}
	return RemoveDogIncompatibilityInput{dogID: dogID, incompatibilityID: incompatibilityID}, nil
}

// MustNewRemoveDogIncompatibilityInput panics on validation error. For tests.
func MustNewRemoveDogIncompatibilityInput(dogID, incompatibilityID int) RemoveDogIncompatibilityInput {
	in, err := NewRemoveDogIncompatibilityInput(dogID, incompatibilityID)
	if err != nil {
		panic(err)
	}
	return in
}

// RemoveDogIncompatibilityOutput reports the post-mutation state of
// the dog and whether the link was actually removed (false = the dog
// did not have it: idempotent).
type RemoveDogIncompatibilityOutput struct {
	ID                int
	Incompatibilities []domain.Incompatibility
	Removed           bool
}

// RemoveDogIncompatibilityUseCase detaches an incompatibility from a
// dog. Idempotent: a remove of a missing link returns Removed=false
// and skips the DB write. The whole read-modify-write cycle runs
// inside a single transaction with FOR UPDATE on the dog row,
// preventing lost updates (B4).
type RemoveDogIncompatibilityUseCase struct {
	transactor Transactor
	repo       domain.DogRepository
}

func NewRemoveDogIncompatibilityUseCase(transactor Transactor, repo domain.DogRepository) *RemoveDogIncompatibilityUseCase {
	return &RemoveDogIncompatibilityUseCase{transactor: transactor, repo: repo}
}

func (uc *RemoveDogIncompatibilityUseCase) Execute(ctx context.Context, input RemoveDogIncompatibilityInput) (RemoveDogIncompatibilityOutput, error) {
	var out RemoveDogIncompatibilityOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.repo.GetByIDForUpdate(txCtx, input.DogID())
		if err != nil {
			return fmt.Errorf("get dog %d: %w", input.DogID(), err)
		}
		if dog == nil {
			return ErrNotFound
		}

		removed, err := dog.RemoveIncompatibility(input.IncompatibilityID())
		if err != nil {
			return err
		}
		if removed {
			if err := uc.repo.Update(txCtx, dog); err != nil {
				return fmt.Errorf("update dog %d: %w", input.DogID(), err)
			}
		}
		out = RemoveDogIncompatibilityOutput{
			ID:                dog.ID(),
			Incompatibilities: dog.Incompatibilities(),
			Removed:           removed,
		}
		return nil
	})
	return out, err
}
