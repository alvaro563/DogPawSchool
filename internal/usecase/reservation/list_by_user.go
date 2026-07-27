package reservation

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ListByUserReservationsInput is the validated input for listing
// the reservations of a user (via dog ownership). status, from, to
// are optional filters: pass nil to skip. limit / offset are
// normalized by the factory.
type ListByUserReservationsInput struct {
	userID int
	status *domain.ReservationStatus
	from   *time.Time
	to     *time.Time
	limit  int
	offset int
}

func (in ListByUserReservationsInput) UserID() int                       { return in.userID }
func (in ListByUserReservationsInput) Status() *domain.ReservationStatus { return in.status }
func (in ListByUserReservationsInput) From() *time.Time                  { return in.from }
func (in ListByUserReservationsInput) To() *time.Time                    { return in.to }
func (in ListByUserReservationsInput) Limit() int                        { return in.limit }
func (in ListByUserReservationsInput) Offset() int                       { return in.offset }

// NewListByUserReservationsInput validates the user id, the optional
// status enum, the from<=to range, and normalizes pagination.
func NewListByUserReservationsInput(
	userID int,
	status *domain.ReservationStatus,
	from, to *time.Time,
	limit, offset int,
) (ListByUserReservationsInput, error) {
	if userID <= 0 {
		return ListByUserReservationsInput{}, &ValidationError{Field: "user_id"}
	}
	if status != nil && !status.IsValid() {
		return ListByUserReservationsInput{}, &ValidationError{Field: "status"}
	}
	if from != nil && to != nil && from.After(*to) {
		return ListByUserReservationsInput{}, &ValidationError{Field: "from"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByUserReservationsInput{userID: userID, status: status, from: from, to: to, limit: limit, offset: offset}, nil
}

// MustNewListByUserReservationsInput panics on validation error. For tests.
func MustNewListByUserReservationsInput(
	userID int,
	status *domain.ReservationStatus,
	from, to *time.Time,
	limit, offset int,
) ListByUserReservationsInput {
	in, err := NewListByUserReservationsInput(userID, status, from, to, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByUserReservationsOutput carries the resulting views.
type ListByUserReservationsOutput struct {
	Views []*domain.ReservationView
}

// ListByUserReservationsUseCase returns a paginated list of the
// reservations that belong to the user (their dog's reservations).
type ListByUserReservationsUseCase struct {
	repo domain.ReservationRepository
}

func NewListByUserReservationsUseCase(repo domain.ReservationRepository) *ListByUserReservationsUseCase {
	return &ListByUserReservationsUseCase{repo: repo}
}

func (uc *ListByUserReservationsUseCase) Execute(ctx context.Context, input ListByUserReservationsInput) (ListByUserReservationsOutput, error) {
	views, err := uc.repo.ListByUserView(ctx, input.UserID(), input.Status(), input.From(), input.To(), input.Limit(), input.Offset())
	if err != nil {
		return ListByUserReservationsOutput{}, fmt.Errorf("list reservations for user %d: %w", input.UserID(), err)
	}
	return ListByUserReservationsOutput{Views: views}, nil
}
