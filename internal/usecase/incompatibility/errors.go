package incompatibility

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required field: %s", e.Field)
}

func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

var (
	ErrNotFound      = errors.New("not found")
	ErrDuplicateName = errors.New("incompatibility name already exists")
	ErrInUse         = errors.New("incompatibility is in use by at least one dog")
)
