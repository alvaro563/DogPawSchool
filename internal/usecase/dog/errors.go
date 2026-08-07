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

// ErrNotFound is returned by use cases when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotATrait is returned when an incompatibility ID references a row
// without a code (i.e., it is a trigger, not a trait).
var ErrNotATrait = errors.New("incompatibility is not a trait")

// ErrNotATrigger is returned when an incompatibility ID references a row
// without a target_trait_code (i.e., it is a trait, not a trigger).
var ErrNotATrigger = errors.New("incompatibility is not a trigger")

// ErrInvalidHeatForSex is returned by SetDogHeatUseCase when heat=true is
// attempted on a dog whose sex is not FEMALE. This is a business rule
// enforced at the use case layer (the DB does not constrain it).
var ErrInvalidHeatForSex = errors.New("heat can only be set on female dogs")
