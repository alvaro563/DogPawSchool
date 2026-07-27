package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
	reservationuc "dogpaw/internal/usecase/reservation"
)

// reservationNoShower is the contract the CloseActivity use case
// needs to mark individual reservations as no-show. Implemented by
// reservation.MarkReservationNoShowUseCase.
type reservationNoShower interface {
	Execute(ctx context.Context, input reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error)
}

// reservationCompleter is the contract the CloseActivity use case
// needs to mark individual reservations as completed. Implemented by
// reservation.CompleteReservationUseCase.
type reservationCompleter interface {
	Execute(ctx context.Context, input reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error)
}

// CloseActivityInput is the validated command to close an activity
// and batch-process its reservations. Fields are private: only
// NewCloseActivityInput can construct one.
type CloseActivityInput struct {
	activityID           int
	noShowReservationIDs []int
	now                  time.Time
}

func (in CloseActivityInput) ActivityID() int             { return in.activityID }
func (in CloseActivityInput) NoShowReservationIDs() []int { return in.noShowReservationIDs }
func (in CloseActivityInput) Now() time.Time              { return in.now }

// NewCloseActivityInput is the validating factory.
func NewCloseActivityInput(activityID int, noShowReservationIDs []int, now func() time.Time) (CloseActivityInput, error) {
	if activityID <= 0 {
		return CloseActivityInput{}, &ValidationError{Field: "activity_id"}
	}
	for i, id := range noShowReservationIDs {
		if id <= 0 {
			return CloseActivityInput{}, &ValidationError{Field: fmt.Sprintf("no_show_reservation_ids[%d]", i)}
		}
	}
	if now == nil {
		now = time.Now
	}
	// Deduplicate the no-show list.
	seen := make(map[int]struct{}, len(noShowReservationIDs))
	deduped := make([]int, 0, len(noShowReservationIDs))
	for _, id := range noShowReservationIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			deduped = append(deduped, id)
		}
	}
	return CloseActivityInput{
		activityID:           activityID,
		noShowReservationIDs: deduped,
		now:                  now(),
	}, nil
}

// MustNewCloseActivityInput panics on validation error. For tests.
func MustNewCloseActivityInput(activityID int, noShowReservationIDs []int, now func() time.Time) CloseActivityInput {
	in, err := NewCloseActivityInput(activityID, noShowReservationIDs, now)
	if err != nil {
		panic(err)
	}
	return in
}

// CloseActivityOutput is the result of a successful close.
type CloseActivityOutput struct {
	Activity *domain.Activity
}

// CloseActivityUseCase closes an activity: verifies the activity has
// finished, batch-processes every CONFIRMED reservation (marking them
// as no-show or completed), then marks the activity as closed. The
// entire flow runs inside a single transaction.
type CloseActivityUseCase struct {
	transactor      transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	reservationRepo domain.ReservationRepository
	noShower        reservationNoShower
	completer       reservationCompleter
	now             func() time.Time
}

func NewCloseActivityUseCase(
	transactor transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	noShower reservationNoShower,
	completer reservationCompleter,
	now func() time.Time,
) *CloseActivityUseCase {
	return &CloseActivityUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		reservationRepo: reservationRepo,
		noShower:        noShower,
		completer:       completer,
		now:             now,
	}
}

func (uc *CloseActivityUseCase) Execute(ctx context.Context, input CloseActivityInput) (CloseActivityOutput, error) {
	now := input.Now()
	uc.now = func() time.Time { return now }

	var output CloseActivityOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		a, err := uc.runInTx(txCtx, input)
		if err != nil {
			return err
		}
		output = CloseActivityOutput{Activity: a}
		return nil
	})
	return output, err
}

func (uc *CloseActivityUseCase) runInTx(ctx context.Context, input CloseActivityInput) (*domain.Activity, error) {
	now := uc.now()

	// 1. Load activity.
	activity, err := uc.activityRepo.GetByID(ctx, input.ActivityID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get activity %d: %w", input.ActivityID(), err)
	}
	if activity == nil {
		return nil, ErrNotFound
	}

	// 2. Policy: the activity must have finished.
	if !activity.IsFinished(now) {
		return nil, ErrNotFinished
	}

	// 3. Policy: the activity must not already be closed.
	if activity.IsClosed() {
		return nil, ErrAlreadyClosed
	}

	// 4. Load all CONFIRMED reservations for this activity.
	allReservations, err := uc.reservationRepo.ListByActivity(ctx, input.ActivityID())
	if err != nil {
		return nil, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}

	// 5. Build a set of CONFIRMED reservation IDs and validate
	// the no-show list against them.
	confirmedSet := make(map[int]*domain.Reservation, len(allReservations))
	for _, r := range allReservations {
		if r.Status() == domain.StatusConfirmed {
			confirmedSet[r.ID()] = r
		}
	}

	noShowSet := make(map[int]struct{}, len(input.NoShowReservationIDs()))
	for _, id := range input.NoShowReservationIDs() {
		if _, ok := confirmedSet[id]; !ok {
			// Check if it exists at all (maybe it belongs to
			// another activity or has a different status).
			exists := false
			for _, r := range allReservations {
				if r.ID() == id {
					exists = true
					if r.Status() != domain.StatusConfirmed {
						return nil, ErrReservationNotConfirmed
					}
					return nil, ErrReservationNotInActivity
				}
			}
			if !exists {
				return nil, ErrReservationNotFound
			}
		}
		noShowSet[id] = struct{}{}
	}

	// 6. Process each CONFIRMED reservation.
	for _, r := range confirmedSet {
		// Load the dog to get the owner's user ID for the
		// child use case's ownership check.
		dog, err := uc.dogRepo.GetByID(ctx, r.DogID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("dog %d not found: %w", r.DogID(), err)
			}
			return nil, fmt.Errorf("get dog %d: %w", r.DogID(), err)
		}
		if dog == nil {
			return nil, fmt.Errorf("dog %d not found", r.DogID())
		}

		if _, ok := noShowSet[r.ID()]; ok {
			in, err := reservationuc.NewMarkReservationNoShowInput(dog.UserID(), r.ID(), func() time.Time { return now })
			if err != nil {
				return nil, fmt.Errorf("build no-show input for reservation %d: %w", r.ID(), err)
			}
			if _, err := uc.noShower.Execute(ctx, in); err != nil {
				return nil, fmt.Errorf("mark no-show reservation %d: %w", r.ID(), err)
			}
		} else {
			in, err := reservationuc.NewCompleteReservationInput(dog.UserID(), r.ID(), func() time.Time { return now })
			if err != nil {
				return nil, fmt.Errorf("build complete input for reservation %d: %w", r.ID(), err)
			}
			if _, err := uc.completer.Execute(ctx, in); err != nil {
				return nil, fmt.Errorf("complete reservation %d: %w", r.ID(), err)
			}
		}
	}

	// 7. Close the activity.
	if err := activity.Close(); err != nil {
		return nil, ErrAlreadyClosed
	}

	// 8. Persist the closed state.
	if err := uc.activityRepo.Update(ctx, activity); err != nil {
		return nil, fmt.Errorf("update activity %d: %w", input.ActivityID(), err)
	}

	return activity, nil
}
