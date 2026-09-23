package domain

import (
	"context"
	"errors"
	"time"
)

// AccountLoginLimiter is the contract for the auth-failure tracker
// used by LoginUseCase. The Postgres implementation writes one row
// per attempt into login_attempts and counts failures inside a
// rolling window. Other implementations (e.g. Redis) could swap in
// without changing the use case.
//
// The interface deliberately returns ErrAccountLockout (not a
// plain error) so the handler can read RetryAfter and emit the
// RFC 6585 / IETF draft-8 Retry-After + RateLimit-* headers.
//
// Three operations:
//   - Check       called BEFORE the bcrypt check; rejects with
//                 ErrAccountLockout if the limit is already
//                 exceeded.
//   - Record      called AFTER the password check, on every
//                 attempt (success or failure).
//   - ResetByEmail called only on successful login; clears the
//                 failure history for that email so a legitimate
//                 user who finally entered the right password is
//                 not penalised for their previous typos. The IP
//                 history is left intact.
type AccountLoginLimiter interface {
	Check(ctx context.Context, email, remoteIP string) error
	Record(ctx context.Context, email, remoteIP string, success bool) error
	ResetByEmail(ctx context.Context, email string) error
}

// LockoutReason tells the handler which axis triggered the lock.
// Used for logging and (later) for finer-grained frontend copy.
// Exposed on ErrAccountLockout so the handler can include it in
// the JSON error body.
type LockoutReason string

const (
	LockoutReasonEmail LockoutReason = "email"
	LockoutReasonIP    LockoutReason = "ip"
)

// ErrAccountLockout is returned by AccountLoginLimiter.Check when
// too many failed attempts have been recorded against the same
// email OR from the same remote IP within their respective
// rolling windows. RetryAfter is the time until the OLDEST counted
// attempt falls out of the window — i.e. the soonest moment the
// caller could legitimately retry.
//
// This is a struct error (not a sentinel) because the handler
// needs the RetryAfter duration to populate the Retry-After
// header. The errors.Is(err, ErrAccountLocked) sentinel below
// allows use cases that just want to know "is this a lockout?"
// to do so without unwrapping.
type ErrAccountLockout struct {
	RetryAfter time.Duration
	Reason     LockoutReason
}

func (e *ErrAccountLockout) Error() string {
	return "account temporarily locked: " + string(e.Reason)
}

// Is enables errors.Is(err, ErrAccountLocked) — standard Go
// sentinel matching even though the concrete error carries extra
// data. Use errors.As to extract RetryAfter and Reason.
func (e *ErrAccountLockout) Is(target error) bool {
	return target == ErrAccountLocked
}

// ErrAccountLocked is the sentinel form of the lockout. Errors.As
// can extract *ErrAccountLockout for the structured fields;
// errors.Is(err, ErrAccountLocked) returns true for any lockout
// regardless of axis.
var ErrAccountLocked = errors.New("account temporarily locked")
