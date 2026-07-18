package domain

import (
	"context"
	"fmt"
	"time"
)

// ReservationStatus tracks the lifecycle of a Reservation.
//
// The state machine is:
//
//	StatusConfirmed
//	  ├─ Cancel(activityDate, now) → StatusCancelledInTime | StatusCancelledLate
//	  │     └─ Forgive()              → StatusForgiven
//	  ├─ Complete()                   → StatusCompleted
//	  └─ MarkNoShow()                  → StatusNoShow
type ReservationStatus string

const (
	StatusConfirmed       ReservationStatus = "CONFIRMED"
	StatusCompleted       ReservationStatus = "COMPLETED"
	StatusCancelledInTime ReservationStatus = "CANCELLED_IN_TIME"
	StatusCancelledLate   ReservationStatus = "CANCELLED_LATE"
	StatusForgiven        ReservationStatus = "FORGIVEN"
	StatusNoShow          ReservationStatus = "NO_SHOW"
)

// IsValid reports whether the value is a recognized ReservationStatus.
func (status ReservationStatus) IsValid() bool {
	switch status {
	case StatusConfirmed, StatusCompleted, StatusCancelledInTime,
		StatusCancelledLate, StatusForgiven, StatusNoShow:
		return true
	}
	return false
}

// cancellationLateWindow is how close to the activity date a cancel is
// considered "late" (i.e. the slot could no longer be resold).
const cancellationLateWindow = 2 * time.Hour

// Reservation is the join of a Dog into an Activity, paid from a Pass.
type Reservation struct {
	id         int
	activityID int
	dogID      int
	passID     int
	status     ReservationStatus
	createdAt  time.Time
}

// NewReservation creates a Reservation in the default StatusConfirmed
// state. Use NewReservationWithStatus for explicit status.
func NewReservation(id, activityID, dogID, passID int, createdAt time.Time) (*Reservation, error) {
	return NewReservationWithStatus(id, activityID, dogID, passID, StatusConfirmed, createdAt)
}

// NewReservationWithStatus creates a Reservation with an explicit initial
// status. All id fields must be positive (except id, which is the DB id
// and may be 0 for not-yet-persisted reservations).
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

func (reservation *Reservation) ID() int                   { return reservation.id }
func (reservation *Reservation) ActivityID() int           { return reservation.activityID }
func (reservation *Reservation) DogID() int                { return reservation.dogID }
func (reservation *Reservation) PassID() int               { return reservation.passID }
func (reservation *Reservation) Status() ReservationStatus { return reservation.status }
func (reservation *Reservation) CreatedAt() time.Time      { return reservation.createdAt }

// Cancel transitions a confirmed reservation to either CancelledInTime
// or CancelledLate depending on how close to the activity date the cancel
// happens. Returns an error if the reservation is not in StatusConfirmed.
func (reservation *Reservation) Cancel(activityDate, now time.Time) error {
	if reservation.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot cancel, current status is %s", reservation.status)
	}
	if now.After(activityDate.Add(-cancellationLateWindow)) {
		reservation.status = StatusCancelledLate
	} else {
		reservation.status = StatusCancelledInTime
	}
	return nil
}

// Complete transitions a confirmed reservation to StatusCompleted.
// Returns an error if the reservation is not in StatusConfirmed.
func (reservation *Reservation) Complete() error {
	if reservation.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot complete, current status is %s", reservation.status)
	}
	reservation.status = StatusCompleted
	return nil
}

// MarkNoShow transitions a confirmed reservation to StatusNoShow.
// Returns an error if the reservation is not in StatusConfirmed.
func (reservation *Reservation) MarkNoShow() error {
	if reservation.status != StatusConfirmed {
		return fmt.Errorf("reservation: cannot mark no-show, current status is %s", reservation.status)
	}
	reservation.status = StatusNoShow
	return nil
}

// Forgive transitions a late-cancelled reservation to StatusForgiven,
// which is the only way to refund a late cancellation. Returns an error
// if the reservation is not in StatusCancelledLate.
func (reservation *Reservation) Forgive() error {
	if reservation.status != StatusCancelledLate {
		return fmt.Errorf("reservation: cannot forgive, current status is %s", reservation.status)
	}
	reservation.status = StatusForgiven
	return nil
}

// IsConfirmed reports whether the reservation is still active.
func (reservation *Reservation) IsConfirmed() bool { return reservation.status == StatusConfirmed }

// IsCancelled reports whether the reservation has been cancelled (in
// time, late, or forgiven).
func (reservation *Reservation) IsCancelled() bool {
	return reservation.status == StatusCancelledInTime || reservation.status == StatusCancelledLate || reservation.status == StatusForgiven
}

// WasCancelledInTime reports whether the reservation was cancelled with
// enough lead time for the slot to be resold.
func (reservation *Reservation) WasCancelledInTime() bool {
	return reservation.status == StatusCancelledInTime
}

// WasCancelledLate reports whether the reservation was cancelled within
// the cancellation late window (i.e. too close to the activity date to
// resell the slot).
func (reservation *Reservation) WasCancelledLate() bool {
	return reservation.status == StatusCancelledLate
}

// ReservationRepository is the persistence contract for Reservation.
// Implemented by internal/repository/postgres (future).
type ReservationRepository interface {
	Create(ctx context.Context, reservation *Reservation) error
	Update(ctx context.Context, reservation *Reservation) error
	GetByID(ctx context.Context, id int) (*Reservation, error)
	ListByActivity(ctx context.Context, activityID int) ([]*Reservation, error)
	ListByDog(ctx context.Context, dogID int) ([]*Reservation, error)
	ListByPass(ctx context.Context, passID int) ([]*Reservation, error)
}
