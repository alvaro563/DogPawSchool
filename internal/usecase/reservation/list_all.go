package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllReservationsInput is the validated input for listing all
// reservations in the system. It only carries pagination; there are
// no entity-level filters.
type ListAllReservationsInput struct {
	limit  int
	offset int
}

func (in ListAllReservationsInput) Limit() int  { return in.limit }
func (in ListAllReservationsInput) Offset() int { return in.offset }

// NewListAllReservationsInput normalizes pagination. Error is always
// nil for pure-pagination inputs.
func NewListAllReservationsInput(limit, offset int) (ListAllReservationsInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListAllReservationsInput{limit: limit, offset: offset}, nil
}

// MustNewListAllReservationsInput panics on error. For tests.
func MustNewListAllReservationsInput(limit, offset int) ListAllReservationsInput {
	in, err := NewListAllReservationsInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListAllReservationsOutput carries the resulting views.
type ListAllReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListAllReservationsUseCase returns a paginated list of every
// reservation in the system, most recent first.
type ListAllReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListAllReservationsUseCase(repo domain.ReservationRepository) *ListAllReservationsUseCase {
	return &ListAllReservationsUseCase{repo: repo}
}

func (uc *ListAllReservationsUseCase) Execute(ctx context.Context, input ListAllReservationsInput) (ListAllReservationsOutput, error) {
	views, err := uc.repo.ListAllView(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListAllReservationsOutput{}, fmt.Errorf("list all reservations: %w", err)
	}
	return ListAllReservationsOutput{Views: views}, nil
}
