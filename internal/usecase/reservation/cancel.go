package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

// CancelReservationInput is the validated payload for cancelling an
// existing reservation. UserID is the owner of the dog (and the
// pass) and is taken from the URL path by the handler.
type CancelReservationInput struct {
	UserID        int
	ReservationID int
}

// CancelReservationOutput is the result of a successful cancel. The
// full domain object is returned so the handler can serialize the
// new status (CANCELLED_IN_TIME or CANCELLED_LATE) directly.
type CancelReservationOutput struct {
	Reservation *domain.Reservation
}

// CancelReservationUseCase cancels a CONFIRMED reservation. The
// whole flow is wrapped in a single database transaction so that, on
// failure, the reservation status is never half-updated and the pass
// refund is never half-applied.
//
// Refund policy is enforced by the domain:
//
//   - If the cancel happens more than cancellationLateWindow before
//     the activity date, the reservation transitions to
//     StatusCancelledInTime AND the pass session is refunded
//     (remainingSessions++ and a +1 movement is appended to the
//     audit log).
//   - If the cancel happens within cancellationLateWindow, the
//     reservation transitions to StatusCancelledLate and NO refund
//     is applied. An admin can later call Forgive (future use case)
//     to refund a late cancellation.
//
// Ownership: the path UserID must match the dog.UserID() and
// pass.UserID() of the existing reservation. The two checks share
// the same error sentinel each (ErrInvalidDog, ErrInvalidPass) so
// we do not leak the existence of other users' data.
type CancelReservationUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
	now             func() time.Time
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
		now:             time.Now,
	}
}

// Execute runs the cancel flow atomically. Returns a typed error
// from this package (ErrInvalidReservation, ErrAlreadyCancelled,
// etc.) for the expected failure modes; any other error is wrapped
// with %w so the handler logs it as a 500.
func (uc *CancelReservationUseCase) Execute(ctx context.Context, input CancelReservationInput) (CancelReservationOutput, error) {
	if err := input.validate(); err != nil {
		return CancelReservationOutput{}, err
	}

	var output CancelReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		reservation, err := uc.runInTx(txCtx, input)
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

// runInTx performs every step of the cancel flow inside the
// transaction-bound context. Keeping this private and side-effect
// free (apart from the explicit repository calls) makes the
// Transactor.WithinTx closure easy to read.
func (uc *CancelReservationUseCase) runInTx(ctx context.Context, input CancelReservationInput) (*domain.Reservation, error) {
	now := uc.now()

	// 1. The reservation must exist.
	reservation, err := uc.reservationRepo.GetByID(ctx, input.ReservationID)
	if err != nil {
		if errors.Is(err, postgres.ErrReservationNotFound) {
			return nil, ErrInvalidReservation
		}
		return nil, fmt.Errorf("get reservation %d: %w", input.ReservationID, err)
	}

	// 2. The reservation must be in a cancellable state. The
	// domain's Cancel method also enforces this, but checking here
	// lets us return ErrAlreadyCancelled before any other work.
	if !reservation.IsConfirmed() {
		return nil, ErrAlreadyCancelled
	}

	// 3. The activity is needed for two reasons: (a) the
	// cancellation window depends on its date, and (b) we refuse
	// to cancel a reservation whose activity has already happened.
	activity, err := uc.activityRepo.GetByID(ctx, reservation.ActivityID())
	if err != nil {
		if errors.Is(err, postgres.ErrActivityNotFound) {
			return nil, ErrInvalidActivity
		}
		return nil, fmt.Errorf("get activity %d: %w", reservation.ActivityID(), err)
	}
	if activity.IsInThePast(now) {
		return nil, ErrActivityInPast
	}

	// 4. The dog must be owned by UserID. The same error is
	// returned for "not found" and "owned by another user" so we
	// do not leak the existence of other users' dogs.
	dog, err := uc.dogRepo.GetByID(ctx, reservation.DogID())
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidDog
		}
		return nil, fmt.Errorf("get dog %d: %w", reservation.DogID(), err)
	}
	if dog.UserID() != input.UserID {
		return nil, ErrInvalidDog
	}

	// 5. The pass must be owned by UserID (defensive: the
	// reservation was created with this pass, so the FK should
	// always be consistent; this catches data corruption early).
	pass, err := uc.passRepo.GetByID(ctx, reservation.PassID())
	if err != nil {
		if errors.Is(err, postgres.ErrPassNotFound) {
			return nil, ErrInvalidPass
		}
		return nil, fmt.Errorf("get pass %d: %w", reservation.PassID(), err)
	}
	if pass.UserID() != input.UserID {
		return nil, ErrInvalidPass
	}

	// 6. Apply the status change. The domain decides in-time vs
	// late based on the activity date and the current time. The
	// IsConfirmed check above already covers the "already
	// cancelled" case; the domain's defensive check is also kept
	// in case the status changed between the read and the write.
	if err := reservation.Cancel(activity.Date(), now); err != nil {
		return nil, ErrAlreadyCancelled
	}

	// 7. Refund the pass session if the cancel was in-time. We
	// also require pass.CanRefund() to be true: a pass is
	// "refundable" only when remainingSessions < numOfSessions.
	// This guards the (admittedly rare) case where a reservation
	// points at a pass that was never debited for it — refunding
	// would inflate the pass above its purchased capacity.
	if reservation.WasCancelledInTime() && pass.CanRefund() {
		reason := fmt.Sprintf("Reservation %d cancelled in time", reservation.ID())
		movement, err := pass.RefundSession(reason, now)
		if err != nil {
			return nil, fmt.Errorf("refund pass %d: %w", reservation.PassID(), err)
		}
		if err := uc.passRepo.Update(ctx, pass); err != nil {
			return nil, fmt.Errorf("update pass %d: %w", reservation.PassID(), err)
		}
		if err := uc.passRepo.AddMovement(ctx, &movement); err != nil {
			return nil, fmt.Errorf("add movement for pass %d: %w", reservation.PassID(), err)
		}
	}

	// 8. Persist the status change. The reservation row carries
	// the new status; updated_at is bumped by the DB trigger.
	if err := uc.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID, err)
	}

	return reservation, nil
}

func (input CancelReservationInput) validate() error {
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	if input.ReservationID <= 0 {
		return &ValidationError{Field: "reservation_id"}
	}
	return nil
}
