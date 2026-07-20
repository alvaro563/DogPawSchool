package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByDogReservationsInput is the input for listing every
// reservation for a specific dog. limit / offset are normalized by
// the handler.
type ListByDogReservationsInput struct {
	DogID  int
	Limit  int
	Offset int
}

// ListByDogReservationsOutput carries the resulting views, ordered
// by created_at DESC (most recent first).
type ListByDogReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByDogReservationsUseCase returns a paginated list of every
// reservation for the given dog. No ownership check: this is an
// admin-style "history of this dog" view; the path is the only
// authorization gate until an auth middleware is added.
type ListByDogReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByDogReservationsUseCase(repo domain.ReservationRepository) *ListByDogReservationsUseCase {
	return &ListByDogReservationsUseCase{repo: repo}
}

func (uc *ListByDogReservationsUseCase) Execute(ctx context.Context, input ListByDogReservationsInput) (ListByDogReservationsOutput, error) {
	if input.DogID <= 0 {
		return ListByDogReservationsOutput{}, &ValidationError{Field: "dog_id"}
	}
	views, err := uc.repo.ListByDogView(ctx, input.DogID, input.Limit, input.Offset)
	if err != nil {
		return ListByDogReservationsOutput{}, fmt.Errorf("list reservations for dog %d: %w", input.DogID, err)
	}
	return ListByDogReservationsOutput{Views: views}, nil
}
