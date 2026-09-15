package activity

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
//
// The use case holds no mutable state: the clock travels with the
// input, so a single instance is safe to share across concurrent
// requests.
type CloseActivityUseCase struct {
	transactor      transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	reservationRepo domain.ReservationRepository
	noShower        reservationNoShower
	completer       reservationCompleter
}

func NewCloseActivityUseCase(
	transactor transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	noShower reservationNoShower,
	completer reservationCompleter,
) *CloseActivityUseCase {
	return &CloseActivityUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		reservationRepo: reservationRepo,
		noShower:        noShower,
		completer:       completer,
	}
}

func (uc *CloseActivityUseCase) Execute(ctx context.Context, input CloseActivityInput) (CloseActivityOutput, error) {
	var output CloseActivityOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		a, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = CloseActivityOutput{Activity: a}
		return nil
	})
	return output, err
}

func (uc *CloseActivityUseCase) runInTx(ctx context.Context, input CloseActivityInput, now time.Time) (*domain.Activity, error) {
	// 1. Load activity.
	activity, err := uc.activityRepo.GetByID(ctx, input.ActivityID(), 0, true)
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

	// 5. Index the CONFIRMED reservations. `confirmed` keeps a
	// deterministic (id-ascending) processing order: step 6 issues a
	// write per reservation, and iterating the map instead would
	// randomize the order in which those writes take row locks, which
	// risks a deadlock between two concurrent closes and makes tests
	// order-dependent.
	confirmedSet := make(map[int]*domain.Reservation, len(allReservations))
	confirmed := make([]*domain.Reservation, 0, len(allReservations))
	for _, r := range allReservations {
		if r.Status() == domain.StatusConfirmed {
			confirmedSet[r.ID()] = r
			confirmed = append(confirmed, r)
		}
	}
	sort.Slice(confirmed, func(i, j int) bool { return confirmed[i].ID() < confirmed[j].ID() })

	// Validate the no-show list against them. allReservations is
	// already scoped to this activity, so an id that is present there
	// but absent from confirmedSet can only be a non-CONFIRMED
	// reservation — there is no "belongs to another activity" case to
	// distinguish here.
	noShowSet := make(map[int]struct{}, len(input.NoShowReservationIDs()))
	for _, id := range input.NoShowReservationIDs() {
		if _, ok := confirmedSet[id]; !ok {
			for _, r := range allReservations {
				if r.ID() == id {
					return nil, ErrReservationNotConfirmed
				}
			}
			return nil, ErrReservationNotFound
		}
		noShowSet[id] = struct{}{}
	}

	// 6. Process each CONFIRMED reservation. The shared helper lives
	// at package scope so the standalone bulk-complete use case can
	// reuse it without duplicating the loop / sort / lock-order
	// rationale.
	if err := processConfirmedReservations(ctx, uc.dogRepo, confirmed, noShowSet, uc.noShower, uc.completer, now); err != nil {
		return nil, err
	}

	// 7. Close the activity. The domain error is wrapped rather than
	// replaced so errors.Is(err, ErrAlreadyClosed) still holds for the
	// handler while the log keeps the underlying reason.
	if err := activity.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAlreadyClosed, err)
	}

	// 8. Persist the closed state.
	if err := uc.activityRepo.Update(ctx, activity); err != nil {
		return nil, fmt.Errorf("update activity %d: %w", input.ActivityID(), err)
	}

	return activity, nil
}

// processConfirmedReservations walks every CONFIRMED reservation
// (already sorted id-ascending by the caller) and dispatches it to
// either noShower or completer. The deterministic id-ascending order
// avoids lock-order deadlocks across concurrent batch writers.
//
// noShowSet is the set of reservation IDs that must be marked
// NO_SHOW; the rest are COMPLETED. nil noShower disables the no-show
// branch (used by BulkCompleteReservations, which has no no-show
// half).
//
// The caller owns the transaction: this function does not commit or
// roll back. Errors are wrapped so the use case can attribute them to
// the specific reservation id in logs.
func processConfirmedReservations(
	ctx context.Context,
	dogRepo domain.DogRepository,
	confirmed []*domain.Reservation,
	noShowSet map[int]struct{},
	noShower reservationNoShower,
	completer reservationCompleter,
	now time.Time,
) error {
	for _, r := range confirmed {
		dog, err := dogRepo.GetByID(ctx, r.DogID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("dog %d not found: %w", r.DogID(), err)
			}
			return fmt.Errorf("get dog %d: %w", r.DogID(), err)
		}
		if dog == nil {
			return fmt.Errorf("dog %d not found", r.DogID())
		}

		if _, ok := noShowSet[r.ID()]; ok && noShower != nil {
			in, err := reservationuc.NewMarkReservationNoShowInput(dog.UserID(), r.ID(), func() time.Time { return now })
			if err != nil {
				return fmt.Errorf("build no-show input for reservation %d: %w", r.ID(), err)
			}
			if _, err := noShower.Execute(ctx, in); err != nil {
				return fmt.Errorf("mark no-show reservation %d: %w", r.ID(), err)
			}
		} else {
			in, err := reservationuc.NewCompleteReservationInput(dog.UserID(), r.ID(), func() time.Time { return now })
			if err != nil {
				return fmt.Errorf("build complete input for reservation %d: %w", r.ID(), err)
			}
			if _, err := completer.Execute(ctx, in); err != nil {
				return fmt.Errorf("complete reservation %d: %w", r.ID(), err)
			}
		}
	}
	return nil
}
