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
// and skips the DB write.
type RemoveDogIncompatibilityUseCase struct {
	repo domain.DogRepository
}

func NewRemoveDogIncompatibilityUseCase(repo domain.DogRepository) *RemoveDogIncompatibilityUseCase {
	return &RemoveDogIncompatibilityUseCase{repo: repo}
}

func (uc *RemoveDogIncompatibilityUseCase) Execute(ctx context.Context, input RemoveDogIncompatibilityInput) (RemoveDogIncompatibilityOutput, error) {
	dog, err := uc.repo.GetByID(ctx, input.DogID())
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, fmt.Errorf("get dog %d: %w", input.DogID(), err)
	}
	if dog == nil {
		return RemoveDogIncompatibilityOutput{}, ErrNotFound
	}

	removed, err := dog.RemoveIncompatibility(input.IncompatibilityID())
	if err != nil {
		return RemoveDogIncompatibilityOutput{}, err
	}
	if removed {
		if err := uc.repo.Update(ctx, dog); err != nil {
			return RemoveDogIncompatibilityOutput{}, fmt.Errorf("update dog %d: %w", input.DogID(), err)
		}
	}
	return RemoveDogIncompatibilityOutput{
		ID:                dog.ID(),
		Incompatibilities: dog.Incompatibilities(),
		Removed:           removed,
	}, nil
}
