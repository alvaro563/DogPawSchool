package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByPassReservationsInput is the input for listing every
// reservation paid from a specific pass (pass audit view). limit /
// offset are normalized by the handler.
type ListByPassReservationsInput struct {
	PassID int
	Limit  int
	Offset int
}

// ListByPassReservationsOutput carries the resulting views, ordered
// by created_at DESC (most recent first).
type ListByPassReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByPassReservationsUseCase returns a paginated list of every
// reservation that was paid from the given pass. No ownership
// check: this is a "what was consumed by this pass" view; the
// path is the only authorization gate until an auth middleware is
// added.
type ListByPassReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByPassReservationsUseCase(repo domain.ReservationRepository) *ListByPassReservationsUseCase {
	return &ListByPassReservationsUseCase{repo: repo}
}

func (uc *ListByPassReservationsUseCase) Execute(ctx context.Context, input ListByPassReservationsInput) (ListByPassReservationsOutput, error) {
	if input.PassID <= 0 {
		return ListByPassReservationsOutput{}, &ValidationError{Field: "pass_id"}
	}
	views, err := uc.repo.ListByPassView(ctx, input.PassID, input.Limit, input.Offset)
	if err != nil {
		return ListByPassReservationsOutput{}, fmt.Errorf("list reservations for pass %d: %w", input.PassID, err)
	}
	return ListByPassReservationsOutput{Views: views}, nil
}
