package user

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetUserInput is the validated request to fetch a single user by id.
// All fields are private: only NewGetUserInput can construct one.
type GetUserInput struct {
	id int
}

func (in GetUserInput) ID() int { return in.id }

// NewGetUserInput validates id > 0.
func NewGetUserInput(id int) (GetUserInput, error) {
	if id <= 0 {
		return GetUserInput{}, &ValidationError{Field: "id"}
	}
	return GetUserInput{id: id}, nil
}

// MustNewGetUserInput panics on validation error. For tests.
func MustNewGetUserInput(id int) GetUserInput {
	in, err := NewGetUserInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// GetUserOutput carries the resolved user.
type GetUserOutput struct {
	User *domain.User
}

// GetUserUseCase loads a single user by id. Returns ErrNotFound when
// the row is missing.
type GetUserUseCase struct {
	repo domain.UserRepository
}

func NewGetUserUseCase(repo domain.UserRepository) *GetUserUseCase {
	return &GetUserUseCase{repo: repo}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, input GetUserInput) (GetUserOutput, error) {
	user, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return GetUserOutput{}, fmt.Errorf("get user %d: %w", input.ID(), err)
	}
	if user == nil {
		return GetUserOutput{}, ErrNotFound
	}
	return GetUserOutput{User: user}, nil
}
