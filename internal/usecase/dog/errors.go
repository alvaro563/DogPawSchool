package dog

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
