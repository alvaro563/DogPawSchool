package domain

import (
	"context"
	"fmt"
	"time"
)

// InvitationStatus tracks the lifecycle of an Invitation.
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "PENDING"
	InvitationAccepted InvitationStatus = "ACCEPTED"
	InvitationExpired  InvitationStatus = "EXPIRED"
	InvitationRevoked  InvitationStatus = "REVOKED"
)

// IsValid reports whether the value is a recognized InvitationStatus.
func (status InvitationStatus) IsValid() bool {
	switch status {
	case InvitationPending, InvitationAccepted, InvitationExpired, InvitationRevoked:
		return true
	}
	return false
}

// Invitation is an invitation for a new user to join the platform. An
// invitation is created by an existing user (typically an admin) and
// carries a unique token that the invitee uses to complete
// registration. The invitation transitions through a simple lifecycle:
// PENDING → ACCEPTED | REVOKED | EXPIRED.
type Invitation struct {
	id        int
	email     string
	token     string
	role      UserRole
	status    InvitationStatus
	createdBy int
	expiresAt time.Time
	createdAt time.Time
	updatedAt time.Time
}

// NewInvitation creates an Invitation with all fields explicitly
// provided. This is the reconstruction constructor used by the
// repository when loading from the database: every persisted field is
// passed in and validated. expiresAt must be strictly after createdAt.
func NewInvitation(id, createdBy int, email, token string, role UserRole, status InvitationStatus, expiresAt, createdAt, updatedAt time.Time) (*Invitation, error) {
	if id < 0 {
		return nil, fmt.Errorf("invitation: id must not be negative")
	}
	if createdBy <= 0 {
		return nil, fmt.Errorf("invitation: createdBy must be greater than 0")
	}
	if email == "" {
		return nil, fmt.Errorf("invitation: email must not be empty")
	}
	if !basicEmailRegex.MatchString(email) {
		return nil, fmt.Errorf("invitation: invalid email format")
	}
	if token == "" {
		return nil, fmt.Errorf("invitation: token must not be empty")
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("invitation: invalid role %q", role)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invitation: invalid status %q", status)
	}
	if createdAt.IsZero() {
		return nil, fmt.Errorf("invitation: createdAt must be a valid time")
	}
	if updatedAt.IsZero() {
		return nil, fmt.Errorf("invitation: updatedAt must be a valid time")
	}
	if expiresAt.IsZero() {
		return nil, fmt.Errorf("invitation: expiresAt must be a valid time")
	}
	if expiresAt.Before(createdAt) || expiresAt.Equal(createdAt) {
		return nil, fmt.Errorf("invitation: expiresAt must be after createdAt")
	}
	return &Invitation{
		id:        id,
		email:     email,
		token:     token,
		role:      role,
		status:    status,
		createdBy: createdBy,
		expiresAt: expiresAt,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

// NewPendingInvitation creates a brand-new Invitation in PENDING
// status with id=0 (not yet persisted) and timestamps set to now.
// Use this when creating a fresh invitation; use NewInvitation for
// reconstructing an existing one from the database.
func NewPendingInvitation(createdBy int, email, token string, role UserRole, expiresAt, now time.Time) (*Invitation, error) {
	if createdBy <= 0 {
		return nil, fmt.Errorf("invitation: createdBy must be greater than 0")
	}
	if email == "" {
		return nil, fmt.Errorf("invitation: email must not be empty")
	}
	if !basicEmailRegex.MatchString(email) {
		return nil, fmt.Errorf("invitation: invalid email format")
	}
	if token == "" {
		return nil, fmt.Errorf("invitation: token must not be empty")
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("invitation: invalid role %q", role)
	}
	if now.IsZero() {
		return nil, fmt.Errorf("invitation: now must be a valid time")
	}
	if expiresAt.IsZero() {
		return nil, fmt.Errorf("invitation: expiresAt must be a valid time")
	}
	if expiresAt.Before(now) || expiresAt.Equal(now) {
		return nil, fmt.Errorf("invitation: expiresAt must be after now")
	}
	return &Invitation{
		email:     email,
		token:     token,
		role:      role,
		status:    InvitationPending,
		createdBy: createdBy,
		expiresAt: expiresAt,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// MustNewInvitation is like NewInvitation but panics on error. Intended
// for tests and seed data where the inputs are known to be valid.
func MustNewInvitation(id, createdBy int, email, token string, role UserRole, status InvitationStatus, expiresAt, createdAt, updatedAt time.Time) *Invitation {
	inv, err := NewInvitation(id, createdBy, email, token, role, status, expiresAt, createdAt, updatedAt)
	if err != nil {
		panic(err)
	}
	return inv
}

func (inv *Invitation) ID() int                  { return inv.id }
func (inv *Invitation) Email() string            { return inv.email }
func (inv *Invitation) Token() string            { return inv.token }
func (inv *Invitation) Role() UserRole           { return inv.role }
func (inv *Invitation) Status() InvitationStatus { return inv.status }
func (inv *Invitation) CreatedBy() int           { return inv.createdBy }
func (inv *Invitation) ExpiresAt() time.Time     { return inv.expiresAt }
func (inv *Invitation) CreatedAt() time.Time     { return inv.createdAt }
func (inv *Invitation) UpdatedAt() time.Time     { return inv.updatedAt }

// IsExpired reports whether the invitation has expired relative to now.
func (inv *Invitation) IsExpired(now time.Time) bool {
	return now.After(inv.expiresAt)
}

// CanBeUsed reports whether the invitation is currently usable: still
// PENDING and not expired.
func (inv *Invitation) CanBeUsed(now time.Time) bool {
	return inv.status == InvitationPending && !inv.IsExpired(now)
}

// Accept transitions a pending invitation to ACCEPTED. Returns an error
// if the invitation is not in PENDING status.
func (inv *Invitation) Accept() error {
	if inv.status != InvitationPending {
		return fmt.Errorf("invitation: cannot accept, current status is %s", inv.status)
	}
	inv.status = InvitationAccepted
	return nil
}

// Revoke transitions a pending invitation to REVOKED. Returns an error
// if the invitation is not in PENDING status.
func (inv *Invitation) Revoke() error {
	if inv.status != InvitationPending {
		return fmt.Errorf("invitation: cannot revoke, current status is %s", inv.status)
	}
	inv.status = InvitationRevoked
	return nil
}

// InvitationRepository is the persistence contract for Invitation.
// Implemented by internal/repository/postgres.
type InvitationRepository interface {
	Create(ctx context.Context, inv *Invitation) (int, error)
	GetByID(ctx context.Context, id int) (*Invitation, error)
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	Update(ctx context.Context, inv *Invitation) error
	ListPending(ctx context.Context, limit, offset int) ([]*Invitation, error)
	ListByEmail(ctx context.Context, email string) ([]*Invitation, error)
}
