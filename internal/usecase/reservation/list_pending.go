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

// NewPendingReservationEntry is the only path to construct an
// entry from outside the use case (handler tests, future
// serializers). It does NOT validate fields: the use case is the
// only authority on roster semantics.
func NewPendingReservationEntry(
	reservationID, dogID, ownerID, activityID int,
	dogName, ownerName, activityName, activityLocation string,
	activityDate time.Time,
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
func (uc *ListPendingReservationsUseCase) Execute(ctx context.Context, input ListPendingReservationsInput) (ListPendingReservationsOutput, error) {
	views, err := uc.reservationRepo.ListPendingView(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListPendingReservationsOutput{}, fmt.Errorf("list pending reservations: %w", err)
	}

	pending := make([]PendingReservationEntry, 0, len(views))
	ownerIDSet := make(map[int]struct{}, len(views))
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

	return ListPendingReservationsOutput{Pending: pending}, nil
}
