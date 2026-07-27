// Package activity contains the use cases for managing school
// activities (classes, routes, individual sessions, extras). It
// depends only on the domain layer; the persistence and HTTP
// concerns are injected via the ActivityRepository interface.
package activity

import (
	"errors"
	"fmt"
)

// ValidationError is returned by use cases when a required field is
// missing or a value is invalid. The handler layer maps it to a 400
// response.
type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required field: %s", e.Field)
}

// IsValidationError reports whether err is a *ValidationError from
// this package.
func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

// ErrNotFound is returned by use cases when the requested activity
// does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotFinished is returned by CloseActivity when the activity's
// date + duration is at or after now (the activity has not ended
// yet). Maps to 400 activity_not_finished.
var ErrNotFinished = errors.New("activity has not finished yet")

// ErrAlreadyClosed is returned by CloseActivity when the activity
// is already closed. Maps to 409 already_closed.
var ErrAlreadyClosed = errors.New("activity is already closed")

// ErrReservationNotFound is returned by CloseActivity when a
// no_show_reservation_id does not resolve to an existing CONFIRMED
// reservation for this activity. Maps to 400 invalid_reservation_id.
var ErrReservationNotFound = errors.New("reservation not found for this activity")

// ErrReservationNotInActivity is returned by CloseActivity when a
// no_show_reservation_id belongs to a different activity. Maps to
// 400 invalid_reservation_id.
var ErrReservationNotInActivity = errors.New("reservation belongs to a different activity")

// ErrReservationNotConfirmed is returned by CloseActivity when a
// reservation in the no_show list is not in CONFIRMED status.
// Maps to 409 reservation_not_confirmed.
var ErrReservationNotConfirmed = errors.New("reservation is not in CONFIRMED status")
