package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByDogReservationsInput is the validated input for listing
// every reservation for a specific dog.
type ListByDogReservationsInput struct {
	dogID  int
	limit  int
	offset int
}

func (in ListByDogReservationsInput) DogID() int  { return in.dogID }
func (in ListByDogReservationsInput) Limit() int  { return in.limit }
func (in ListByDogReservationsInput) Offset() int { return in.offset }

// NewListByDogReservationsInput validates the dog id and normalizes pagination.
func NewListByDogReservationsInput(dogID, limit, offset int) (ListByDogReservationsInput, error) {
	if dogID <= 0 {
		return ListByDogReservationsInput{}, &ValidationError{Field: "dog_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByDogReservationsInput{dogID: dogID, limit: limit, offset: offset}, nil
}

// MustNewListByDogReservationsInput panics on validation error. For tests.
func MustNewListByDogReservationsInput(dogID, limit, offset int) ListByDogReservationsInput {
	in, err := NewListByDogReservationsInput(dogID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByDogReservationsOutput carries the resulting views.
type ListByDogReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByDogReservationsUseCase returns a paginated list of every
// reservation for the given dog. No ownership check: this is an
// admin-style "history of this dog" view.
type ListByDogReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByDogReservationsUseCase(repo domain.ReservationRepository) *ListByDogReservationsUseCase {
	return &ListByDogReservationsUseCase{repo: repo}
}

func (uc *ListByDogReservationsUseCase) Execute(ctx context.Context, input ListByDogReservationsInput) (ListByDogReservationsOutput, error) {
	views, err := uc.repo.ListByDogView(ctx, input.DogID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByDogReservationsOutput{}, fmt.Errorf("list reservations for dog %d: %w", input.DogID(), err)
	}
	return ListByDogReservationsOutput{Views: views}, nil
}
