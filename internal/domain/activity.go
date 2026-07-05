package domain

import (
	"context"
	"time"
)

type ActivityType string

const (
	TypeSocialization ActivityType = "SOCIALIZATION_GROUP"
	TypeRoute         ActivityType = "ROUTE"
	TypeIndividual    ActivityType = "INDIVIDUAL_CLASS"
)

type Activity struct {
	ID              int
	Name            string
	Type            ActivityType
	MaxCapacity     int
	Location        string
	DurationInHours int
	Date            time.Time
}

type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) error
	Update(ctx context.Context, activity *Activity) error
	GetByID(ctx context.Context, id int) (*Activity, error)
	ListUpcoming(ctx context.Context) ([]*Activity, error)
	Delete(ctx context.Context, id int) error
}
