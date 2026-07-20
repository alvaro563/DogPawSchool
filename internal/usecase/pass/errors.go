// Package pass contains the use cases for managing prepaid session
// passes (bonos). It depends only on the domain layer; persistence
// and HTTP concerns are injected via the PassRepository interface.
package pass

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

// ErrNotFound is returned by use cases when the requested pass does
// not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidUserID is returned when the supplied user_id does not
// resolve to an existing user. The handler maps it to 400.
var ErrInvalidUserID = errors.New("invalid user_id")
