package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// MarkReservationNoShowInput is the validated command to mark a
// reservation as no-show. Fields are private: only
// NewMarkReservationNoShowInput can construct one.
type MarkReservationNoShowInput struct {
	userID        int
	reservationID int
	now           time.Time
}

func (in MarkReservationNoShowInput) UserID() int        { return in.userID }
func (in MarkReservationNoShowInput) ReservationID() int { return in.reservationID }
func (in MarkReservationNoShowInput) Now() time.Time     { return in.now }

// NewMarkReservationNoShowInput is the validating factory. It
// validates the two ids and accepts a now-Provider so the use
// case can be tested with a fixed clock.
func NewMarkReservationNoShowInput(userID, reservationID int, now func() time.Time) (MarkReservationNoShowInput, error) {
	if userID <= 0 {
		return MarkReservationNoShowInput{}, &ValidationError{Field: "user_id"}
	}
	if reservationID <= 0 {
		return MarkReservationNoShowInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return MarkReservationNoShowInput{userID: userID, reservationID: reservationID, now: now()}, nil
}

// MustNewMarkReservationNoShowInput panics on validation error. For tests.
func MustNewMarkReservationNoShowInput(userID, reservationID int, now func() time.Time) MarkReservationNoShowInput {
	in, err := NewMarkReservationNoShowInput(userID, reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// MarkReservationNoShowOutput is the result of a successful
// no-show mark. The full reservation is returned so the handler
// can serialize the new state (StatusNoShow).
type MarkReservationNoShowOutput struct {
	Reservation *domain.Reservation
}

// MarkReservationNoShowUseCase marks a CONFIRMED reservation as
// no-show. The full flow is wrapped in a single transaction so
// that the load-then-update is atomic.
//
// Policy (enforced by the use case, not the entity):
//
//   - The reservation must exist.
//   - The reservation's activity must exist.
//   - The activity must have already started (date < now). This
//     is the central new policy: you cannot mark no-show a slot
//     that has not been missed yet.
//   - The reservation's dog must exist and be owned by the user
//     in the path (owner check; no leak).
//   - The reservation must be in StatusConfirmed (the domain
//     enforces this transition in Reservation.MarkNoShow; the use
//     case translates any failure into ErrNotCancellable, 409).
//   - No pass refund: the slot is past, the session is consumed.
//   - No pass movement appended: same reason.
//
// The use case holds no mutable state: the clock travels with the
// input, so a single instance is safe to share across concurrent
// requests.
type MarkReservationNoShowUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	reservationRepo domain.ReservationRepository
}

func NewMarkReservationNoShowUseCase(
	transactor Transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
) *MarkReservationNoShowUseCase {
	return &MarkReservationNoShowUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		reservationRepo: reservationRepo,
	}
}

func (uc *MarkReservationNoShowUseCase) Execute(ctx context.Context, input MarkReservationNoShowInput) (MarkReservationNoShowOutput, error) {
	var output MarkReservationNoShowOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		r, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = MarkReservationNoShowOutput{Reservation: r}
		return nil
	})
	return output, err
}

func (uc *MarkReservationNoShowUseCase) runInTx(ctx context.Context, input MarkReservationNoShowInput, now time.Time) (*domain.Reservation, error) {
	// 1. Load reservation.
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

	// 2. Load activity (needed for the "activity started" check).
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

	// 3. Policy: the activity must have already started
	// (date < now, strict). activity.IsInThePast uses date.Before,
	// so a slot starting at exactly `now` is NOT considered
	// started; this matches the Cancel flow's `ErrActivityInPast`
	// semantics for consistency.
	if !activity.IsInThePast(now) {
		return nil, ErrActivityNotStarted
	}

	// 4. Load dog (owner check).
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
		// Same no-leak policy as Cancel: surface as the same
		// ErrInvalidDog so the client cannot probe for the
		// existence of other users' dogs.
		return nil, ErrInvalidDog
	}

	// 5. Apply the status transition. The domain enforces
	// StatusConfirmed; any other state returns an error. We
	// translate that to ErrNotCancellable (409 not_cancellable).
	if err := reservation.MarkNoShow(); err != nil {
		return nil, ErrNotCancellable
	}

	// 6. Persist the new status. No pass refund, no pass movement:
	// the slot is past and the session is already consumed.
	if err := uc.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
	}

	return reservation, nil
}
