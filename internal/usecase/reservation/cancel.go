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
	adminOverride bool
}

func (in CancelReservationInput) UserID() int            { return in.userID }
func (in CancelReservationInput) ReservationID() int     { return in.reservationID }
func (in CancelReservationInput) Now() time.Time         { return in.now }
func (in CancelReservationInput) AdminOverride() bool    { return in.adminOverride }

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

// NewCancelReservationAdminInput validates the reservation ID and
// sets adminOverride so the use case skips the dog/pass ownership
// checks. The user_id is intentionally left at 0 — the admin does
// not need to know (and the handler should not require) the owner
// of the reservation being cancelled. The correct factory for the
// admin endpoint is this one, NOT NewCancelReservationInput with a
// placeholder user_id.
func NewCancelReservationAdminInput(reservationID int, now func() time.Time) (CancelReservationInput, error) {
	if reservationID <= 0 {
		return CancelReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return CancelReservationInput{reservationID: reservationID, now: now(), adminOverride: true}, nil
}

// MustNewCancelReservationAdminInput panics on validation error. For
// tests and other call sites that already know the inputs are
// valid.
func MustNewCancelReservationAdminInput(reservationID int, now func() time.Time) CancelReservationInput {
	in, err := NewCancelReservationAdminInput(reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
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
	// 1. Reservation must exist. We use FOR UPDATE so a concurrent
	// cancel-then-cancel sequence serializes on this row: the second
	// cancel waits for the first to commit, then sees the new
	// status (CANCELLED_IN_TIME / LATE) and is rejected by the SQL
	// guard in step 8. Without the lock, both cancels read the
	// status as CONFIRMED concurrently and both proceed to refund.
	reservation, err := uc.reservationRepo.GetByIDForUpdate(ctx, input.ReservationID())
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

	// Capture the status we just read. This is the snapshot we will
	// pass to Update so the SQL guard can detect a concurrent
	// transition that races past the in-memory check.
	originalStatus := reservation.Status()

	// 3. Activity: needed for cancellation window + the "no cancel
	// after the fact" guard.
	// Admin viewer: use case already gates ownership via input.UserID()
	// below; the SQL visibility filter is admin so every reachable
	// activity is observable here.
	activity, err := uc.activityRepo.GetByID(ctx, reservation.ActivityID(), 0, true)
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
	if !input.AdminOverride() && dog.UserID() != input.UserID() {
		return nil, ErrInvalidDog
	}

	// 5. Pass ownership. FOR UPDATE on the pass row serializes
	// against concurrent register/forgive/reject flows that also
	// need the row lock; without it, two parallel cancel/refund
	// flows can read the same pass.RemainingSessions() and both
	// RefundSession + Update, producing a double-refund or a
	// counter/audit-log drift.
	pass, err := uc.passRepo.GetByIDForUpdate(ctx, reservation.PassID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrInvalidPass
		}
		return nil, fmt.Errorf("get pass %d: %w", reservation.PassID(), err)
	}
	if pass == nil {
		return nil, ErrInvalidPass
	}
	if !input.AdminOverride() && pass.UserID() != input.UserID() {
		return nil, ErrInvalidPass
	}

	// Snapshot the pass row's updated_at right after the FOR UPDATE
	// read. ConsumeSession/RefundSession do NOT touch this field;
	// it is bumped only by the DB trigger on UPDATE. The value we
	// hold now is the only correct snapshot — if we took it after
	// RefundSession the in-memory value would still be the same
	// (the trigger hasn't fired yet), so the read order doesn't
	// matter, but the explicit comment makes the contract obvious.
	passUpdatedAt := pass.UpdatedAt()

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
		if err := uc.passRepo.Update(ctx, pass, passUpdatedAt); err != nil {
			return nil, fmt.Errorf("update pass %d: %w", reservation.PassID(), err)
		}
	}

	// 8. Persist the status change. The originalStatus guard
	// prevents a concurrent transition (e.g. confirm+reject race,
	// second cancel) from silently overwriting this write.
	if err := uc.reservationRepo.Update(ctx, reservation, originalStatus); err != nil {
		if errors.Is(err, domain.ErrReservationStateChanged) {
			// Wrap so the handler can still match the sentinel via
			// errors.Is while the log carries the reservation id.
			return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
		}
		return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
	}

	return reservation, nil
}
