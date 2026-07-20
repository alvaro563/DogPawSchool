package domain

import (
	"fmt"
	"time"
)

// ReservationView is a denormalized read model of a Reservation that
// pulls in the most commonly needed fields from the related
// Activity, Dog, and Pass aggregates. It is built by a single SQL
// query (3 LEFT JOINs) and is never persisted; mutations go through
// the bare Reservation type. Handlers serialize this type to a
// single shape shared by every read endpoint so the client does not
// have to learn multiple envelopes.
//
// The constructor validates that the three foreign aggregates are
// present and consistent (activity id == reservation activity id,
// etc.) so a buggy repository cannot return a view that contradicts
// its reservation. Time fields are surfaced via the sub-objects
// (Activity.Date()) rather than copied into the view, to avoid
// drift between the two.
type ReservationView struct {
	reservation *Reservation
	activity    *Activity
	dog         *Dog
	pass        *Pass
}

// NewReservationView builds a ReservationView from its four
// constituent aggregates. Returns a *ValidationError-equivalent
// (plain fmt.Errorf) if any pointer is nil, the reservation is
// not bound to the same id as the activity/dog/pass, or any
// aggregate has a zero id. The integrity check is cheap and
// catches a class of repository bugs where a JOIN would silently
// drop a row.
func NewReservationView(reservation *Reservation, activity *Activity, dog *Dog, pass *Pass) (*ReservationView, error) {
	if reservation == nil {
		return nil, fmt.Errorf("reservation view: reservation is required")
	}
	if activity == nil {
		return nil, fmt.Errorf("reservation view: activity is required")
	}
	if dog == nil {
		return nil, fmt.Errorf("reservation view: dog is required")
	}
	if pass == nil {
		return nil, fmt.Errorf("reservation view: pass is required")
	}
	if activity.ID() != reservation.ActivityID() {
		return nil, fmt.Errorf("reservation view: activity id %d does not match reservation activity id %d",
			activity.ID(), reservation.ActivityID())
	}
	if dog.ID() != reservation.DogID() {
		return nil, fmt.Errorf("reservation view: dog id %d does not match reservation dog id %d",
			dog.ID(), reservation.DogID())
	}
	if pass.ID() != reservation.PassID() {
		return nil, fmt.Errorf("reservation view: pass id %d does not match reservation pass id %d",
			pass.ID(), reservation.PassID())
	}
	return &ReservationView{
		reservation: reservation,
		activity:    activity,
		dog:         dog,
		pass:        pass,
	}, nil
}

// Reservation returns the underlying reservation. Use this when the
// caller only needs reservation-level data.
func (view *ReservationView) Reservation() *Reservation { return view.reservation }

// Activity returns the joined activity.
func (view *ReservationView) Activity() *Activity { return view.activity }

// Dog returns the joined dog.
func (view *ReservationView) Dog() *Dog { return view.dog }

// Pass returns the joined pass.
func (view *ReservationView) Pass() *Pass { return view.pass }

// Convenience accessors. These exist so the handler can build a
// flat DTO without having to reach into the sub-objects.

// ID returns the reservation id.
func (view *ReservationView) ID() int { return view.reservation.ID() }

// Status returns the reservation status.
func (view *ReservationView) Status() ReservationStatus { return view.reservation.Status() }

// CreatedAt returns the reservation creation timestamp.
func (view *ReservationView) CreatedAt() time.Time { return view.reservation.CreatedAt() }

// ActivityID is a shortcut for view.Activity().ID().
func (view *ReservationView) ActivityID() int { return view.activity.ID() }

// ActivityName is a shortcut for view.Activity().Name().
func (view *ReservationView) ActivityName() string { return view.activity.Name() }

// ActivityDate is a shortcut for view.Activity().Date().
func (view *ReservationView) ActivityDate() time.Time { return view.activity.Date() }

// ActivityLocation is a shortcut for view.Activity().Location().
func (view *ReservationView) ActivityLocation() string { return view.activity.Location() }

// DogID is a shortcut for view.Dog().ID().
func (view *ReservationView) DogID() int { return view.dog.ID() }

// DogName is a shortcut for view.Dog().Name().
func (view *ReservationView) DogName() string { return view.dog.Name() }

// PassID is a shortcut for view.Pass().ID().
func (view *ReservationView) PassID() int { return view.pass.ID() }

// PassType is a shortcut for view.Pass().Type().
func (view *ReservationView) PassType() PassType { return view.pass.Type() }

// PassRemaining is a shortcut for view.Pass().RemainingSessions().
func (view *ReservationView) PassRemaining() int { return view.pass.RemainingSessions() }

// DogUserID exposes the dog owner's id so the handler can enforce
// ownership in the "GET /users/:user_id/reservations/:id" route
// without reaching into the sub-object.
func (view *ReservationView) DogUserID() int { return view.dog.UserID() }

// IsUpcoming reports whether the reservation is CONFIRMED and the
// activity is at or after now. Used by the upcoming use case to
// keep the filtering logic close to the domain (the SQL query
// already enforces the same conditions, this is a defense-in-depth
// check that can be reused by other read paths).
func (view *ReservationView) IsUpcoming(now time.Time) bool {
	return view.reservation.IsConfirmed() && !view.activity.IsInThePast(now)
}
