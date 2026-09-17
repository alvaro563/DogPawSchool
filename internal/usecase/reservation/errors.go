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

// ErrDogSizeMismatch is returned when the activity has a size
// restriction (MINI / MEDIUM / LARGE) and the candidate dog's
// SizeBracket does not match. Bypassed by admin override. The
// handler maps it to 400 with field="size_mismatch".
var ErrDogSizeMismatch = errors.New("dog size does not match activity target")

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

// ErrDuplicateReservationForDog is returned when the partial unique
// index uniq_reservation_dog_active fires at insert time. The index
// covers only active reservations (CONFIRMED, PENDING_TO_CONFIRM),
// so the only way to reach this error in normal flow is a concurrent
// race — two simultaneous registrations of the same dog to the same
// activity arriving between the ListByActivity pre-check and the
// INSERT. Cancelling and re-registering the same dog is explicitly
// supported: terminal statuses (CANCELLED_*, NO_SHOW, FORGIVEN,
// COMPLETED) are excluded from the index, so the user can recover
// from accidental removal by the admin without manual SQL.
// Maps to 409 duplicate_reservation.
var ErrDuplicateReservationForDog = errors.New("dog already booked for this activity")

// ErrDogPassOwnerMismatch is returned by RegisterReservationUseCase
// when adminOverride is true and the dog and the pass belong to
// different users. Admin authority lets a single operator act on
// behalf of any user, but it does not let them combine one user's
// dog with another user's pass: the booking would still attribute
// a pass session to a user that does not own the dog. Under
// normal (non-admin) flow the per-entity ownership checks
// already imply dog.UserID() == pass.UserID() == userID, so
// this sentinel is only reachable on the admin path. Maps to
// 400 dog_pass_owner_mismatch.
var ErrDogPassOwnerMismatch = errors.New("dog and pass must belong to the same user")

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

// ErrNotLateCancelled is returned by ForgiveReservationUseCase when
// the target reservation is not in StatusCancelledLate. Only that
// status can be transitioned to StatusForgiven; all others (already
// forgiven, still confirmed, cancelled in time, completed, no-show,
// pending) return this error. Maps to 409 not_late_cancelled.
var ErrNotLateCancelled = errors.New("reservation is not in a state that can be forgiven")

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

// SexNeuteredConflictError is returned by RegisterReservationUseCase
// when the candidate dog is a non-castrated male and one or more
// other non-castrated males are already holding a slot in the activity.
// Only the *blocking* sex/neutered conflicts are aggregated here —
// pending (non-blocking) cases are surfaced as StatusPendingToConfirm
// in the reservation itself, not as an error.
//
// The struct carries only stable domain data (the incoming dog and the
// list of dogs blocking the booking). Translating these into a
// user-facing message is the handler layer's responsibility — the
// domain and use case layers are language-neutral.
type SexNeuteredConflictError struct {
	IncomingDog  *domain.Dog
	BlockingDogs []*domain.Dog
	Conflicts    []domain.SexNeuteredConflict
}

func (e *SexNeuteredConflictError) Error() string {
	if e == nil || e.IncomingDog == nil {
		return "sex_neutered_conflict: incoming dog is nil"
	}
	return fmt.Sprintf("sex_neutered_conflict: %s blocked by %d intact male(s)",
		e.IncomingDog.Name(), len(e.BlockingDogs))
}
