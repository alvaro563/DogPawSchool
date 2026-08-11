package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingAllInput is the validated input for listing every
// upcoming CONFIRMED reservation in the system.
type ListUpcomingAllInput struct {
	limit  int
	offset int
}

func (in ListUpcomingAllInput) Limit() int  { return in.limit }
func (in ListUpcomingAllInput) Offset() int { return in.offset }

// NewListUpcomingAllInput normalizes pagination. Error is always nil
// for pure-pagination inputs.
func NewListUpcomingAllInput(limit, offset int) (ListUpcomingAllInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListUpcomingAllInput{limit: limit, offset: offset}, nil
}

// MustNewListUpcomingAllInput panics on error. For tests.
func MustNewListUpcomingAllInput(limit, offset int) ListUpcomingAllInput {
	in, err := NewListUpcomingAllInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListUpcomingAllOutput carries the resulting views, ordered by
// activity date ASC (next class first).
type ListUpcomingAllOutput struct {
	Views []*domain.ReservationView
}

// ListUpcomingAllUseCase returns the views of every CONFIRMED
// reservation whose activity is at or after the current time,
// across all users, ordered by activity date ASC.
type ListUpcomingAllUseCase struct {
	repo domain.ReservationRepository
}

func NewListUpcomingAllUseCase(repo domain.ReservationRepository) *ListUpcomingAllUseCase {
	return &ListUpcomingAllUseCase{repo: repo}
}

func (uc *ListUpcomingAllUseCase) Execute(ctx context.Context, input ListUpcomingAllInput) (ListUpcomingAllOutput, error) {
	views, err := uc.repo.ListAllUpcomingView(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListUpcomingAllOutput{}, fmt.Errorf("list upcoming all reservations: %w", err)
	}
	return ListUpcomingAllOutput{Views: views}, nil
}
