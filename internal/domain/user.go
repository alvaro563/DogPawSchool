package domain

import (
	"context"
	"fmt"
)

type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleRegular UserRole = "REGULAR"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleRegular:
		return true
	}
	return false
}

type User struct {
	id       int
	name     string
	email    string
	password string
	role     UserRole
	isActive bool
}

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

func (u *User) ID() int          { return u.id }
func (u *User) Name() string     { return u.name }
func (u *User) Email() string    { return u.email }
func (u *User) Password() string { return u.password }
func (u *User) Role() UserRole   { return u.role }
func (u *User) IsActive() bool   { return u.isActive }

func (u *User) IsAdmin() bool  { return u.role == RoleAdmin }
func (u *User) CanLogin() bool { return u.isActive && u.role.IsValid() }
func (u *User) Activate()      { u.isActive = true }
func (u *User) Deactivate()    { u.isActive = false }

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	Delete(ctx context.Context, id int) error
}
