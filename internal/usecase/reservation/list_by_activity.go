package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByActivityReservationsInput is the validated input for listing
// every reservation for a specific activity (class roster view).
type ListByActivityReservationsInput struct {
	activityID int
	limit      int
	offset     int
}

func (in ListByActivityReservationsInput) ActivityID() int { return in.activityID }
func (in ListByActivityReservationsInput) Limit() int      { return in.limit }
func (in ListByActivityReservationsInput) Offset() int     { return in.offset }

// NewListByActivityReservationsInput validates the activity id and normalizes pagination.
func NewListByActivityReservationsInput(activityID, limit, offset int) (ListByActivityReservationsInput, error) {
	if activityID <= 0 {
		return ListByActivityReservationsInput{}, &ValidationError{Field: "activity_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByActivityReservationsInput{activityID: activityID, limit: limit, offset: offset}, nil
}

// MustNewListByActivityReservationsInput panics on validation error. For tests.
func MustNewListByActivityReservationsInput(activityID, limit, offset int) ListByActivityReservationsInput {
	in, err := NewListByActivityReservationsInput(activityID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByActivityReservationsOutput carries the resulting views.
type ListByActivityReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByActivityReservationsUseCase returns a paginated list of
// every reservation for the given activity.
type ListByActivityReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByActivityReservationsUseCase(repo domain.ReservationRepository) *ListByActivityReservationsUseCase {
	return &ListByActivityReservationsUseCase{repo: repo}
}

func (uc *ListByActivityReservationsUseCase) Execute(ctx context.Context, input ListByActivityReservationsInput) (ListByActivityReservationsOutput, error) {
	views, err := uc.repo.ListByActivityView(ctx, input.ActivityID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByActivityReservationsOutput{}, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}
	return ListByActivityReservationsOutput{Views: views}, nil
}
