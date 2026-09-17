package reservation

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllReservationsInput is the validated input for listing all
// reservations in the system. It carries pagination and an optional
// status filter.
type ListAllReservationsInput struct {
	limit  int
	offset int
	status *domain.ReservationStatus
}

func (in ListAllReservationsInput) Limit() int  { return in.limit }
func (in ListAllReservationsInput) Offset() int { return in.offset }
func (in ListAllReservationsInput) Status() *domain.ReservationStatus {
	return in.status
}

// NewListAllReservationsInput normalizes pagination and validates the
// optional status filter. A nil status means "no filter".
func NewListAllReservationsInput(limit, offset int, status *domain.ReservationStatus) (ListAllReservationsInput, error) {
	limit, offset = normalizePagination(limit, offset)
	if status != nil && !status.IsValid() {
		return ListAllReservationsInput{}, &ValidationError{Field: "status"}
	}
	return ListAllReservationsInput{limit: limit, offset: offset, status: status}, nil
}

// MustNewListAllReservationsInput panics on error. For tests.
func MustNewListAllReservationsInput(limit, offset int, status *domain.ReservationStatus) ListAllReservationsInput {
	in, err := NewListAllReservationsInput(limit, offset, status)
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
	views, err := uc.repo.ListAllView(ctx, input.Limit(), input.Offset(), input.Status())
	if err != nil {
		return ListAllReservationsOutput{}, fmt.Errorf("list all reservations: %w", err)
	}
	return ListAllReservationsOutput{Views: views}, nil
}
