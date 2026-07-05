package domain

import (
	"context"
	"time"
)

type PassType string

const (
	PassGeneric PassType = "GENERICO"
	PassSpecial PassType = "ESPECIFICO"
)

type PassMovement struct {
	ID        int
	PassID    int
	Amount    int
	Reason    string
	CreatedAt time.Time
}

type Pass struct {
	ID                int
	NumOfSessions     int
	RemainingSessions int
	Price             int
	Type              PassType
	CreatedAt         time.Time
	ExpiresAt         *time.Time
	UserID            int
	Movements         []PassMovement
}

func (p *Pass) IsExpired(now time.Time) bool {
	return p.ExpiresAt != nil && now.After(*p.ExpiresAt)
}

func (p *Pass) IsExhausted() bool {
	return p.RemainingSessions <= 0
}

type PassRepository interface {
	Create(ctx context.Context, pass *Pass) error
	Update(ctx context.Context, pass *Pass) error
	GetByID(ctx context.Context, id int) (*Pass, error)
	ListByOwner(ctx context.Context, userID int) ([]*Pass, error)
	AddMovement(ctx context.Context, movement *PassMovement) error
}
