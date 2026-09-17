package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ForgiveReservationInput is the validated admin command to transition
// a CANCELLED_LATE reservation to FORGIVEN. The use case is admin-only;
// there is no user-facing factory. The transition is the only way to
// refund a late cancellation.
type ForgiveReservationInput struct {
	reservationID int
	now           time.Time
}

func (in ForgiveReservationInput) ReservationID() int { return in.reservationID }
func (in ForgiveReservationInput) Now() time.Time     { return in.now }

// NewForgiveReservationInput validates reservationID > 0 and accepts a
// now-Provider so the use case can be tested with a fixed clock. A nil
// provider is replaced with time.Now.
func NewForgiveReservationInput(reservationID int, now func() time.Time) (ForgiveReservationInput, error) {
	if reservationID <= 0 {
		return ForgiveReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return ForgiveReservationInput{reservationID: reservationID, now: now()}, nil
}

// MustNewForgiveReservationInput panics on validation error. For tests.
func MustNewForgiveReservationInput(reservationID int, now func() time.Time) ForgiveReservationInput {
	in, err := NewForgiveReservationInput(reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// ForgiveReservationOutput is the result of a successful forgive.
// PassSessionRefunded is true iff the pass actually gained one session
// back. It can be false when pass.CanRefund() returns false (e.g. fresh
// pass with nothing to refund) — the reservation still transitions to
// FORGIVEN, the operation just does not move any session balance.
type ForgiveReservationOutput struct {
	Reservation         *domain.Reservation
	PassSessionRefunded bool
}

// ForgiveReservationUseCase is the admin-only path that converts a
// CANCELLED_LATE reservation into FORGIVEN and, when the pass has
// available balance, refunds the consumed session.
//
// The only entry-point is admin (the route is mounted in the admin
// group of the router), so the use case does not perform ownership or
// activity-window checks. The transition is the only way to refund a
// late cancellation.
type ForgiveReservationUseCase struct {
	transactor      Transactor
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
}

func NewForgiveReservationUseCase(
	transactor Transactor,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
) *ForgiveReservationUseCase {
	return &ForgiveReservationUseCase{
		transactor:      transactor,
		passRepo:        passRepo,
		reservationRepo: reservationRepo,
	}
}

// Execute runs the forgive flow atomically: load + lock reservation,
// validate the status precondition, transition to FORGIVEN, refund the
// pass session if possible, and persist both writes. On any error the
// transactor rolls the transaction back, leaving neither the
// reservation status nor the pass balance changed.
func (uc *ForgiveReservationUseCase) Execute(ctx context.Context, input ForgiveReservationInput) (ForgiveReservationOutput, error) {
	var output ForgiveReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		reservation, refunded, err := uc.runInTx(txCtx, input)
		if err != nil {
			return err
		}
		output = ForgiveReservationOutput{Reservation: reservation, PassSessionRefunded: refunded}
		return nil
	})
	if err != nil {
		return ForgiveReservationOutput{}, err
	}
	return output, nil
}

func (uc *ForgiveReservationUseCase) runInTx(ctx context.Context, input ForgiveReservationInput) (*domain.Reservation, bool, error) {
	// 1. Load the reservation row. The downstream status precondition
	// (Forgive -> only CANCELLED_LATE) prevents double-forgive even
	// without FOR UPDATE: the second concurrent admin sees
	// StatusForgiven and is rejected by the precondition. We mirror
	// RejectPendingReservationUseCase which uses the same pattern.
	reservation, err := uc.reservationRepo.GetByID(ctx, input.ReservationID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, false, ErrNotFound
		}
		return nil, false, fmt.Errorf("get reservation %d: %w", input.ReservationID(), err)
	}
	if reservation == nil {
		return nil, false, ErrNotFound
	}

	// 2. Pre-condition: only CANCELLED_LATE can be forgiven. The
	// domain-level check is authoritative; we map its error to the
	// use-case sentinel so the handler can match with errors.Is.
	if err := reservation.Forgive(); err != nil {
		return nil, false, fmt.Errorf("%w: %s", ErrNotLateCancelled, reservation.Status())
	}

	// 3. Lock the pass and refund the consumed session. Refund is
	// conditional on pass.CanRefund() — when the pass is already at
	// initial balance there is nothing to refund, but the reservation
	// still transitions to FORGIVEN.
	pass, err := uc.passRepo.GetByIDForUpdate(ctx, reservation.PassID())
	refunded := false
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, false, fmt.Errorf("get pass %d: %w", reservation.PassID(), err)
		}
		// Pass not found: skip the refund but still forgive the
		// reservation. Surfaces refunded=false on the wire so the
		// admin can investigate.
	} else if pass != nil && pass.CanRefund() {
		reason := fmt.Sprintf("Reservation %d forgiven by admin", reservation.ID())
		if _, err := pass.RefundSession(reason, input.Now()); err != nil {
			return nil, false, fmt.Errorf("refund pass %d: %w", reservation.PassID(), err)
		}
		if err := uc.passRepo.Update(ctx, pass); err != nil {
			return nil, false, fmt.Errorf("update pass %d: %w", reservation.PassID(), err)
		}
		refunded = true
	}

	// 4. Persist the new reservation status.
	if err := uc.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, false, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
	}

	return reservation, refunded, nil
}
