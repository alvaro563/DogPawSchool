package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// CompleteReservationInput is the validated command to mark a
// reservation as completed. Fields are private: only
// NewCompleteReservationInput can construct one.
type CompleteReservationInput struct {
	userID        int
	reservationID int
	now           time.Time
}

func (in CompleteReservationInput) UserID() int        { return in.userID }
func (in CompleteReservationInput) ReservationID() int { return in.reservationID }
func (in CompleteReservationInput) Now() time.Time     { return in.now }

// NewCompleteReservationInput is the validating factory. It
// validates the two ids and accepts a now-Provider so the use
// case can be tested with a fixed clock.
func NewCompleteReservationInput(userID, reservationID int, now func() time.Time) (CompleteReservationInput, error) {
	if userID <= 0 {
		return CompleteReservationInput{}, &ValidationError{Field: "user_id"}
	}
	if reservationID <= 0 {
		return CompleteReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	if now == nil {
		now = time.Now
	}
	return CompleteReservationInput{userID: userID, reservationID: reservationID, now: now()}, nil
}

// MustNewCompleteReservationInput panics on validation error. For tests.
func MustNewCompleteReservationInput(userID, reservationID int, now func() time.Time) CompleteReservationInput {
	in, err := NewCompleteReservationInput(userID, reservationID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// CompleteReservationOutput is the result of a successful
// reservation completion. The full reservation is returned so the
// handler can serialize the new state (StatusCompleted).
type CompleteReservationOutput struct {
	Reservation *domain.Reservation
}

// CompleteReservationUseCase marks a CONFIRMED reservation as
// completed. The full flow is wrapped in a single transaction so
// that the load-then-update is atomic.
//
// Policy (enforced by the use case, not the entity):
//
//   - The reservation must exist.
//   - The reservation's activity must exist.
//   - The activity must have finished (date + duration < now).
//     This is the central new policy: you cannot complete a
//     reservation for an activity that is still running.
//   - The reservation's dog must exist and be owned by the user
//     in the path (owner check; no leak).
//   - The reservation must be in StatusConfirmed (the domain
//     enforces this transition in Reservation.Complete; the use
//     case translates any failure into ErrNotCompletable, 409).
//   - No pass refund: the session was consumed at registration
//     and the activity has been delivered.
type CompleteReservationUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	reservationRepo domain.ReservationRepository
	now             func() time.Time
}

func NewCompleteReservationUseCase(
	transactor Transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	now func() time.Time,
) *CompleteReservationUseCase {
	return &CompleteReservationUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		reservationRepo: reservationRepo,
		now:             now,
	}
}

func (uc *CompleteReservationUseCase) Execute(ctx context.Context, input CompleteReservationInput) (CompleteReservationOutput, error) {
	now := input.Now()
	uc.now = func() time.Time { return now }

	var output CompleteReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		r, err := uc.runInTx(txCtx, input)
		if err != nil {
			return err
		}
		output = CompleteReservationOutput{Reservation: r}
		return nil
	})
	return output, err
}

func (uc *CompleteReservationUseCase) runInTx(ctx context.Context, input CompleteReservationInput) (*domain.Reservation, error) {
	now := uc.now()

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

	// 2. Load activity (needed for the "activity finished" check).
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

	// 3. Policy: the activity must have finished
	// (date + duration < now, strict). activity.IsFinished uses
	// date.Add(duration).Before(now), so a reservation for an
	// activity that ended exactly at `now` is NOT considered
	// finished; this matches the Cancel flow's strict semantics
	// for consistency.
	if !activity.IsFinished(now) {
		return nil, ErrActivityNotFinished
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
		// Same no-leak policy as Cancel/NoShow: surface as the
		// same ErrInvalidDog so the client cannot probe for the
		// existence of other users' dogs.
		return nil, ErrInvalidDog
	}

	// 5. Apply the status transition. The domain enforces
	// StatusConfirmed; any other state returns an error. We
	// translate that to ErrNotCompletable (409 not_completable).
	if err := reservation.Complete(); err != nil {
		return nil, ErrNotCompletable
	}

	// 6. Persist the new status. No pass refund: the session was
	// consumed at registration and the activity has been delivered.
	if err := uc.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
	}

	return reservation, nil
}
