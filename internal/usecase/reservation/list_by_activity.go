package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByActivityReservationsInput is the input for listing every
// reservation for a specific activity (class roster view). limit /
// offset are normalized by the handler.
type ListByActivityReservationsInput struct {
	ActivityID int
	Limit      int
	Offset     int
}

// ListByActivityReservationsOutput carries the resulting views,
// ordered by created_at ASC (chronological order is the most
// useful for a class roster).
type ListByActivityReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByActivityReservationsUseCase returns a paginated list of
// every reservation for the given activity. No ownership check:
// this is an admin/educator "who is signed up for this class"
// view; the path is the only authorization gate until an auth
// middleware is added.
type ListByActivityReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByActivityReservationsUseCase(repo domain.ReservationRepository) *ListByActivityReservationsUseCase {
	return &ListByActivityReservationsUseCase{repo: repo}
}

func (uc *ListByActivityReservationsUseCase) Execute(ctx context.Context, input ListByActivityReservationsInput) (ListByActivityReservationsOutput, error) {
	if input.ActivityID <= 0 {
		return ListByActivityReservationsOutput{}, &ValidationError{Field: "activity_id"}
	}
	views, err := uc.repo.ListByActivityView(ctx, input.ActivityID, input.Limit, input.Offset)
	if err != nil {
		return ListByActivityReservationsOutput{}, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID, err)
	}
	return ListByActivityReservationsOutput{Views: views}, nil
}
