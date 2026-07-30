package auth

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

// ErrNotFound is returned by use cases when a requested resource does
// not exist. Handlers map this to 404.
var ErrNotFound = errors.New("not found")

// ErrInvalidCredentials is returned by LoginUseCase when the email is
// unknown or the password does not match the stored hash. Handlers map
// this to 401 without revealing which field was wrong (prevents user
// enumeration).
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserInactive is returned by LoginUseCase when the account exists
// and the password is correct but the user has been deactivated.
// Handlers map this to the same 401 response as ErrInvalidCredentials.
var ErrUserInactive = errors.New("user is inactive")

// ErrSamePassword is returned by ChangePasswordUseCase when the new
// password equals the current one. Handlers map this to 409 Conflict.
var ErrSamePassword = errors.New("new password matches the old one")
