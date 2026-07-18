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
