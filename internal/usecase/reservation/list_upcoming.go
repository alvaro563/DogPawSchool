package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingByUserInput is the validated input for listing a
// user's upcoming reservations.
type ListUpcomingByUserInput struct {
	userID int
	limit  int
	offset int
}

func (in ListUpcomingByUserInput) UserID() int { return in.userID }
func (in ListUpcomingByUserInput) Limit() int  { return in.limit }
func (in ListUpcomingByUserInput) Offset() int { return in.offset }

// NewListUpcomingByUserInput validates user id and normalizes pagination.
func NewListUpcomingByUserInput(userID, limit, offset int) (ListUpcomingByUserInput, error) {
	if userID <= 0 {
		return ListUpcomingByUserInput{}, &ValidationError{Field: "user_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListUpcomingByUserInput{userID: userID, limit: limit, offset: offset}, nil
}

// MustNewListUpcomingByUserInput panics on validation error. For tests.
func MustNewListUpcomingByUserInput(userID, limit, offset int) ListUpcomingByUserInput {
	in, err := NewListUpcomingByUserInput(userID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListUpcomingByUserOutput carries the resulting views, ordered by
// activity date ASC (next class first).
type ListUpcomingByUserOutput struct {
	Views []*domain.ReservationView
}

// ListUpcomingByUserUseCase returns the views of every CONFIRMED
// reservation whose activity is at or after the current time,
// filtered by user (via dog ownership), ordered by activity date
// ASC.
type ListUpcomingByUserUseCase struct {
	repo domain.ReservationRepository
}

func NewListUpcomingByUserUseCase(repo domain.ReservationRepository) *ListUpcomingByUserUseCase {
	return &ListUpcomingByUserUseCase{repo: repo}
}

func (uc *ListUpcomingByUserUseCase) Execute(ctx context.Context, input ListUpcomingByUserInput) (ListUpcomingByUserOutput, error) {
	views, err := uc.repo.ListByUserUpcomingView(ctx, input.UserID(), input.Limit(), input.Offset())
	if err != nil {
		return ListUpcomingByUserOutput{}, fmt.Errorf("list upcoming reservations for user %d: %w", input.UserID(), err)
	}
	return ListUpcomingByUserOutput{Views: views}, nil
}
