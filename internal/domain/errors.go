package domain

import "errors"

// Domain-level persistence sentinels. Repository implementations
// (internal/repository/postgres) translate their driver-specific errors
// into these so that use cases and handlers never import the
// infrastructure package. This keeps the dependency rule intact: outer
// layers depend on the domain, never on postgres.
var (
	// ErrNotFound is returned when a requested row does not exist. It is
	// shared by every aggregate; callers that need to distinguish which
	// aggregate was missing do so by context (which repo they called).
	ErrNotFound = errors.New("not found")

	// ErrDuplicateEntry is returned when an insert violates a uniqueness
	// rule that is not surfaced through a more specific sentinel below.
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrDuplicatePassport is returned when a dog insert/update violates
	// the UNIQUE passport constraint.
	ErrDuplicatePassport = errors.New("passport already exists")

	// ErrInvalidUserReference is returned when an insert references a
	// user_id that does not exist (FK violation on the owner).
	ErrInvalidUserReference = errors.New("user_id does not exist")

	// ErrIncompatibilityInUse is returned when deleting an incompatibility
	// that is still attached to at least one dog.
	ErrIncompatibilityInUse = errors.New("incompatibility in use")

	// ErrDuplicateIncompatibilityName is returned when an insert/update
	// violates the UNIQUE incompatibility name constraint.
	ErrDuplicateIncompatibilityName = errors.New("incompatibility name already exists")

	// ErrDuplicateReservation is returned when a reservation insert
	// violates the UNIQUE (activity_id, dog_id) constraint: the same dog
	// is already booked into that activity.
	ErrDuplicateReservation = errors.New("dog already booked for this activity")
)
