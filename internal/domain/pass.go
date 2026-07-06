package domain

import (
	"context"
	"fmt"
	"time"
)

type PassType string

const (
	PassGeneric PassType = "GENERICO"
	PassSpecial PassType = "ESPECIFICO"
)

func (t PassType) IsValid() bool {
	switch t {
	case PassGeneric, PassSpecial:
		return true
	}
	return false
}

type PassMovement struct {
	id        int
	passID    int
	amount    int
	reason    string
	createdAt time.Time
}

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

func (m *PassMovement) ID() int              { return m.id }
func (m *PassMovement) PassID() int          { return m.passID }
func (m *PassMovement) Amount() int          { return m.amount }
func (m *PassMovement) Reason() string       { return m.reason }
func (m *PassMovement) CreatedAt() time.Time { return m.createdAt }

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

func (p *Pass) ID() int                { return p.id }
func (p *Pass) NumOfSessions() int     { return p.numOfSessions }
func (p *Pass) RemainingSessions() int { return p.remainingSessions }
func (p *Pass) Price() int             { return p.price }
func (p *Pass) Type() PassType         { return p.passType }
func (p *Pass) CreatedAt() time.Time   { return p.createdAt }
func (p *Pass) ExpiresAt() *time.Time  { return p.expiresAt }
func (p *Pass) UserID() int            { return p.userID }

func (p *Pass) Movements() []PassMovement {
	out := make([]PassMovement, len(p.movements))
	copy(out, p.movements)
	return out
}

func (p *Pass) IsExpired(now time.Time) bool {
	return p.expiresAt != nil && now.After(*p.expiresAt)
}

func (p *Pass) IsExhausted() bool {
	return p.remainingSessions <= 0
}

func (p *Pass) CanConsume(now time.Time) bool {
	return !p.IsExhausted() && !p.IsExpired(now)
}

func (p *Pass) ConsumeSession(reason string, now time.Time) (PassMovement, error) {
	if reason == "" {
		return PassMovement{}, fmt.Errorf("pass: reason must not be empty")
	}
	if p.IsExhausted() {
		return PassMovement{}, fmt.Errorf("pass: cannot consume, already exhausted")
	}
	if p.IsExpired(now) {
		return PassMovement{}, fmt.Errorf("pass: cannot consume, expired")
	}
	movement := PassMovement{
		passID:    p.id,
		amount:    -1,
		reason:    reason,
		createdAt: now,
	}
	p.movements = append(p.movements, movement)
	p.remainingSessions--
	return movement, nil
}

func (p *Pass) CanRefund() bool {
	return p.remainingSessions < p.numOfSessions
}

func (p *Pass) RefundSession(reason string, now time.Time) (PassMovement, error) {
	if reason == "" {
		return PassMovement{}, fmt.Errorf("pass: reason must not be empty")
	}
	if !p.CanRefund() {
		return PassMovement{}, fmt.Errorf("pass: cannot refund, no sessions to refund")
	}
	if p.IsExpired(now) {
		return PassMovement{}, fmt.Errorf("pass: cannot refund, expired")
	}
	movement := PassMovement{
		passID:    p.id,
		amount:    1,
		reason:    reason,
		createdAt: now,
	}
	p.movements = append(p.movements, movement)
	p.remainingSessions++
	return movement, nil
}

type PassRepository interface {
	Create(ctx context.Context, pass *Pass) error
	Update(ctx context.Context, pass *Pass) error
	GetByID(ctx context.Context, id int) (*Pass, error)
	ListByOwner(ctx context.Context, userID int) ([]*Pass, error)
	AddMovement(ctx context.Context, movement *PassMovement) error
}
