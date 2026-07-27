package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByPassReservationsInput is the validated input for listing
// every reservation paid from a specific pass (pass audit view).
type ListByPassReservationsInput struct {
	passID int
	limit  int
	offset int
}

func (in ListByPassReservationsInput) PassID() int { return in.passID }
func (in ListByPassReservationsInput) Limit() int  { return in.limit }
func (in ListByPassReservationsInput) Offset() int { return in.offset }

// NewListByPassReservationsInput validates the pass id and normalizes pagination.
func NewListByPassReservationsInput(passID, limit, offset int) (ListByPassReservationsInput, error) {
	if passID <= 0 {
		return ListByPassReservationsInput{}, &ValidationError{Field: "pass_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByPassReservationsInput{passID: passID, limit: limit, offset: offset}, nil
}

// MustNewListByPassReservationsInput panics on validation error. For tests.
func MustNewListByPassReservationsInput(passID, limit, offset int) ListByPassReservationsInput {
	in, err := NewListByPassReservationsInput(passID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByPassReservationsOutput carries the resulting views.
type ListByPassReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByPassReservationsUseCase returns a paginated list of every
// reservation that was paid from the given pass. No ownership
// check: this is a "what was consumed by this pass" view.
type ListByPassReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByPassReservationsUseCase(repo domain.ReservationRepository) *ListByPassReservationsUseCase {
	return &ListByPassReservationsUseCase{repo: repo}
}

func (uc *ListByPassReservationsUseCase) Execute(ctx context.Context, input ListByPassReservationsInput) (ListByPassReservationsOutput, error) {
	views, err := uc.repo.ListByPassView(ctx, input.PassID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByPassReservationsOutput{}, fmt.Errorf("list reservations for pass %d: %w", input.PassID(), err)
	}
	return ListByPassReservationsOutput{Views: views}, nil
}
