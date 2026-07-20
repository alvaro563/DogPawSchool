package reservation

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

// GetReservationInput is the input for fetching a single
// reservation by id, with ownership enforcement.
type GetReservationInput struct {
	UserID        int
	ReservationID int
}

// GetReservationOutput carries the denormalized view.
type GetReservationOutput struct {
	View *domain.ReservationView
}

// GetReservationUseCase returns the denormalized ReservationView
// for a single reservation id. It enforces that the reservation
// belongs to the user in the path (via dog.UserID) so a user cannot
// fetch another user's reservation. The same error (not_found) is
// returned for "id does not exist" and "exists but not owned by
// this user" so the API does not leak the existence of other
// users' reservations.
type GetReservationUseCase struct {
	repo domain.ReservationRepository
}

func NewGetReservationUseCase(repo domain.ReservationRepository) *GetReservationUseCase {
	return &GetReservationUseCase{repo: repo}
}

func (uc *GetReservationUseCase) Execute(ctx context.Context, input GetReservationInput) (GetReservationOutput, error) {
	if input.UserID <= 0 {
		return GetReservationOutput{}, &ValidationError{Field: "user_id"}
	}
	if input.ReservationID <= 0 {
		return GetReservationOutput{}, &ValidationError{Field: "reservation_id"}
	}
	view, err := uc.repo.GetView(ctx, input.ReservationID)
	if err != nil {
		if errors.Is(err, postgres.ErrReservationNotFound) {
			return GetReservationOutput{}, ErrInvalidReservation
		}
		return GetReservationOutput{}, fmt.Errorf("get reservation view %d: %w", input.ReservationID, err)
	}
	// Ownership: the view exposes the dog's userID (denormalized
	// in the JOIN). If the dog does not belong to the user, return
	// not_found to avoid leaking existence.
	if view.DogUserID() != input.UserID {
		return GetReservationOutput{}, ErrReservationNotOwned
	}
	return GetReservationOutput{View: view}, nil
}
