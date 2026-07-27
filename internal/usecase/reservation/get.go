package reservation

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// GetReservationInput is the validated input for fetching a single
// reservation by id, with ownership enforcement.
type GetReservationInput struct {
	userID        int
	reservationID int
}

func (in GetReservationInput) UserID() int        { return in.userID }
func (in GetReservationInput) ReservationID() int { return in.reservationID }

// NewGetReservationInput validates both ids.
func NewGetReservationInput(userID, reservationID int) (GetReservationInput, error) {
	if userID <= 0 {
		return GetReservationInput{}, &ValidationError{Field: "user_id"}
	}
	if reservationID <= 0 {
		return GetReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	return GetReservationInput{userID: userID, reservationID: reservationID}, nil
}

// MustNewGetReservationInput panics on validation error. For tests.
func MustNewGetReservationInput(userID, reservationID int) GetReservationInput {
	in, err := NewGetReservationInput(userID, reservationID)
	if err != nil {
		panic(err)
	}
	return in
}

// GetReservationOutput carries the denormalized view.
type GetReservationOutput struct {
	View *domain.ReservationView
}

// GetReservationUseCase returns the denormalized ReservationView
// for a single reservation id, enforcing that the reservation
// belongs to the user in the path. The same error is returned for
// "id does not exist" and "exists but not owned by this user" so
// the API does not leak the existence of other users' reservations.
type GetReservationUseCase struct {
	repo domain.ReservationRepository
}

func NewGetReservationUseCase(repo domain.ReservationRepository) *GetReservationUseCase {
	return &GetReservationUseCase{repo: repo}
}

func (uc *GetReservationUseCase) Execute(ctx context.Context, input GetReservationInput) (GetReservationOutput, error) {
	view, err := uc.repo.GetView(ctx, input.ReservationID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return GetReservationOutput{}, ErrInvalidReservation
		}
		return GetReservationOutput{}, fmt.Errorf("get reservation view %d: %w", input.ReservationID(), err)
	}
	if view.DogUserID() != input.UserID() {
		return GetReservationOutput{}, ErrReservationNotOwned
	}
	return GetReservationOutput{View: view}, nil
}
