package user

import (
	"errors"
	"fmt"
)

// ValidationError is returned by the use case input factories when a
// required field is missing or invalid. The Field name mirrors the
// request DTO binding so handlers can map it back to a 400 response.
type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required field: %s", e.Field)
}

// IsValidationError reports whether err is or wraps a *ValidationError.
func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

// ErrNotFound is returned by use cases when a requested user does not
// exist. Handlers map this to 404.
var ErrNotFound = errors.New("not found")
