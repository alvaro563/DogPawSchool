// Package reservation contains the use cases for booking reservations:
// a dog paid from a pass joins an activity. It depends only on the
// domain layer; persistence is injected via repository interfaces.
package reservation

import (
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ValidationError is returned by use cases when a required field is
// missing or a value is invalid. The handler layer maps it to a 400
// response.
type ValidationError struct {
	Field string
}

func (validationError *ValidationError) Error() string {
	return fmt.Sprintf("missing required field: %s", validationError.Field)
}

// IsValidationError reports whether err is a *ValidationError from
// this package.
func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

// ErrNotFound is returned by use cases when the requested reservation
// does not exist. Mirrors the pattern in the other use case
// packages.
var ErrNotFound = errors.New("not found")

// ErrInvalidActivity is returned when the activity_id path/body field
// does not resolve to an existing activity. The handler maps it to
// 400 invalid_activity_id.
var ErrInvalidActivity = errors.New("invalid activity_id")

// ErrActivityInPast is returned when the booking targets an activity
// whose date is already in the past. The handler maps it to 400
// activity_in_past.
var ErrActivityInPast = errors.New("activity is in the past")

// ErrActivityFull is returned when the activity has reached its
// max_capacity with CONFIRMED bookings. The handler maps it to 409
// activity_full.
var ErrActivityFull = errors.New("activity is full")

// ErrInvalidDog is returned when the dog_id does not resolve to an
// existing dog OR the dog belongs to a different user. Both cases
// are intentionally surfaced as the same error to avoid leaking the
// existence of other users' dogs. The handler maps it to 400
// invalid_dog_id.
var ErrInvalidDog = errors.New("invalid dog_id")

// ErrInvalidPass is returned when the pass_id does not resolve to an
// existing pass OR the pass belongs to a different user. Same
// rationale as ErrInvalidDog. The handler maps it to 400
// invalid_pass_id.
var ErrInvalidPass = errors.New("invalid pass_id")

// ErrPassExhausted is returned when the pass has no remaining
// sessions. The handler maps it to 400 pass_exhausted.
var ErrPassExhausted = errors.New("pass has no remaining sessions")

// ErrPassExpired is returned when the pass has an expiry in the past.
// The handler maps it to 400 pass_expired.
var ErrPassExpired = errors.New("pass has expired")

// ErrInvalidReservation is returned by Cancel when the reservation_id
// does not resolve to an existing reservation. The handler maps it to
// 400 invalid_reservation_id.
var ErrInvalidReservation = errors.New("invalid reservation_id")

// ErrAlreadyCancelled is returned by Cancel when the reservation is
// not in StatusConfirmed (i.e., it has already been cancelled, was
// completed, marked no-show, or forgiven). The handler maps it to
// 409 already_cancelled.
var ErrAlreadyCancelled = errors.New("reservation is not in a cancellable state")

// ErrReservationNotOwned is returned by Get when the reservation
// exists but its dog is owned by a different user than the one in
// the URL path. Intentionally surfaced as 404 not_found (not 403) so
// we do not leak the existence of other users' reservations. The
// handler maps it to 404.
var ErrReservationNotOwned = errors.New("reservation is not owned by this user")

// ErrInvalidStatusFilter is returned by ListByUserReservations when
// the optional status query param is set to a value that is not a
// valid ReservationStatus enum value. The handler maps it to 400
// invalid_status.
var ErrInvalidStatusFilter = errors.New("invalid status filter")

// ErrDuplicateReservationForDog is returned when the
// UNIQUE (activity_id, dog_id) constraint fires at insert time.
// This is the only way to detect a duplicate booking when the
// existing row is not in StatusConfirmed (e.g., the user previously
// cancelled in time and is trying to rebook — which we want to
// allow; the constraint is enforced to keep the history clean, so
// we surface this as 409 and let the user cancel+rebook
// explicitly).
var ErrDuplicateReservationForDog = errors.New("dog already booked for this activity")

// ErrActivityNotStarted is returned by MarkReservationNoShow when
// the activity's date is at or after now. The policy is that only
// already-started activities can be marked no-show, so the slot
// is definitively "missed". Maps to 400 activity_not_started.
var ErrActivityNotStarted = errors.New("activity has not started yet")

// ErrNotCancellable is returned by MarkReservationNoShow when the
// reservation is not in StatusConfirmed (i.e., it is already
// cancelled, completed, no-show, or forgiven). Maps to 409
// not_cancellable. The wire key is intentionally distinct from
// already_cancelled (used by Cancel) to give clients a precise
// distinction between the two action errors.
var ErrNotCancellable = errors.New("reservation is not in a state that allows no-show")

// ErrActivityNotFinished is returned by CompleteReservation when
// the activity's date + duration is at or after now (the activity
// has not ended yet). Maps to 400 activity_not_finished.
var ErrActivityNotFinished = errors.New("activity has not finished yet")

// ErrNotCompletable is returned by CompleteReservation when the
// reservation is not in StatusConfirmed (i.e., it is already
// cancelled, completed, no-show, or forgiven). Maps to 409
// not_completable.
var ErrNotCompletable = errors.New("reservation is not in a state that allows completion")

// ErrNotPending is returned by ConfirmPendingReservation and
// RejectPendingReservation when the reservation is not in
// StatusPendingToConfirm. Maps to 409 not_pending.
var ErrNotPending = errors.New("reservation is not pending to confirm")

// IncompatibleDogsError is returned by RegisterReservationUseCase when
// the candidate dog and one or more dogs already holding a slot in the
// activity present a trigger->trait compatibility collision. It carries
// every detected conflict so the handler can surface them. Maps to 409
// dog_incompatible. The severity resolution (block vs hold-pending) is
// decided inside the register flow, not here.
type IncompatibleDogsError struct {
	Conflicts []domain.CompatibilityConflict
}

func (e *IncompatibleDogsError) Error() string {
	return fmt.Sprintf("incompatible dogs: %d conflict(s)", len(e.Conflicts))
}
