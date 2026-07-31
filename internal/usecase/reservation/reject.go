package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RejectPendingReservationInput is the validated admin command to
// demote a PENDING_TO_CONFIRM reservation to CANCELLED_IN_TIME,
// refunding the consumed pass session.
type RejectPendingReservationInput struct {
	reservationID int
	now           time.Time
}

func (in RejectPendingReservationInput) ReservationID() int { return in.reservationID }
func (in RejectPendingReservationInput) Now() time.Time     { return in.now }

// NewRejectPendingReservationInput validates reservationID > 0 and
// accepts a now-Provider so the use case can be tested with a fixed
// clock. A nil provider is replaced with time.Now.
func NewRejectPendingReservationInput(reservationID int, now func() time.Time) (RejectPendingReservationInput, error) {
	if reservationID <= 0 {
		return RejectPendingReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return RejectPendingReservationInput{reservationID: reservationID, now: now()}, nil
}

// MustNewRejectPendingReservationInput panics on validation error. For tests.
func MustNewRejectPendingReservationInput(reservationID int, now func() time.Time) RejectPendingReservationInput {
	in, err := NewRejectPendingReservationInput(reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// RejectPendingReservationOutput is the result of a successful reject.
type RejectPendingReservationOutput struct {
	Reservation *domain.Reservation
}

// RejectPendingReservationUseCase lets an admin reject a reservation
// held in StatusPendingToConfirm. The reservation transitions to
// StatusCancelledInTime (freeing the slot) and, following the cancel
// in-time policy, the pass session consumed at booking is refunded.
type RejectPendingReservationUseCase struct {
	transactor      Transactor
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
}

func NewRejectPendingReservationUseCase(
	transactor Transactor,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
) *RejectPendingReservationUseCase {
	return &RejectPendingReservationUseCase{
		transactor:      transactor,
		passRepo:        passRepo,
		reservationRepo: reservationRepo,
	}
}

func (uc *RejectPendingReservationUseCase) Execute(ctx context.Context, input RejectPendingReservationInput) (RejectPendingReservationOutput, error) {
	var output RejectPendingReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		reservation, err := uc.reservationRepo.GetByID(txCtx, input.ReservationID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("get reservation %d: %w", input.ReservationID(), err)
		}
		if reservation == nil {
			return ErrNotFound
		}
		if !reservation.IsPending() {
			return fmt.Errorf("%w: current status is %s", ErrNotPending, reservation.Status())
		}
		if err := reservation.RejectPending(); err != nil {
			return fmt.Errorf("%w: %v", ErrNotPending, err)
		}

		// Refund the pass session consumed at booking, mirroring the
		// cancel in-time policy. The audit movement rides along on the
		// aggregate and is flushed by Update.
		if pass, err := uc.passRepo.GetByID(txCtx, reservation.PassID()); err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("get pass %d: %w", reservation.PassID(), err)
			}
		} else if pass != nil && reservation.WasCancelledInTime() && pass.CanRefund() {
			reason := fmt.Sprintf("Reservation %d rejected by admin", reservation.ID())
			if _, err := pass.RefundSession(reason, input.Now()); err != nil {
				return fmt.Errorf("refund pass %d: %w", reservation.PassID(), err)
			}
			if err := uc.passRepo.Update(txCtx, pass); err != nil {
				return fmt.Errorf("update pass %d: %w", reservation.PassID(), err)
			}
		}

		if err := uc.reservationRepo.Update(txCtx, reservation); err != nil {
			return fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
		}
		output = RejectPendingReservationOutput{Reservation: reservation}
		return nil
	})
	if err != nil {
		return RejectPendingReservationOutput{}, err
	}
	return output, nil
}
