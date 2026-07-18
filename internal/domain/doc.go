// Package domain contains the core business entities and their invariants.
// It is the innermost layer of Clean Architecture: zero dependencies on
// infrastructure (DB, HTTP, etc.).
//
// Files in this package:
//
//   - dog.go:              Dog, Sex, AgeBracket, SizeBracket
//   - incompatibility.go: Incompatibility, IncompatibilityLevel
//   - reservation.go:     Reservation, ReservationStatus
//   - user.go:            User, UserRole
//   - pass.go:            Pass, PassMovement, PassType
//   - activity.go:        Activity, ActivityType
//
// The *Repository interfaces declared in this package are implemented by
// the outer layers (e.g., internal/repository/postgres). This is the
// Dependency Inversion Principle: outer layers depend on the domain, not
// vice versa.
package domain
