package domain

import (
	"context"
	"fmt"
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

func (s ReservationStatus) IsValid() bool {
	switch s {
	case StatusConfirmed, StatusCompleted, StatusCancelledInTime,
		StatusCancelledLate, StatusForgiven, StatusNoShow:
		return true
	}
	return false
}

type Reservation struct {
	id         int
	activityID int
	dogID      int
	passID     int
	status     ReservationStatus
	createdAt  time.Time
}

func NewReservation(id, activityID, dogID, passID int, createdAt time.Time) (*Reservation, error) {
	return NewReservationWithStatus(id, activityID, dogID, passID, StatusConfirmed, createdAt)
}

func NewReservationWithStatus(id, activityID, dogID, passID int, status ReservationStatus, createdAt time.Time) (*Reservation, error) {
	if id < 0 {
		return nil, fmt.Errorf("reservation: id must not be negative")
	}
	if activityID <= 0 {
		return nil, fmt.Errorf("reservation: activityID must be greater than 0")
	}
	if dogID <= 0 {
		return nil, fmt.Errorf("reservation: dogID must be greater than 0")
	}
	if passID <= 0 {
		return nil, fmt.Errorf("reservation: passID must be greater than 0")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("reservation: invalid status %q", status)
	}
	if createdAt.IsZero() {
		return nil, fmt.Errorf("reservation: createdAt must be a valid time")
	}
	return &Reservation{
		id:         id,
		activityID: activityID,
		dogID:      dogID,
		passID:     passID,
		status:     status,
		createdAt:  createdAt,
	}, nil
}

func (r *Reservation) ID() int                   { return r.id }
func (r *Reservation) ActivityID() int           { return r.activityID }
func (r *Reservation) DogID() int                { return r.dogID }
func (r *Reservation) PassID() int               { return r.passID }
func (r *Reservation) Status() ReservationStatus { return r.status }
func (r *Reservation) CreatedAt() time.Time      { return r.createdAt }

func (r *Reservation) Cancel(activityDate, now time.Time) error {
	if r.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot cancel, current status is %s", r.status)
	}
	if now.After(activityDate.Add(-2 * time.Hour)) {
		r.status = StatusCancelledLate
	} else {
		r.status = StatusCancelledInTime
	}
	return nil
}

func (r *Reservation) Complete() error {
	if r.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot complete, current status is %s", r.status)
	}
	r.status = StatusCompleted
	return nil
}

func (r *Reservation) MarkNoShow() error {
	if r.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot mark no-show, current status is %s", r.status)
	}
	r.status = StatusNoShow
	return nil
}

func (r *Reservation) Forgive() error {
	if r.status != StatusCancelledLate {
		return fmt.Errorf("reservation: cannot forgive, current status is %s", r.status)
	}
	r.status = StatusForgiven
	return nil
}

func (r *Reservation) IsConfirmed() bool { return r.status == StatusConfirmed }
func (r *Reservation) IsCancelled() bool {
	return r.status == StatusCancelledInTime || r.status == StatusCancelledLate || r.status == StatusForgiven
}
func (r *Reservation) WasCancelledInTime() bool { return r.status == StatusCancelledInTime }
func (r *Reservation) WasCancelledLate() bool   { return r.status == StatusCancelledLate }

type ReservationRepository interface {
	Create(ctx context.Context, reservation *Reservation) error
	Update(ctx context.Context, reservation *Reservation) error
	GetByID(ctx context.Context, id int) (*Reservation, error)
	ListByActivity(ctx context.Context, activityID int) ([]*Reservation, error)
	ListByDog(ctx context.Context, dogID int) ([]*Reservation, error)
	ListByPass(ctx context.Context, passID int) ([]*Reservation, error)
}
