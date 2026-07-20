package reservation

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ListByUserReservationsInput is the input for listing the
// reservations of a user (via dog ownership). status, from, to are
// optional filters: pass nil to skip. limit / offset are
// normalized by the handler via NormalizePagination before being
// passed in.
type ListByUserReservationsInput struct {
	UserID int
	Status *domain.ReservationStatus
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// ListByUserReservationsOutput carries the resulting views, ordered
// by created_at DESC (most recent first).
type ListByUserReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByUserReservationsUseCase returns a paginated list of the
// reservations that belong to the user (their dog's reservations).
// No transactor is needed: this is a single read with no side
// effects.
type ListByUserReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByUserReservationsUseCase(repo domain.ReservationRepository) *ListByUserReservationsUseCase {
	return &ListByUserReservationsUseCase{repo: repo}
}

func (uc *ListByUserReservationsUseCase) Execute(ctx context.Context, input ListByUserReservationsInput) (ListByUserReservationsOutput, error) {
	if err := input.validate(); err != nil {
		return ListByUserReservationsOutput{}, err
	}
	views, err := uc.repo.ListByUserView(ctx, input.UserID, input.Status, input.From, input.To, input.Limit, input.Offset)
	if err != nil {
		return ListByUserReservationsOutput{}, fmt.Errorf("list reservations for user %d: %w", input.UserID, err)
	}
	return ListByUserReservationsOutput{Views: views}, nil
}

func (input ListByUserReservationsInput) validate() error {
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	if input.Status != nil && !input.Status.IsValid() {
		return &ValidationError{Field: "status"}
	}
	if input.From != nil && input.To != nil && input.From.After(*input.To) {
		return &ValidationError{Field: "from"}
	}
	return nil
}
