package domain

import "time"

// AttendanceReportEntry is a flat read-model projection of a COMPLETED
// reservation joined with its activity and dog. It is NOT a domain
// entity: no behavior, no invariants, no identity beyond the row.
// Used solely by the admin attendance report endpoints (JSON + CSV).
//
// Convention follows ReservationView (internal/domain/reservation_view.go):
// a flat struct that the repository interface returns without
// needing a new aggregate.
type AttendanceReportEntry struct {
	ReservationID int
	ActivityID    int
	ActivityName  string
	ActivityDate  time.Time
	DogID         int
	DogName       string
	DogPassport   string
}
