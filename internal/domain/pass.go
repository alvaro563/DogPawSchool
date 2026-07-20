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
// consumes sessions (or incremented on refund). The createdAt and
// updatedAt timestamps are both maintained automatically: createdAt
// is set on first persist, updatedAt is bumped by a DB trigger on
// every UPDATE.
type Pass struct {
	id                int
	numOfSessions     int
	remainingSessions int
	price             int
	passType          PassType
	createdAt         time.Time
	updatedAt         time.Time
	expiresAt         *time.Time
	userID            int
	movements         []PassMovement
}

// NewPass creates a Pass. The caller must pass the current
// remainingSessions explicitly: for a brand-new pass, pass the same
// value as numOfSessions; for a pass loaded from the DB, pass the
// value that the DB currently holds (the repository is the only
// caller in the latter case). The invariant
// remainingSessions <= numOfSessions is enforced here so a buggy
// caller cannot put the aggregate into an inconsistent state.
func NewPass(id, numOfSessions, remainingSessions, price int, passType PassType, userID int, createdAt, updatedAt time.Time, expiresAt *time.Time) (*Pass, error) {
	if id < 0 {
		return nil, fmt.Errorf("pass: id must not be negative")
	}
	if numOfSessions <= 0 {
		return nil, fmt.Errorf("pass: numOfSessions must be greater than 0")
	}
	if remainingSessions < 0 {
		return nil, fmt.Errorf("pass: remainingSessions must not be negative")
	}
	if remainingSessions > numOfSessions {
		return nil, fmt.Errorf("pass: remainingSessions (%d) must not exceed numOfSessions (%d)", remainingSessions, numOfSessions)
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
	if updatedAt.IsZero() {
		return nil, fmt.Errorf("pass: updatedAt must be a valid time")
	}
	return &Pass{
		id:                id,
		numOfSessions:     numOfSessions,
		remainingSessions: remainingSessions,
		price:             price,
		passType:          passType,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		expiresAt:         expiresAt,
		userID:            userID,
	}, nil
}

// MustNewPass is like NewPass but panics on error. Intended for
// tests and seed data where the inputs are known to be valid.
func MustNewPass(id, numOfSessions, remainingSessions, price int, passType PassType, userID int, createdAt, updatedAt time.Time, expiresAt *time.Time) *Pass {
	pass, err := NewPass(id, numOfSessions, remainingSessions, price, passType, userID, createdAt, updatedAt, expiresAt)
	if err != nil {
		panic(err)
	}
	return pass
}

func (pass *Pass) ID() int                { return pass.id }
func (pass *Pass) NumOfSessions() int     { return pass.numOfSessions }
func (pass *Pass) RemainingSessions() int { return pass.remainingSessions }
func (pass *Pass) Price() int             { return pass.price }
func (pass *Pass) Type() PassType         { return pass.passType }
func (pass *Pass) CreatedAt() time.Time   { return pass.createdAt }
func (pass *Pass) UpdatedAt() time.Time   { return pass.updatedAt }
func (pass *Pass) ExpiresAt() *time.Time  { return pass.expiresAt }
func (pass *Pass) UserID() int            { return pass.userID }

// Movements returns a defensive copy of the pass movements.
func (pass *Pass) Movements() []PassMovement {
	out := make([]PassMovement, len(pass.movements))
	copy(out, pass.movements)
	return out
}

// PassPatch is a partial update: only the non-nil fields are mutated.
// See ApplyPatch for per-field validation. The non-modifiable fields
// (id, numOfSessions, remainingSessions, userID, createdAt) are
// deliberately excluded from the patch — changing them would either
// break the audit-log invariant (sum of movements must match the
// pass state) or be a different operation (e.g., re-assigning
// ownership).
type PassPatch struct {
	Price     *int
	PassType  *PassType
	ExpiresAt *time.Time
}

// PassValidationError is returned by ApplyPatch when a supplied
// value is invalid.
type PassValidationError struct {
	Field string
}

func (validationError *PassValidationError) Error() string {
	return fmt.Sprintf("pass: invalid value for %s", validationError.Field)
}

// ApplyPatch mutates the pass in place with the fields present in
// the patch. An empty patch is a no-op. Only the editable fields
// (price, pass_type, expires_at) are accepted.
func (pass *Pass) ApplyPatch(patch PassPatch) error {
	if patch.Price != nil {
		if *patch.Price < 0 {
			return &PassValidationError{Field: "price"}
		}
		pass.price = *patch.Price
	}
	if patch.PassType != nil {
		if !patch.PassType.IsValid() {
			return &PassValidationError{Field: "pass_type"}
		}
		pass.passType = *patch.PassType
	}
	if patch.ExpiresAt != nil {
		if patch.ExpiresAt.IsZero() {
			return &PassValidationError{Field: "expires_at"}
		}
		pass.expiresAt = patch.ExpiresAt
	}
	return nil
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
// Errors if there is nothing to refund or the reason is empty.
//
// This method intentionally does NOT check pass.IsExpired. The policy
// "no refund on expired pass" is enforced at the use case layer
// (owner-side refunds). Admin-side refunds (e.g., activity
// cancellation) can and should override this rule. ConsumeSession
// still checks expiry because consuming a session on an expired pass
// is never a valid operation.
func (pass *Pass) RefundSession(reason string, now time.Time) (PassMovement, error) {
	if reason == "" {
		return PassMovement{}, fmt.Errorf("pass: reason must not be empty")
	}
	if !pass.CanRefund() {
		return PassMovement{}, fmt.Errorf("pass: cannot refund, no sessions to refund")
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
// internal/repository/postgres.
type PassRepository interface {
	Create(ctx context.Context, pass *Pass) (int, error)
	Update(ctx context.Context, pass *Pass) error
	GetByID(ctx context.Context, id int) (*Pass, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Pass, error)
	ListByOwner(ctx context.Context, userID, limit, offset int) ([]*Pass, error)
	AddMovement(ctx context.Context, movement *PassMovement) error
}
