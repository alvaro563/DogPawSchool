package domain

import (
	"context"
	"fmt"
	"time"
)

type ActivityType string

const (
	TypeSocialization ActivityType = "SOCIALIZATION_GROUP"
	TypeRoute         ActivityType = "ROUTE"
	TypeIndividual    ActivityType = "INDIVIDUAL_CLASS"
)

func (t ActivityType) IsValid() bool {
	switch t {
	case TypeSocialization, TypeRoute, TypeIndividual:
		return true
	}
	return false
}

type Activity struct {
	id              int
	name            string
	activityType    ActivityType
	maxCapacity     int
	location        string
	durationInHours int
	date            time.Time
}

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

func (a *Activity) ID() int              { return a.id }
func (a *Activity) Name() string         { return a.name }
func (a *Activity) Type() ActivityType   { return a.activityType }
func (a *Activity) MaxCapacity() int     { return a.maxCapacity }
func (a *Activity) Location() string     { return a.location }
func (a *Activity) DurationInHours() int { return a.durationInHours }
func (a *Activity) Date() time.Time      { return a.date }

func (a *Activity) IsFull(currentBookings int) bool {
	return currentBookings >= a.maxCapacity
}

func (a *Activity) IsInThePast(now time.Time) bool {
	return a.date.Before(now)
}

func (a *Activity) IsUpcoming(now time.Time) bool {
	return !a.date.Before(now)
}

func (a *Activity) IsIndividualClass() bool    { return a.activityType == TypeIndividual }
func (a *Activity) IsSocializationGroup() bool { return a.activityType == TypeSocialization }
func (a *Activity) IsRoute() bool              { return a.activityType == TypeRoute }

type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) error
	Update(ctx context.Context, activity *Activity) error
	GetByID(ctx context.Context, id int) (*Activity, error)
	ListUpcoming(ctx context.Context) ([]*Activity, error)
	Delete(ctx context.Context, id int) error
}
