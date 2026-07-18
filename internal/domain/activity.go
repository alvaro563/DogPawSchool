package domain

import (
	"context"
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

// Activity is a scheduled school session: a class, a route, an
// individual session, or an extra event. Dogs are booked into Activities
// via Reservation.
type Activity struct {
	id              int
	name            string
	activityType    ActivityType
	maxCapacity     int
	location        string
	durationInHours int
	date            time.Time
}

// NewActivity creates an Activity with validated fields.
func NewActivity(id int, name, location string, activityType ActivityType, maxCapacity, durationInHours int, date time.Time) (*Activity, error) {
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
	return &Activity{
		id:              id,
		name:            name,
		activityType:    activityType,
		maxCapacity:     maxCapacity,
		location:        location,
		durationInHours: durationInHours,
		date:            date,
	}, nil
}

// MustNewActivity is like NewActivity but panics on error. Intended for
// tests and seed data where the inputs are known to be valid.
func MustNewActivity(id int, name, location string, activityType ActivityType, maxCapacity, durationInHours int, date time.Time) *Activity {
	activity, err := NewActivity(id, name, location, activityType, maxCapacity, durationInHours, date)
	if err != nil {
		panic(err)
	}
	return activity
}

func (activity *Activity) ID() int              { return activity.id }
func (activity *Activity) Name() string         { return activity.name }
func (activity *Activity) Type() ActivityType   { return activity.activityType }
func (activity *Activity) MaxCapacity() int     { return activity.maxCapacity }
func (activity *Activity) Location() string     { return activity.location }
func (activity *Activity) DurationInHours() int { return activity.durationInHours }
func (activity *Activity) Date() time.Time      { return activity.date }

// IsFull reports whether the activity has reached its max capacity given
// the current number of bookings.
func (activity *Activity) IsFull(currentBookings int) bool {
	return currentBookings >= activity.maxCapacity
}

// IsInThePast reports whether the activity date is strictly before now.
func (activity *Activity) IsInThePast(now time.Time) bool {
	return activity.date.Before(now)
}

// IsUpcoming reports whether the activity date is at or after now.
func (activity *Activity) IsUpcoming(now time.Time) bool {
	return !activity.date.Before(now)
}

// IsIndividualClass reports whether this activity is a 1-on-1 session.
func (activity *Activity) IsIndividualClass() bool { return activity.activityType == TypeIndividual }

// IsSocializationGroup reports whether this activity is a group
// socialization class.
func (activity *Activity) IsSocializationGroup() bool {
	return activity.activityType == TypeSocialization
}

// IsRoute reports whether this activity is a walking route.
func (activity *Activity) IsRoute() bool { return activity.activityType == TypeRoute }

// IsExtra reports whether this activity is an ad-hoc extra event.
func (activity *Activity) IsExtra() bool { return activity.activityType == TypeExtra }

// ActivityPatch is a partial update: only the non-nil fields are
// mutated. See ApplyPatch for per-field validation.
type ActivityPatch struct {
	Name            *string
	Location        *string
	ActivityType    *ActivityType
	MaxCapacity     *int
	DurationInHours *int
	Date            *time.Time
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
	return nil
}

// ActivityRepository is the persistence contract for Activity.
// Implemented by internal/repository/postgres.
type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) (int, error)
	GetByID(ctx context.Context, id int) (*Activity, error)
	Update(ctx context.Context, activity *Activity) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, limit, offset int) ([]*Activity, error)
	ListUpcoming(ctx context.Context, limit, offset int) ([]*Activity, error)
}
