package reservation

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ListActivityRosterInput is the validated command to fetch the
// class roster of a single activity: the activity itself plus the
// reservations partitioned into CONFIRMED and PENDING_TO_CONFIRM
// (the only two states that occupy a slot on the class day).
type ListActivityRosterInput struct {
	activityID int
}

func (in ListActivityRosterInput) ActivityID() int { return in.activityID }

// NewListActivityRosterInput validates the activity id.
func NewListActivityRosterInput(activityID int) (ListActivityRosterInput, error) {
	if activityID <= 0 {
		return ListActivityRosterInput{}, &ValidationError{Field: "activity_id"}
	}
	return ListActivityRosterInput{activityID: activityID}, nil
}

// MustNewListActivityRosterInput panics on validation error. For tests.
func MustNewListActivityRosterInput(activityID int) ListActivityRosterInput {
	in, err := NewListActivityRosterInput(activityID)
	if err != nil {
		panic(err)
	}
	return in
}

// ActivityRosterEntry is the flat shape the handler serializes for
// every attendee on the roster. It is intentionally minimal: only
// the fields the admin needs to take an action on the class day
// (confirm, reject, identify the dog+owner). Activity, pass, and
// timestamp fields are not exposed because they live on the
// activity and reservation aggregates in the response envelope.
//
// Reasons is the audit trail captured at booking time; it is
// populated only for PENDING_TO_CONFIRM entries (confirmed ones
// have no reason rows). When the admin opens the class day, they
// see exactly what triggered the pending escalation. Nil/empty
// means "no recorded reasons" — the read APIs should omit the
// field on the wire.
type ActivityRosterEntry struct {
	reservationID int
	dogID         int
	dogName       string
	ownerID       int
	ownerName     string
	reasons       []domain.PendingReason
}

func (e ActivityRosterEntry) ReservationID() int { return e.reservationID }
func (e ActivityRosterEntry) DogID() int         { return e.dogID }
func (e ActivityRosterEntry) DogName() string    { return e.dogName }
func (e ActivityRosterEntry) OwnerID() int       { return e.ownerID }
func (e ActivityRosterEntry) OwnerName() string  { return e.ownerName }

// Reasons returns the audit-trail reasons for this entry, in the
// original evaluation order. Returns nil when the entry has no
// recorded reasons (the read APIs should omit the field on the
// wire).
func (e ActivityRosterEntry) Reasons() []domain.PendingReason {
	return e.reasons
}

// NewActivityRosterEntry is the only path to construct an entry from
// outside the use case (handler tests, future serializers). It does
// NOT validate the fields: the use case is the only authority on
// roster semantics; the constructor is a plain data holder.
func NewActivityRosterEntry(reservationID, dogID, ownerID int, dogName, ownerName string, reasons []domain.PendingReason) ActivityRosterEntry {
	return ActivityRosterEntry{
		reservationID: reservationID,
		dogID:         dogID,
		dogName:       dogName,
		ownerID:       ownerID,
		ownerName:     ownerName,
		reasons:       reasons,
	}
}

// ListActivityRosterOutput is the response envelope. Confirmed and
// Pending are kept as separate slices so the handler can serialize
// them directly without any client-side filtering. Both slices are
// ordered by reservation created_at ASC (the same order used by
// ListByActivityView in the underlying query).
type ListActivityRosterOutput struct {
	Activity  *domain.Activity
	Confirmed []ActivityRosterEntry
	Pending   []ActivityRosterEntry
}

// ListActivityRosterUseCase builds the admin "class roster" view of
// a single activity. It collapses three SQL round-trips into one
// use-case call:
//
//  1. Load the activity (404 if it does not exist).
//  2. Load every reservation view for the activity.
//  3. Resolve every distinct owner id in a single batched user
//     query.
//
// The partitioning by status (CONFIRMED vs PENDING_TO_CONFIRM) and
// the owner-name resolution happen server-side so the client can
// render the page with a single fetch and zero filtering.
type ListActivityRosterUseCase struct {
	activityRepo    domain.ActivityRepository
	reservationRepo domain.ReservationRepository
	userRepo        domain.UserRepository
}

func NewListActivityRosterUseCase(
	activityRepo domain.ActivityRepository,
	reservationRepo domain.ReservationRepository,
	userRepo domain.UserRepository,
) *ListActivityRosterUseCase {
	return &ListActivityRosterUseCase{
		activityRepo:    activityRepo,
		reservationRepo: reservationRepo,
		userRepo:        userRepo,
	}
}

// Execute returns the activity, the confirmed roster and the pending
// roster. Only the two statuses that occupy a slot on the class day
// are returned (CONFIRMED, PENDING_TO_CONFIRM); every other status
// (CANCELLED_*, COMPLETED, NO_SHOW, FORGIVEN) is intentionally
// dropped because it is irrelevant to a class-day roster and would
// clutter the response. Order is created_at ASC for both slices.
//
// Pending reasons (the audit trail captured at booking time) are
// loaded in a single batched query along with the owner name
// resolution. Confirmed entries always get a nil reasons slice;
// the wire layer omits the field entirely for them.
func (uc *ListActivityRosterUseCase) Execute(ctx context.Context, input ListActivityRosterInput) (ListActivityRosterOutput, error) {
	// Admin viewer: this endpoint is served only by the admin route.
	activity, err := uc.activityRepo.GetByID(ctx, input.ActivityID(), 0, true)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ListActivityRosterOutput{}, ErrInvalidActivity
		}
		return ListActivityRosterOutput{}, fmt.Errorf("get activity %d: %w", input.ActivityID(), err)
	}
	if activity == nil {
		return ListActivityRosterOutput{}, ErrInvalidActivity
	}

	// 100 is enough for any realistic class (max_capacity is small in
	// this domain — 10-20 typical). We avoid pagination on a roster
	// view because the admin needs every attendee on one page.
	views, err := uc.reservationRepo.ListByActivityView(ctx, input.ActivityID(), 100, 0)
	if err != nil {
		return ListActivityRosterOutput{}, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}

	confirmed := make([]ActivityRosterEntry, 0)
	pending := make([]ActivityRosterEntry, 0)
	ownerIDSet := make(map[int]struct{}, len(views))
	pendingResIDs := make([]int, 0)
	for _, view := range views {
		switch view.Status() {
		case domain.StatusConfirmed:
			confirmed = append(confirmed, ActivityRosterEntry{
				reservationID: view.ID(),
				dogID:         view.DogID(),
				dogName:       view.DogName(),
				ownerID:       view.DogUserID(),
			})
			ownerIDSet[view.DogUserID()] = struct{}{}
		case domain.StatusPendingToConfirm:
			pending = append(pending, ActivityRosterEntry{
				reservationID: view.ID(),
				dogID:         view.DogID(),
				dogName:       view.DogName(),
				ownerID:       view.DogUserID(),
			})
			ownerIDSet[view.DogUserID()] = struct{}{}
			pendingResIDs = append(pendingResIDs, view.ID())
		default:
			// CANCELLED_*, COMPLETED, NO_SHOW, FORGIVEN: not on the
			// class-day roster.
		}
	}

	// Resolve owner names in a single batched query. The use case
	// does not fail the whole call when an owner cannot be
	// resolved — the entry stays in the roster with an empty name.
	// This keeps the page useful even if a dog has been orphaned by
	// a user deletion (deactivated, soft-deleted, etc.).
	ownerIDs := make([]int, 0, len(ownerIDSet))
	for id := range ownerIDSet {
		ownerIDs = append(ownerIDs, id)
	}
	if len(ownerIDs) > 0 {
		users, err := uc.userRepo.GetByIDs(ctx, ownerIDs)
		if err != nil {
			return ListActivityRosterOutput{}, fmt.Errorf("resolve owners for activity %d: %w", input.ActivityID(), err)
		}
		nameByID := make(map[int]string, len(users))
		for _, u := range users {
			if u != nil {
				nameByID[u.ID()] = u.Name()
			}
		}
		for i := range confirmed {
			confirmed[i].ownerName = nameByID[confirmed[i].ownerID]
		}
		for i := range pending {
			pending[i].ownerName = nameByID[pending[i].ownerID]
		}
	}

	// Pending reasons: one batched query scoped to the pending set.
	// Confirmed entries are excluded from this query because they
	// can never have reason rows.
	if len(pendingResIDs) > 0 {
		reasonsByRes, err := uc.reservationRepo.ListPendingReasonsByReservations(ctx, pendingResIDs)
		if err != nil {
			return ListActivityRosterOutput{}, fmt.Errorf("resolve pending reasons for activity %d: %w", input.ActivityID(), err)
		}
		for i := range pending {
			if rs, ok := reasonsByRes[pending[i].reservationID]; ok {
				pending[i].reasons = rs
			}
		}
	}

	return ListActivityRosterOutput{
		Activity:  activity,
		Confirmed: confirmed,
		Pending:   pending,
	}, nil
}
