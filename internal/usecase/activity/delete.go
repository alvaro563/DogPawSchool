package activity

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// DeleteActivityInput is the validated command to delete an activity.
// Only NewDeleteActivityInput can construct one.
type DeleteActivityInput struct {
	id int
}

func (in DeleteActivityInput) ID() int { return in.id }

// NewDeleteActivityInput validates id > 0.
func NewDeleteActivityInput(id int) (DeleteActivityInput, error) {
	if id <= 0 {
		return DeleteActivityInput{}, &ValidationError{Field: "id"}
	}
	return DeleteActivityInput{id: id}, nil
}

// MustNewDeleteActivityInput panics on validation error. For tests.
func MustNewDeleteActivityInput(id int) DeleteActivityInput {
	in, err := NewDeleteActivityInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// DeleteActivityOutput is empty: a successful delete returns no payload.
type DeleteActivityOutput struct{}

// DeleteActivityUseCase removes an activity by id. The repository
// refuses while the activity holds CONFIRMED / PENDING_TO_CONFIRM
// reservations (mapped here to ErrActivityHasReservations → 409);
// reservations in terminal states cascade away with the row.
type DeleteActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewDeleteActivityUseCase(repo domain.ActivityRepository) *DeleteActivityUseCase {
	return &DeleteActivityUseCase{repo: repo}
}

func (uc *DeleteActivityUseCase) Execute(ctx context.Context, input DeleteActivityInput) (DeleteActivityOutput, error) {
	if err := uc.repo.Delete(ctx, input.ID()); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return DeleteActivityOutput{}, ErrNotFound
		}
		if errors.Is(err, domain.ErrActivityHasReservations) {
			return DeleteActivityOutput{}, ErrActivityHasReservations
		}
		return DeleteActivityOutput{}, fmt.Errorf("delete activity: %w", err)
	}
	return DeleteActivityOutput{}, nil
}
