package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// CancelReservationInput is the validated command to cancel a
// reservation.
type CancelReservationInput struct {
	userID        int
	reservationID int
	now           time.Time
}

func (in CancelReservationInput) UserID() int        { return in.userID }
func (in CancelReservationInput) ReservationID() int { return in.reservationID }
func (in CancelReservationInput) Now() time.Time     { return in.now }

// NewCancelReservationInput validates the two ids and accepts a
// now-Provider so the use case can be tested with a fixed clock.
func NewCancelReservationInput(userID, reservationID int, now func() time.Time) (CancelReservationInput, error) {
	if userID <= 0 {
		return CancelReservationInput{}, &ValidationError{Field: "user_id"}
	}
	if reservationID <= 0 {
		return CancelReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return CancelReservationInput{userID: userID, reservationID: reservationID, now: now()}, nil
}

// MustNewCancelReservationInput panics on validation error. For tests.
func MustNewCancelReservationInput(userID, reservationID int, now func() time.Time) CancelReservationInput {
	in, err := NewCancelReservationInput(userID, reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// CancelReservationOutput is the result of a successful cancel.
type CancelReservationOutput struct {
	Reservation *domain.Reservation
}

// CancelReservationUseCase cancels a CONFIRMED reservation. The
// whole flow is wrapped in a single database transaction.
//
// Refund policy is enforced by the domain:
//
//   - If the cancel happens more than cancellationLateWindow before
//     the activity date, the reservation transitions to
//     StatusCancelledInTime AND the pass session is refunded.
//   - If the cancel happens within cancellationLateWindow, the
//     reservation transitions to StatusCancelledLate and NO refund
//     is applied.
//
// The use case holds no mutable state: the clock travels with the
// input (CancelReservationInput freezes it at construction), so a
// single instance is safe to share across concurrent requests.
type CancelReservationUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
}

func NewCancelReservationUseCase(
	transactor Transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
) *CancelReservationUseCase {
	return &CancelReservationUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		passRepo:        passRepo,
		reservationRepo: reservationRepo,
	}
}

func (uc *CancelReservationUseCase) Execute(ctx context.Context, input CancelReservationInput) (CancelReservationOutput, error) {
	var output CancelReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		reservation, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = CancelReservationOutput{Reservation: reservation}
		return nil
	})
	if err != nil {
		return CancelReservationOutput{}, err
	}
	return output, nil
}

func (uc *CancelReservationUseCase) runInTx(ctx context.Context, input CancelReservationInput, now time.Time) (*domain.Reservation, error) {
	// 1. Reservation must exist.
	reservation, err := uc.reservationRepo.GetByID(ctx, input.ReservationID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrInvalidReservation
		}
		return nil, fmt.Errorf("get reservation %d: %w", input.ReservationID(), err)
	}
	if reservation == nil {
		return nil, ErrInvalidReservation
	}

	// 2. Reservation must be cancellable.
	if !reservation.IsConfirmed() {
		return nil, ErrAlreadyCancelled
	}

	// 3. Activity: needed for cancellation window + the "no cancel
	// after the fact" guard.
	activity, err := uc.activityRepo.GetByID(ctx, reservation.ActivityID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrInvalidActivity
		}
		return nil, fmt.Errorf("get activity %d: %w", reservation.ActivityID(), err)
	}
	if activity == nil {
		return nil, ErrInvalidActivity
	}
	if activity.IsInThePast(now) {
		return nil, ErrActivityInPast
	}

	// 4. Dog ownership.
	dog, err := uc.dogRepo.GetByID(ctx, reservation.DogID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrInvalidDog
		}
		return nil, fmt.Errorf("get dog %d: %w", reservation.DogID(), err)
	}
	if dog == nil {
		return nil, ErrInvalidDog
	}
	if dog.UserID() != input.UserID() {
		return nil, ErrInvalidDog
	}

	// 5. Pass ownership.
	pass, err := uc.passRepo.GetByID(ctx, reservation.PassID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrInvalidPass
		}
		return nil, fmt.Errorf("get pass %d: %w", reservation.PassID(), err)
	}
	if pass == nil {
		return nil, ErrInvalidPass
	}
	if pass.UserID() != input.UserID() {
		return nil, ErrInvalidPass
	}

	// 6. Apply the status change. The domain decides in-time vs
	// late.
	// The domain error is wrapped rather than replaced: the handler
	// still matches ErrAlreadyCancelled with errors.Is, but the log
	// keeps the reason the transition was refused.
	if err := reservation.Cancel(activity.Date(), now); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAlreadyCancelled, err)
	}

	// 7. Refund the pass session if the cancel was in-time. The audit
	// movement rides along on the aggregate and is flushed by Update.
	if reservation.WasCancelledInTime() && pass.CanRefund() {
		reason := fmt.Sprintf("Reservation %d cancelled in time", reservation.ID())
		if _, err := pass.RefundSession(reason, now); err != nil {
			return nil, fmt.Errorf("refund pass %d: %w", reservation.PassID(), err)
		}
		if err := uc.passRepo.Update(ctx, pass); err != nil {
			return nil, fmt.Errorf("update pass %d: %w", reservation.PassID(), err)
		}
	}

	// 8. Persist the status change.
	if err := uc.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
	}

	return reservation, nil
}
