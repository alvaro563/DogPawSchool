package domain

import (
	"context"
	"time"
)

type ReservationStatus string

const (
	StatusConfirmed       ReservationStatus = "CONFIRMED"
	StatusCompleted       ReservationStatus = "COMPLETED"
	StatusCancelledInTime ReservationStatus = "CANCELLED_IN_TIME"
	StatusCancelledLate   ReservationStatus = "CANCELLED_LATE"
	StatusForgiven        ReservationStatus = "FORGIVEN"
	StatusNoShow          ReservationStatus = "NO_SHOW"
)

type Reservation struct {
	ID         int
	ActivityID int
	DogID      int
	PassID     int
	Status     ReservationStatus
	CreatedAt  time.Time
}

type ReservationRepository interface {
	Create(ctx context.Context, reservation *Reservation) error
	Update(ctx context.Context, reservation *Reservation) error
	GetByID(ctx context.Context, id int) (*Reservation, error)
	ListByActivity(ctx context.Context, activityID int) ([]*Reservation, error)
	ListByDog(ctx context.Context, dogID int) ([]*Reservation, error)
	ListByPass(ctx context.Context, passID int) ([]*Reservation, error)
}
