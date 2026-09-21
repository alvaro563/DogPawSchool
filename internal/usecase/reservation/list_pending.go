package reservation

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ListPendingReservationsInput is the validated command to fetch
// every reservation awaiting admin approval (StatusPendingToConfirm).
// The slice is paginated server-side; the admin dashboard's "pending"
// page uses the maximum (100) because it expects to render every
// pending entry in a single triage view.
type ListPendingReservationsInput struct {
	limit  int
	offset int
}

func (in ListPendingReservationsInput) Limit() int  { return in.limit }
func (in ListPendingReservationsInput) Offset() int { return in.offset }

// NewListPendingReservationsInput normalizes pagination. The
// status filter is implicit (PENDING_TO_CONFIRM) so this factory
// takes no entity-level fields; error is always nil.
func NewListPendingReservationsInput(limit, offset int) (ListPendingReservationsInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListPendingReservationsInput{limit: limit, offset: offset}, nil
}

// MustNewListPendingReservationsInput panics on error. For tests.
func MustNewListPendingReservationsInput(limit, offset int) ListPendingReservationsInput {
	in, err := NewListPendingReservationsInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// PendingReservationEntry is the flat shape the handler serializes
// for every reservation awaiting admin approval. It carries the
// minimum fields the "pending" page needs to render a row with
// owner name and a pair of confirm/reject buttons. Time fields
// live on the wire response; they are not exposed as separate
// accessors because the page does not need them outside of the
// pre-serialized envelope.
//
// Reasons is the audit trail captured at booking time — when the
// admin opens the page they see exactly what triggered the
// PENDING_TO_CONFIRM escalation. Nil/empty means the reservation
// has no recorded reasons (the read APIs should omit the field).
type PendingReservationEntry struct {
	reservationID    int
	dogID            int
	dogName          string
	ownerID          int
	ownerName        string
	activityID       int
	activityName     string
	activityDate     time.Time
	activityLocation string
	reasons          []domain.PendingReason
}

func (e PendingReservationEntry) ReservationID() int    { return e.reservationID }
func (e PendingReservationEntry) DogID() int            { return e.dogID }
func (e PendingReservationEntry) DogName() string      { return e.dogName }
func (e PendingReservationEntry) OwnerID() int         { return e.ownerID }
func (e PendingReservationEntry) OwnerName() string    { return e.ownerName }
func (e PendingReservationEntry) ActivityID() int       { return e.activityID }
func (e PendingReservationEntry) ActivityName() string { return e.activityName }
func (e PendingReservationEntry) ActivityDate() time.Time {
	return e.activityDate
}
func (e PendingReservationEntry) ActivityLocation() string {
	return e.activityLocation
}

// Reasons returns the audit-trail reasons for this entry, in the
// original evaluation order. Returns nil when the reservation has no
// recorded reasons (the read APIs should omit the field on the wire).
func (e PendingReservationEntry) Reasons() []domain.PendingReason {
	return e.reasons
}

// NewPendingReservationEntry is the only path to construct an
// entry from outside the use case (handler tests, future
// serializers). It does NOT validate fields: the use case is the
// only authority on roster semantics.
func NewPendingReservationEntry(
	reservationID, dogID, ownerID, activityID int,
	dogName, ownerName, activityName, activityLocation string,
	activityDate time.Time,
	reasons []domain.PendingReason,
) PendingReservationEntry {
	return PendingReservationEntry{
		reservationID:    reservationID,
		dogID:            dogID,
		dogName:          dogName,
		ownerID:          ownerID,
		ownerName:        ownerName,
		activityID:       activityID,
		activityName:     activityName,
		activityDate:     activityDate,
		activityLocation: activityLocation,
		reasons:          reasons,
	}
}

// ListPendingReservationsOutput carries the resulting entries,
// ordered by created_at ASC (oldest first — the order in which the
// admin should triage them).
type ListPendingReservationsOutput struct {
	Pending []PendingReservationEntry
}

// ListPendingReservationsUseCase returns every reservation awaiting
// admin approval, with the dog owner's id and name resolved in a
// single batched user query. Server-side partitioning keeps the
// client thin: it receives a ready-to-render slice and never has
// to fetch user data on its own.
//
// Same ordering decision as the activity roster: oldest first, so
// the admin triages the bookings that have been waiting the
// longest.
type ListPendingReservationsUseCase struct {
	reservationRepo domain.ReservationRepository
	userRepo        domain.UserRepository
}

func NewListPendingReservationsUseCase(
	reservationRepo domain.ReservationRepository,
	userRepo domain.UserRepository,
) *ListPendingReservationsUseCase {
	return &ListPendingReservationsUseCase{
		reservationRepo: reservationRepo,
		userRepo:        userRepo,
	}
}

// Execute returns every reservation in StatusPendingToConfirm.
// Like the activity roster, owner names are resolved in a single
// batched query (skipped when there are no entries). Missing owners
// leave the entry with an empty OwnerName rather than failing the
// whole call — the page remains useful even if an owner has been
// soft-deleted between the booking and the triage.
//
// Reasons (the audit trail captured at booking time) are loaded in
// a second batched query, also skipped when there are no entries.
// Reservations with no recorded reasons get an empty slice rather
// than a nil, so the wire format is deterministic.
func (uc *ListPendingReservationsUseCase) Execute(ctx context.Context, input ListPendingReservationsInput) (ListPendingReservationsOutput, error) {
	views, err := uc.reservationRepo.ListPendingView(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListPendingReservationsOutput{}, fmt.Errorf("list pending reservations: %w", err)
	}

	pending := make([]PendingReservationEntry, 0, len(views))
	ownerIDSet := make(map[int]struct{}, len(views))
	reservationIDs := make([]int, 0, len(views))
	for _, view := range views {
		pending = append(pending, PendingReservationEntry{
			reservationID:    view.ID(),
			dogID:            view.DogID(),
			dogName:          view.DogName(),
			ownerID:          view.DogUserID(),
			activityID:       view.ActivityID(),
			activityName:     view.ActivityName(),
			activityDate:     view.ActivityDate(),
			activityLocation: view.ActivityLocation(),
		})
		ownerIDSet[view.DogUserID()] = struct{}{}
		reservationIDs = append(reservationIDs, view.ID())
	}

	if len(ownerIDSet) > 0 {
		ownerIDs := make([]int, 0, len(ownerIDSet))
		for id := range ownerIDSet {
			ownerIDs = append(ownerIDs, id)
		}
		users, err := uc.userRepo.GetByIDs(ctx, ownerIDs)
		if err != nil {
			return ListPendingReservationsOutput{}, fmt.Errorf("resolve owners for pending reservations: %w", err)
		}
		nameByID := make(map[int]string, len(users))
		for _, u := range users {
			if u != nil {
				nameByID[u.ID()] = u.Name()
			}
		}
		for i := range pending {
			pending[i].ownerName = nameByID[pending[i].ownerID]
		}
	}

	// Reasons: one batched query for the whole page. Reservations
	// without reasons are absent from the map; we leave the slice
	// nil so the handler omits the wire field.
	reasonsByRes, err := uc.reservationRepo.ListPendingReasonsByReservations(ctx, reservationIDs)
	if err != nil {
		return ListPendingReservationsOutput{}, fmt.Errorf("resolve pending reasons: %w", err)
	}
	for i := range pending {
		if rs, ok := reasonsByRes[pending[i].reservationID]; ok {
			pending[i].reasons = rs
		}
	}

	return ListPendingReservationsOutput{Pending: pending}, nil
}
