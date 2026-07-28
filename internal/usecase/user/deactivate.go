package user

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// DeactivateUserInput is the validated command to soft-delete a user
// by flipping their is_active flag to false. All fields are private.
type DeactivateUserInput struct {
	id int
}

func (in DeactivateUserInput) ID() int { return in.id }

// NewDeactivateUserInput validates id > 0.
func NewDeactivateUserInput(id int) (DeactivateUserInput, error) {
	if id <= 0 {
		return DeactivateUserInput{}, &ValidationError{Field: "id"}
	}
	return DeactivateUserInput{id: id}, nil
}

// MustNewDeactivateUserInput panics on validation error. For tests.
func MustNewDeactivateUserInput(id int) DeactivateUserInput {
	in, err := NewDeactivateUserInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// DeactivateUserOutput carries the deactivated user id.
type DeactivateUserOutput struct {
	ID int
}

// DeactivateUserUseCase flips a user's is_active flag to false via the
// domain method User.Deactivate(). The operation is idempotent:
// deactivating an already-inactive user is a no-op and does not call
// repo.Update.
type DeactivateUserUseCase struct {
	repo domain.UserRepository
}

func NewDeactivateUserUseCase(repo domain.UserRepository) *DeactivateUserUseCase {
	return &DeactivateUserUseCase{repo: repo}
}

func (uc *DeactivateUserUseCase) Execute(ctx context.Context, input DeactivateUserInput) (DeactivateUserOutput, error) {
	user, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return DeactivateUserOutput{}, fmt.Errorf("get user %d: %w", input.ID(), err)
	}
	if user == nil {
		return DeactivateUserOutput{}, ErrNotFound
	}

	if !user.IsActive() {
		return DeactivateUserOutput{ID: user.ID()}, nil
	}

	user.Deactivate()
	if err := uc.repo.Update(ctx, user); err != nil {
		return DeactivateUserOutput{}, fmt.Errorf("deactivate user %d: %w", input.ID(), err)
	}
	return DeactivateUserOutput{ID: user.ID()}, nil
}
