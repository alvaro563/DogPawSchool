package domain

import "context"

type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleRegular UserRole = "REGULAR"
)

type User struct {
	ID       int
	Name     string
	Email    string
	Password string
	Role     UserRole
	IsActive bool
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	Delete(ctx context.Context, id int) error
}
