package domain

import (
	"context"
	"fmt"
)

// UserRole determines what a User can do in the system.
type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleRegular UserRole = "REGULAR"
)

// IsValid reports whether the value is a recognized UserRole.
func (role UserRole) IsValid() bool {
	switch role {
	case RoleAdmin, RoleRegular:
		return true
	}
	return false
}

// User owns dogs and passes.
type User struct {
	id       int
	name     string
	email    string
	password string
	role     UserRole
	isActive bool
}

// NewUser creates a User. New users start as is_active=true.
func NewUser(id int, name, email, password string, role UserRole) (*User, error) {
	if id < 0 {
		return nil, fmt.Errorf("user: id must not be negative")
	}
	if name == "" {
		return nil, fmt.Errorf("user: name must not be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("user: email must not be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("user: password must not be empty")
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("user: invalid role %q", role)
	}
	return &User{
		id:       id,
		name:     name,
		email:    email,
		password: password,
		role:     role,
		isActive: true,
	}, nil
}

func (user *User) ID() int          { return user.id }
func (user *User) Name() string     { return user.name }
func (user *User) Email() string    { return user.email }
func (user *User) Password() string { return user.password }
func (user *User) Role() UserRole   { return user.role }
func (user *User) IsActive() bool   { return user.isActive }

// IsAdmin reports whether the user has the ADMIN role.
func (user *User) IsAdmin() bool { return user.role == RoleAdmin }

// CanLogin reports whether the user can currently log in: must be active
// and have a valid role.
func (user *User) CanLogin() bool { return user.isActive && user.role.IsValid() }

// Activate marks the user as active.
func (user *User) Activate() { user.isActive = true }

// Deactivate marks the user as inactive (soft delete).
func (user *User) Deactivate() { user.isActive = false }

// UserRepository is the persistence contract for User. Implemented by
// internal/repository/postgres (future).
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	Delete(ctx context.Context, id int) error
}
