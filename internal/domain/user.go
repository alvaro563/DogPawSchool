package domain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleRegular UserRole = "REGULAR"
)

func (role UserRole) IsValid() bool {
	switch role {
	case RoleAdmin, RoleRegular:
		return true
	}
	return false
}

type User struct {
	id        int
	name      string
	email     string
	password  string
	role      UserRole
	isActive  bool
	updatedAt time.Time
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

// UpdatedAt returns the last time the user was modified. The database
// sets this automatically via a BEFORE UPDATE trigger; the domain model
// may also bump it through MarkUpdated.
func (user *User) UpdatedAt() time.Time { return user.updatedAt }

// SetPassword replaces the stored password hash. Use cases should call
// MarkUpdated afterward so the domain timestamp stays in sync with the
// DB trigger.
func (user *User) SetPassword(password string) {
	user.password = password
}

// MarkUpdated sets updatedAt to the provided time. Intended to be called
// after any mutation that the DB trigger will also reflect.
func (user *User) MarkUpdated(now time.Time) {
	user.updatedAt = now
}

// UserPatch is a partial update for User: only the non-nil fields are
// applied. Each field has its own validation rules; see ApplyPatch.
type UserPatch struct {
	Name  *string
	Email *string
}

// IsEmpty reports whether the patch would not change any field. Use cases
// short-circuit on an empty patch to avoid touching the DB.
func (patch UserPatch) IsEmpty() bool {
	return patch.Name == nil && patch.Email == nil
}

// UserValidationError is returned by ApplyPatch when a supplied value is
// invalid (empty after trim, malformed email, etc.).
type UserValidationError struct {
	Field string
}

func (validationError *UserValidationError) Error() string {
	return fmt.Sprintf("user: invalid value for %s", validationError.Field)
}

// basicEmailRegex mirrors the DB-level CHECK on users.email so the
// domain can reject malformed addresses before they ever reach the
// repository.
var basicEmailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ApplyPatch mutates the user in place with the fields present in
// patch. An empty patch is a no-op. Returns a *UserValidationError
// identifying the offending field, or nil.
func (user *User) ApplyPatch(patch UserPatch) error {
	if patch.Name != nil {
		trimmed := strings.TrimSpace(*patch.Name)
		if trimmed == "" {
			return &UserValidationError{Field: "name"}
		}
		user.name = trimmed
	}
	if patch.Email != nil {
		trimmed := strings.TrimSpace(*patch.Email)
		if trimmed == "" {
			return &UserValidationError{Field: "email"}
		}
		if !basicEmailRegex.MatchString(trimmed) {
			return &UserValidationError{Field: "email"}
		}
		user.email = trimmed
	}
	return nil
}

// UserRepository is the persistence contract for User. Implemented by
// internal/repository/postgres (future).
type UserRepository interface {
	Create(ctx context.Context, user *User) (int, error)
	Update(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	ListAllPaged(ctx context.Context, limit, offset int) ([]*User, error)
	ListAllEmails(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, id int) error
}
