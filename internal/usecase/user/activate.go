package user

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ActivateUserInput is the validated command to reactivate a
// soft-deleted user by flipping their is_active flag back to true.
// All fields are private.
type ActivateUserInput struct {
	id int
}

func (in ActivateUserInput) ID() int { return in.id }

// NewActivateUserInput validates id > 0.
func NewActivateUserInput(id int) (ActivateUserInput, error) {
	if id <= 0 {
		return ActivateUserInput{}, &ValidationError{Field: "id"}
	}
	return ActivateUserInput{id: id}, nil
}

// MustNewActivateUserInput panics on validation error. For tests.
func MustNewActivateUserInput(id int) ActivateUserInput {
	in, err := NewActivateUserInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// ActivateUserOutput carries the activated user id.
type ActivateUserOutput struct {
	ID int
}

// ActivateUserUseCase flips a user's is_active flag back to true via
// the domain method User.Activate(). The operation is idempotent:
// activating an already-active user is a no-op and does not call
// repo.Update.
type ActivateUserUseCase struct {
	repo domain.UserRepository
}

func NewActivateUserUseCase(repo domain.UserRepository) *ActivateUserUseCase {
	return &ActivateUserUseCase{repo: repo}
}

func (uc *ActivateUserUseCase) Execute(ctx context.Context, input ActivateUserInput) (ActivateUserOutput, error) {
	user, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return ActivateUserOutput{}, fmt.Errorf("get user %d: %w", input.ID(), err)
	}
	if user == nil {
		return ActivateUserOutput{}, ErrNotFound
	}

	if user.IsActive() {
		return ActivateUserOutput{ID: user.ID()}, nil
	}

	user.Activate()
	if err := uc.repo.Update(ctx, user); err != nil {
		return ActivateUserOutput{}, fmt.Errorf("activate user %d: %w", input.ID(), err)
	}
	return ActivateUserOutput{ID: user.ID()}, nil
}
