package domain

import (
	"context"
	"fmt"
	"time"
)

// PassType distinguishes the two kinds of prepaid session packs.
type PassType string

const (
	PassGeneric PassType = "GENERICO"
	PassSpecial PassType = "ESPECIFICO"
)

// IsValid reports whether the value is a recognized PassType.
func (passType PassType) IsValid() bool {
	switch passType {
	case PassGeneric, PassSpecial:
		return true
	}
	return false
}

// PassMovement is the append-only audit entry of a session consume (-1)
// or refund (+1). The aggregate Pass carries its movements so the
// domain enforces the pass invariant (remaining = numOf − sum(movements)).
type PassMovement struct {
	id        int
	passID    int
	amount    int
	reason    string
	createdAt time.Time
}

// NewPassMovement creates a PassMovement. amount must be non-zero (it's
// either -1 for consume or +1 for refund).
func NewPassMovement(id, passID, amount int, reason string, createdAt time.Time) (*PassMovement, error) {
	if id < 0 {
		return nil, fmt.Errorf("pass movement: id must not be negative")
	}
	if passID <= 0 {
		return nil, fmt.Errorf("pass movement: passID must be greater than 0")
	}
	if amount == 0 {
		return nil, fmt.Errorf("pass movement: amount must not be zero")
	}
	if reason == "" {
		return nil, fmt.Errorf("pass movement: reason must not be empty")
	}
	if createdAt.IsZero() {
		return nil, fmt.Errorf("pass movement: createdAt must be a valid time")
	}
	return &PassMovement{
		id:        id,
		passID:    passID,
		amount:    amount,
		reason:    reason,
		createdAt: createdAt,
	}, nil
}

func (movement *PassMovement) ID() int              { return movement.id }
func (movement *PassMovement) PassID() int          { return movement.passID }
func (movement *PassMovement) Amount() int          { return movement.amount }
func (movement *PassMovement) Reason() string       { return movement.reason }
func (movement *PassMovement) CreatedAt() time.Time { return movement.createdAt }

// Pass is a prepaid session pack owned by a User. A Pass starts with
// remainingSessions == numOfSessions and is decremented as the user
// consumes sessions (or incremented on refund).
type Pass struct {
	id                int
	numOfSessions     int
	remainingSessions int
	price             int
	passType          PassType
	createdAt         time.Time
	expiresAt         *time.Time
	userID            int
	movements         []PassMovement
}

// NewPass creates a Pass. A new pass starts fully available
// (remainingSessions = numOfSessions) with no movements.
func NewPass(id, numOfSessions, price int, passType PassType, userID int, createdAt time.Time, expiresAt *time.Time) (*Pass, error) {
	if id < 0 {
		return nil, fmt.Errorf("pass: id must not be negative")
	}
	if numOfSessions <= 0 {
		return nil, fmt.Errorf("pass: numOfSessions must be greater than 0")
	}
	if price < 0 {
		return nil, fmt.Errorf("pass: price must not be negative")
	}
	if !passType.IsValid() {
		return nil, fmt.Errorf("pass: invalid passType %q", passType)
	}
	if userID <= 0 {
		return nil, fmt.Errorf("pass: userID must be greater than 0")
	}
	if createdAt.IsZero() {
		return nil, fmt.Errorf("pass: createdAt must be a valid time")
	}
	return &Pass{
		id:                id,
		numOfSessions:     numOfSessions,
		remainingSessions: numOfSessions,
		price:             price,
		passType:          passType,
		createdAt:         createdAt,
		expiresAt:         expiresAt,
		userID:            userID,
	}, nil
}

func (pass *Pass) ID() int                { return pass.id }
func (pass *Pass) NumOfSessions() int     { return pass.numOfSessions }
func (pass *Pass) RemainingSessions() int { return pass.remainingSessions }
func (pass *Pass) Price() int             { return pass.price }
func (pass *Pass) Type() PassType         { return pass.passType }
func (pass *Pass) CreatedAt() time.Time   { return pass.createdAt }
func (pass *Pass) ExpiresAt() *time.Time  { return pass.expiresAt }
func (pass *Pass) UserID() int            { return pass.userID }

// Movements returns a defensive copy of the pass movements.
func (pass *Pass) Movements() []PassMovement {
	out := make([]PassMovement, len(pass.movements))
	copy(out, pass.movements)
	return out
}

// IsExpired reports whether the pass has expired relative to now.
// A pass with no expiry is never expired.
func (pass *Pass) IsExpired(now time.Time) bool {
	return pass.expiresAt != nil && now.After(*pass.expiresAt)
}

// IsExhausted reports whether the pass has no remaining sessions.
func (pass *Pass) IsExhausted() bool {
	return pass.remainingSessions <= 0
}

// CanConsume reports whether the pass is currently usable: not exhausted
// and not expired.
func (pass *Pass) CanConsume(now time.Time) bool {
	return !pass.IsExhausted() && !pass.IsExpired(now)
}

// ConsumeSession decrements remainingSessions by 1 and appends a
// movement (amount=-1) to the audit log. Returns the created movement.
// Errors if the pass is exhausted, expired, or reason is empty.
func (pass *Pass) ConsumeSession(reason string, now time.Time) (PassMovement, error) {
	if reason == "" {
		return PassMovement{}, fmt.Errorf("pass: reason must not be empty")
	}
	if pass.IsExhausted() {
		return PassMovement{}, fmt.Errorf("pass: cannot consume, already exhausted")
	}
	if pass.IsExpired(now) {
		return PassMovement{}, fmt.Errorf("pass: cannot consume, expired")
	}
	movement := PassMovement{
		passID:    pass.id,
		amount:    -1,
		reason:    reason,
		createdAt: now,
	}
	pass.movements = append(pass.movements, movement)
	pass.remainingSessions--
	return movement, nil
}

// CanRefund reports whether the pass has any consumed sessions to refund.
func (pass *Pass) CanRefund() bool {
	return pass.remainingSessions < pass.numOfSessions
}

// RefundSession increments remainingSessions by 1 and appends a
// movement (amount=+1) to the audit log. Returns the created movement.
// Errors if there is nothing to refund, the pass is expired, or reason
// is empty.
func (pass *Pass) RefundSession(reason string, now time.Time) (PassMovement, error) {
	if reason == "" {
		return PassMovement{}, fmt.Errorf("pass: reason must not be empty")
	}
	if !pass.CanRefund() {
		return PassMovement{}, fmt.Errorf("pass: cannot refund, no sessions to refund")
	}
	if pass.IsExpired(now) {
		return PassMovement{}, fmt.Errorf("pass: cannot refund, expired")
	}
	movement := PassMovement{
		passID:    pass.id,
		amount:    1,
		reason:    reason,
		createdAt: now,
	}
	pass.movements = append(pass.movements, movement)
	pass.remainingSessions++
	return movement, nil
}

// PassRepository is the persistence contract for Pass (and its
// PassMovement children). Implemented by
// internal/repository/postgres (future).
type PassRepository interface {
	Create(ctx context.Context, pass *Pass) error
	Update(ctx context.Context, pass *Pass) error
	GetByID(ctx context.Context, id int) (*Pass, error)
	ListByOwner(ctx context.Context, userID int) ([]*Pass, error)
	AddMovement(ctx context.Context, movement *PassMovement) error
}
