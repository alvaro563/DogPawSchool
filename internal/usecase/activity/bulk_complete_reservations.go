package activity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"dogpaw/internal/domain"
)

// BulkCompleteReservationsInput is the validated command to mark
// every CONFIRMED reservation of an activity as COMPLETED in a single
// transaction. CANCELLED / NO_SHOW / FORGIVEN / COMPLETED /
// PENDING_TO_CONFIRM reservations are NOT touched — the latter is
// rejected outright (the admin must resolve each pending
// individually first).
type BulkCompleteReservationsInput struct {
	activityID int
	now        time.Time
}

func (in BulkCompleteReservationsInput) ActivityID() int { return in.activityID }
func (in BulkCompleteReservationsInput) Now() time.Time  { return in.now }

// NewBulkCompleteReservationsInput is the validating factory. now
// defaults to time.Now when nil (so handlers can pass nil for the
// runtime injection pattern).
func NewBulkCompleteReservationsInput(activityID int, now func() time.Time) (BulkCompleteReservationsInput, error) {
	if activityID <= 0 {
		return BulkCompleteReservationsInput{}, &ValidationError{Field: "id"}
	}
	if now == nil {
		now = time.Now
	}
	return BulkCompleteReservationsInput{activityID: activityID, now: now()}, nil
}

// MustNewBulkCompleteReservationsInput panics on validation error.
// For tests.
func MustNewBulkCompleteReservationsInput(activityID int, now func() time.Time) BulkCompleteReservationsInput {
	in, err := NewBulkCompleteReservationsInput(activityID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// BulkCompleteReservationsOutput reports the action's effect on the
// database. CompletedCount is the number of reservations that
// transitioned to COMPLETED in this call; Closed is always true on
// success (this use case is a "complete AND close" action — the
// activity is closed as part of the same transaction). On a
// pre-closed activity, the only effect is the re-assertion of
// closed=true (the loop iterates an empty CONFIRMED set).
type BulkCompleteReservationsOutput struct {
	ActivityID     int
	CompletedCount int
	Closed         bool
}

// BulkCompleteReservationsUseCase walks every CONFIRMED reservation
// of the activity, marks it COMPLETED, and closes the activity,
// atomically. The "close" effect runs only when the activity is not
// already closed; calling this on a closed activity is a no-op for
// the close step (the loop yields an empty CONFIRMED slice anyway).
type BulkCompleteReservationsUseCase struct {
	transactor      transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	reservationRepo domain.ReservationRepository
	completer       reservationCompleter
}

func NewBulkCompleteReservationsUseCase(
	transactor transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	completer reservationCompleter,
) *BulkCompleteReservationsUseCase {
	return &BulkCompleteReservationsUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		reservationRepo: reservationRepo,
		completer:       completer,
	}
}

func (uc *BulkCompleteReservationsUseCase) Execute(ctx context.Context, in BulkCompleteReservationsInput) (BulkCompleteReservationsOutput, error) {
	var out BulkCompleteReservationsOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		activity, count, closed, err := uc.runInTx(txCtx, in)
		if err != nil {
			return err
		}
		out.ActivityID = activity.ID()
		out.CompletedCount = count
		out.Closed = closed
		return nil
	})
	return out, err
}

// runInTx is the policy body. Returns the activity plus the number
// of reservations that transitioned to COMPLETED in this call plus
// whether the activity is closed at the end (always true on success).
func (uc *BulkCompleteReservationsUseCase) runInTx(ctx context.Context, in BulkCompleteReservationsInput) (*domain.Activity, int, bool, error) {
	// 1. Load activity. A closed activity is NOT an error here: bulk-
	// completion is a "repair" tool for the admin; calling it on an
	// activity that Close already finished is a no-op for the
	// reservation loop (no CONFIRMED rows left) and we skip the close
	// step because IsClosed() is already true.
	activity, err := uc.activityRepo.GetByID(ctx, in.ActivityID(), 0, true)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, 0, false, ErrNotFound
		}
		return nil, 0, false, fmt.Errorf("get activity %d: %w", in.ActivityID(), err)
	}
	if activity == nil {
		return nil, 0, false, ErrNotFound
	}

	// 2. Policy: the activity must have already finished.
	if !activity.IsFinished(in.now) {
		return nil, 0, false, ErrNotFinished
	}

	// 3. Load every reservation for this activity (created_at ASC per
	// the repo contract), then partition into CONFIRMED vs
	// PENDING_TO_CONFIRM.
	all, err := uc.reservationRepo.ListByActivity(ctx, in.ActivityID())
	if err != nil {
		return nil, 0, false, fmt.Errorf("list reservations for activity %d: %w", in.ActivityID(), err)
	}

	confirmed := make([]*domain.Reservation, 0, len(all))
	var pendingCount int
	for _, r := range all {
		switch r.Status() {
		case domain.StatusConfirmed:
			confirmed = append(confirmed, r)
		case domain.StatusPendingToConfirm:
			pendingCount++
		}
	}

	// 4. Policy: refuse if there are pending reservations. Admins
	// must resolve each (confirm or reject) before running this batch.
	if pendingCount > 0 {
		return nil, 0, false, ErrPendingToConfirmExists
	}

	// 5. Deterministic id-ascending order. See the rationale comment
	// on the helper: random order would risk lock-order deadlocks
	// between two concurrent bulk-completions.
	sort.Slice(confirmed, func(i, j int) bool { return confirmed[i].ID() < confirmed[j].ID() })

	// 6. Reuse the per-reservation loop shared with CloseActivity.
	// noShower=nil disables the no-show branch entirely (noShowSet is
	// empty in this use case). An empty `confirmed` slice is a valid
	// no-op (returns nil).
	if err := processConfirmedReservations(ctx, uc.dogRepo, confirmed, nil, nil, uc.completer, in.now); err != nil {
		return nil, 0, false, err
	}

	// 7. Close the activity in the same transaction. Idempotent:
	// already-closed activities keep their closed state and we skip
	// the Update to avoid an "already closed" domain error.
	if !activity.IsClosed() {
		if err := activity.Close(); err != nil {
			return nil, 0, false, fmt.Errorf("close activity: %w", err)
		}
		if err := uc.activityRepo.Update(ctx, activity); err != nil {
			return nil, 0, false, fmt.Errorf("update activity %d: %w", in.ActivityID(), err)
		}
	}

	return activity, len(confirmed), true, nil
}
