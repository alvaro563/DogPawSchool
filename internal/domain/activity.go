package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ActivityType distinguishes the four kinds of school activities.
type ActivityType string

const (
	TypeSocialization ActivityType = "SOCIALIZATION_GROUP"
	TypeRoute         ActivityType = "ROUTE"
	TypeIndividual    ActivityType = "INDIVIDUAL_CLASS"
	TypeExtra         ActivityType = "EXTRA"
)

// IsValid reports whether the value is a recognized ActivityType.
func (activityType ActivityType) IsValid() bool {
	switch activityType {
	case TypeSocialization, TypeRoute, TypeIndividual, TypeExtra:
		return true
	}
	return false
}

// ErrDogRequiredForIndividual is returned by NewActivity when the
// caller asks for an INDIVIDUAL_CLASS but does not provide a target
// dog. Group classes (SOCIALIZATION_GROUP, ROUTE) and EXTRA leave
// dogID nil.
var ErrDogRequiredForIndividual = errors.New("individual class requires a target dog")

// ErrSizeTargetNotApplicable is returned by NewActivity when a size
// target is supplied for a type that does not support it
// (INDIVIDUAL_CLASS or EXTRA). Size targets only make sense for
// group activities (SOCIALIZATION_GROUP, ROUTE).
var ErrSizeTargetNotApplicable = errors.New("size target only applies to SOCIALIZATION_GROUP and ROUTE")

// ErrInvalidSizeTarget is returned when sizeTarget is non-nil but
// points to an unrecognised SizeBracket (e.g. UNKNOWN).
var ErrInvalidSizeTarget = errors.New("invalid size target")

// Activity is a scheduled school session: a class, a route, an
// individual session, or an extra event. Dogs are booked into Activities
// via Reservation.
//
// dogID is *int (nullable pointer) because only INDIVIDUAL_CLASS
// activities are targeted at a specific dog; group activities keep
// dogID == nil so that the visibility filter can show them to every
// user (the LEFT JOIN on dogs yields user_id NULL for those rows and
// the WHERE short-circuits on a.dog_id IS NULL).
//
// sizeTarget is *SizeBracket (nullable pointer) because only
// SOCIALIZATION_GROUP and ROUTE may restrict the booking to a single
// size bracket (MINI / MEDIUM / LARGE). A nil sizeTarget means "all
// sizes welcome" — the reservation use case does not filter on dog
// size when this field is nil. INDIVIDUAL_CLASS and EXTRA always
// have nil sizeTarget.
type Activity struct {
	id              int
	name            string
	description     string
	activityType    ActivityType
	maxCapacity     int
	location        string
	durationInHours int
	date            time.Time
	closed          bool
	dogID           *int
	sizeTarget      *SizeBracket
}

// NewActivity creates an Activity with validated fields.
//
// dogID is the target dog id, nil for group classes. Required when
// activityType == TypeIndividual (enforced by the domain and
// redundantly by a CHECK constraint at the DB layer).
//
// sizeTarget restricts bookings to a single size bracket for
// SOCIALIZATION_GROUP and ROUTE activities. Pass nil for "all
// sizes" (the default). Passing a non-nil sizeTarget for
// INDIVIDUAL_CLASS or EXTRA returns ErrSizeTargetNotApplicable.
// UNKNOWN is rejected because no real dog is meant to land there.
func NewActivity(id int, name, description, location string, activityType ActivityType, maxCapacity, durationInHours int, date time.Time, dogID *int, sizeTarget *SizeBracket) (*Activity, error) {
	if id < 0 {
		return nil, fmt.Errorf("activity: id must not be negative")
	}
	if name == "" {
		return nil, fmt.Errorf("activity: name must not be empty")
	}
	if location == "" {
		return nil, fmt.Errorf("activity: location must not be empty")
	}
	if !activityType.IsValid() {
		return nil, fmt.Errorf("activity: invalid activityType %q", activityType)
	}
	if maxCapacity <= 0 {
		return nil, fmt.Errorf("activity: maxCapacity must be greater than 0")
	}
	if durationInHours <= 0 {
		return nil, fmt.Errorf("activity: durationInHours must be greater than 0")
	}
	if date.IsZero() {
		return nil, fmt.Errorf("activity: date must be a valid time")
	}
	if activityType == TypeIndividual && dogID == nil {
		return nil, ErrDogRequiredForIndividual
	}
	if sizeTarget != nil {
		switch activityType {
		case TypeSocialization, TypeRoute:
			// ok
		default:
			return nil, ErrSizeTargetNotApplicable
		}
		if !sizeTarget.IsValid() || *sizeTarget == SizeBracketUnknown {
			return nil, ErrInvalidSizeTarget
		}
	}
	return &Activity{
		id:              id,
		name:            name,
		description:     description,
		activityType:    activityType,
		maxCapacity:     maxCapacity,
		location:        location,
		durationInHours: durationInHours,
		date:            date,
		dogID:           dogID,
		sizeTarget:      sizeTarget,
	}, nil
}

// MustNewActivity is like NewActivity but panics on error. Intended for
// tests and seed data where the inputs are known to be valid.
func MustNewActivity(id int, name, description, location string, activityType ActivityType, maxCapacity, durationInHours int, date time.Time, dogID *int, sizeTarget *SizeBracket) *Activity {
	activity, err := NewActivity(id, name, description, location, activityType, maxCapacity, durationInHours, date, dogID, sizeTarget)
	if err != nil {
		panic(err)
	}
	return activity
}

func (activity *Activity) ID() int              { return activity.id }
func (activity *Activity) Name() string         { return activity.name }
func (activity *Activity) Description() string  { return activity.description }
func (activity *Activity) Type() ActivityType   { return activity.activityType }
func (activity *Activity) MaxCapacity() int     { return activity.maxCapacity }
func (activity *Activity) Location() string     { return activity.location }
func (activity *Activity) DurationInHours() int { return activity.durationInHours }
func (activity *Activity) Date() time.Time      { return activity.date }

// DogID returns the target dog id, or nil when the activity is a
// group / extra that does not target a specific dog.
func (activity *Activity) DogID() *int { return activity.dogID }

// SizeTarget returns the optional size bracket this activity is
// restricted to. nil means "all sizes welcome".
func (activity *Activity) SizeTarget() *SizeBracket { return activity.sizeTarget }

// IsTargetedAtSize reports whether a dog of the given size bracket is
// eligible to book this activity. True when no target is set (all
// sizes) or when the target matches.
func (activity *Activity) IsTargetedAtSize(target SizeBracket) bool {
	if activity == nil || activity.sizeTarget == nil {
		return true
	}
	return *activity.sizeTarget == target
}

// IsFull reports whether the activity has reached its max capacity given
// the current number of bookings.
func (activity *Activity) IsFull(currentBookings int) bool {
	if activity == nil {
		return false
	}
	return currentBookings >= activity.maxCapacity
}

// IsInThePast reports whether the activity date is strictly before now.
func (activity *Activity) IsInThePast(now time.Time) bool {
	if activity == nil {
		return false
	}
	return activity.date.Before(now)
}

// IsUpcoming reports whether the activity date is at or after now.
func (activity *Activity) IsUpcoming(now time.Time) bool {
	if activity == nil {
		return false
	}
	return !activity.date.Before(now)
}

// IsFinished reports whether the activity has ended: date + duration < now.
func (activity *Activity) IsFinished(now time.Time) bool {
	if activity == nil {
		return false
	}
	return activity.date.Add(time.Duration(activity.durationInHours) * time.Hour).Before(now)
}

// ReconstituteActivity rebuilds an Activity from persisted state,
// including the fields that have no public setter. It is the
// repository's entry point: NewActivity is for creating a genuinely new
// activity (always open), this one is for restoring an existing row.
//
// The distinction matters. A public SetClosed would let any caller flip
// the flag and bypass Close's "already closed" guard, so the only way to
// close an activity in memory stays Close(). This mirrors how
// Reservation (NewReservationWithStatus) and Invitation (NewInvitation
// vs NewPendingInvitation) already separate creation from
// reconstitution.
//
// dogID may be nil for group classes. INDIVIDUAL_CLASS rows restored
// here MUST have a non-nil dogID; the row-level CHECK constraint at
// the DB layer enforces this invariant even against future bug paths.
func ReconstituteActivity(id int, name, description, location string, activityType ActivityType, maxCapacity, durationInHours int, date time.Time, closed bool, dogID *int, sizeTarget *SizeBracket) (*Activity, error) {
	activity, err := NewActivity(id, name, description, location, activityType, maxCapacity, durationInHours, date, dogID, sizeTarget)
	if err != nil {
		return nil, err
	}
	activity.closed = closed
	return activity, nil
}

// IsClosed reports whether the activity has been closed by an admin.
func (activity *Activity) IsClosed() bool {
	if activity == nil {
		return false
	}
	return activity.closed
}

// Close transitions the activity to the closed state. Returns an error
// if the activity is already closed.
func (activity *Activity) Close() error {
	if activity == nil {
		return fmt.Errorf("activity: nil receiver")
	}
	if activity.closed {
		return fmt.Errorf("activity: already closed")
	}
	activity.closed = true
	return nil
}

// IsIndividualClass reports whether this activity is a 1-on-1 session.
func (activity *Activity) IsIndividualClass() bool {
	if activity == nil {
		return false
	}
	return activity.activityType == TypeIndividual
}

// IsSocializationGroup reports whether this activity is a group
// socialization class.
func (activity *Activity) IsSocializationGroup() bool {
	if activity == nil {
		return false
	}
	return activity.activityType == TypeSocialization
}

// IsRoute reports whether this activity is a walking route.
func (activity *Activity) IsRoute() bool {
	if activity == nil {
		return false
	}
	return activity.activityType == TypeRoute
}

// IsExtra reports whether this activity is an ad-hoc extra event.
func (activity *Activity) IsExtra() bool {
	if activity == nil {
		return false
	}
	return activity.activityType == TypeExtra
}

// ActivityPatch is a partial update: only the non-nil fields are
// mutated. See ApplyPatch for per-field validation.
//
// SizeTarget uses a triple-state: nil means "no change"; a non-nil
// pointer (including a pointer to an empty SizeBracket value) means
// "set the size target to this value". Because the use case layer
// already validates, the patch simply assigns — type/size
// consistency is re-checked by ReconstituteActivity when the row is
// re-read. To CLEAR a previously set size target, callers pass a
// pointer to an empty SizeBracket string.
type ActivityPatch struct {
	Name            *string
	Description     *string
	Location        *string
	ActivityType    *ActivityType
	MaxCapacity     *int
	DurationInHours *int
	Date            *time.Time
	SizeTarget      *SizeBracket
}

// ActivityValidationError is returned by ApplyPatch when a supplied
// value is invalid.
type ActivityValidationError struct {
	Field string
}

func (validationError *ActivityValidationError) Error() string {
	return fmt.Sprintf("activity: invalid value for %s", validationError.Field)
}

// ApplyPatch mutates the activity in place with the fields present in
// the patch. An empty patch is a no-op.
func (activity *Activity) ApplyPatch(patch ActivityPatch) error {
	if patch.Name != nil {
		if *patch.Name == "" {
			return &ActivityValidationError{Field: "name"}
		}
		activity.name = *patch.Name
	}
	if patch.Description != nil {
		activity.description = *patch.Description
	}
	if patch.Location != nil {
		if *patch.Location == "" {
			return &ActivityValidationError{Field: "location"}
		}
		activity.location = *patch.Location
	}
	if patch.ActivityType != nil {
		if !patch.ActivityType.IsValid() {
			return &ActivityValidationError{Field: "activity_type"}
		}
		activity.activityType = *patch.ActivityType
	}
	if patch.MaxCapacity != nil {
		if *patch.MaxCapacity <= 0 {
			return &ActivityValidationError{Field: "max_capacity"}
		}
		activity.maxCapacity = *patch.MaxCapacity
	}
	if patch.DurationInHours != nil {
		if *patch.DurationInHours <= 0 {
			return &ActivityValidationError{Field: "duration_in_hours"}
		}
		activity.durationInHours = *patch.DurationInHours
	}
	if patch.Date != nil {
		if patch.Date.IsZero() {
			return &ActivityValidationError{Field: "date"}
		}
		activity.date = *patch.Date
	}
	if patch.SizeTarget != nil {
		// Empty string means "clear the target" (back to all sizes).
		if *patch.SizeTarget == "" {
			activity.sizeTarget = nil
		} else if !patch.SizeTarget.IsValid() || *patch.SizeTarget == SizeBracketUnknown {
			return &ActivityValidationError{Field: "size_target"}
		} else {
			target := *patch.SizeTarget
			activity.sizeTarget = &target
		}
	}
	// Auto-clean when the activity type changes to one that does not
	// accept a size target (defence-in-depth: callers can also clear
	// it explicitly, but switching type without clearing would leave
	// a dangling invariant).
	if patch.ActivityType != nil && activity.sizeTarget != nil {
		switch *patch.ActivityType {
		case TypeSocialization, TypeRoute:
			// ok, keep
		default:
			activity.sizeTarget = nil
		}
	}
	return nil
}

// ActivityRepository is the persistence contract for Activity.
// Implemented by internal/repository/postgres.
//
// Every read method takes a viewerID + isAdmin pair so the SQL can
// apply the visibility filter at the database level. Admin viewers
// skip the filter and see every row; non-admin viewers only see
// group activities (dog_id IS NULL) and individual activities for
// dogs they own (LEFT JOIN to dogs.user_id). See individual methods
// for the exact semantic.
type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) (int, error)
	GetByID(ctx context.Context, id int, viewerUserID int, viewerIsAdmin bool) (*Activity, error)
	GetByIDForUpdate(ctx context.Context, id int, viewerUserID int, viewerIsAdmin bool) (*Activity, error)
	Update(ctx context.Context, activity *Activity) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, viewerUserID int, viewerIsAdmin bool, limit, offset int) ([]*Activity, error)
	ListByDateRange(ctx context.Context, viewerUserID int, viewerIsAdmin bool, from, to time.Time, limit, offset int) ([]*Activity, error)
	// ListByClosed returns activities whose closed flag equals
	// `closed`, optionally scoped to a date range. from/to are
	// POINTERS so the caller can pass nil for "no bound on that
	// side"; the repo MUST translate nil to SQL NULL via
	// `nullableTime` so the `$N::timestamptz IS NULL` predicate
	// short-circuits the comparison (avoids filtering by the
	// zero-value `time.Time` which serialises as
	// `0001-01-01 00:00:00 UTC` and would match nothing).
	ListByClosed(ctx context.Context, viewerUserID int, viewerIsAdmin bool, closed bool, from, to *time.Time, limit, offset int) ([]*Activity, error)
	ListUpcoming(ctx context.Context, viewerUserID int, viewerIsAdmin bool, limit, offset int) ([]*Activity, error)
}
