package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingByUserInput is the input for listing a user's
// upcoming reservations (CONFIRMED and the activity is at or after
// now). Ordered by activity date ASC.
type ListUpcomingByUserInput struct {
	UserID int
	Limit  int
	Offset int
}

// ListUpcomingByUserOutput carries the resulting views, ordered by
// activity date ASC (next class first).
type ListUpcomingByUserOutput struct {
	Views []*domain.ReservationView
}

// ListUpcomingByUserUseCase returns the views of every CONFIRMED
// reservation whose activity is at or after the current time,
// filtered by user (via dog ownership), ordered by activity date
// ASC. limit / offset are normalized by the handler.
//
// The activity-date >= now filter and the CONFIRMED filter are
// enforced by the SQL query (the repository method), not by Go code
// — see ListByUserUpcomingView in the domain interface.
type ListUpcomingByUserUseCase struct {
	repo domain.ReservationRepository
}

func NewListUpcomingByUserUseCase(repo domain.ReservationRepository) *ListUpcomingByUserUseCase {
	return &ListUpcomingByUserUseCase{repo: repo}
}

func (uc *ListUpcomingByUserUseCase) Execute(ctx context.Context, input ListUpcomingByUserInput) (ListUpcomingByUserOutput, error) {
	if input.UserID <= 0 {
		return ListUpcomingByUserOutput{}, &ValidationError{Field: "user_id"}
	}
	views, err := uc.repo.ListByUserUpcomingView(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return ListUpcomingByUserOutput{}, fmt.Errorf("list upcoming reservations for user %d: %w", input.UserID, err)
	}
	return ListUpcomingByUserOutput{Views: views}, nil
}
