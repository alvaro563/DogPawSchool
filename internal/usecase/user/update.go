package user

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// UpdateUserInput is the validated command to apply a partial update
// to an existing user. Fields are private: only NewUpdateUserInput
// can construct one. The patch *values* are validated by the domain
// (domain.User.ApplyPatch) — defense in depth.
type UpdateUserInput struct {
	id    int
	patch domain.UserPatch
}

func (in UpdateUserInput) ID() int                 { return in.id }
func (in UpdateUserInput) Patch() domain.UserPatch { return in.patch }

// NewUpdateUserInput validates id > 0. The patch itself is value-only;
// domain validation runs in Execute before persistence.
func NewUpdateUserInput(id int, patch domain.UserPatch) (UpdateUserInput, error) {
	if id <= 0 {
		return UpdateUserInput{}, &ValidationError{Field: "id"}
	}
	return UpdateUserInput{id: id, patch: patch}, nil
}

// MustNewUpdateUserInput panics on validation error. For tests.
func MustNewUpdateUserInput(id int, patch domain.UserPatch) UpdateUserInput {
	in, err := NewUpdateUserInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
}

// UpdateUserOutput carries the post-mutation user id.
type UpdateUserOutput struct {
	ID int
}

// UpdateUserUseCase applies a partial update to a user. An empty
// patch is a no-op and returns the unmodified user without touching
// the DB.
type UpdateUserUseCase struct {
	repo domain.UserRepository
}

func NewUpdateUserUseCase(repo domain.UserRepository) *UpdateUserUseCase {
	return &UpdateUserUseCase{repo: repo}
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (UpdateUserOutput, error) {
	user, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return UpdateUserOutput{}, fmt.Errorf("get user %d: %w", input.ID(), err)
	}
	if user == nil {
		return UpdateUserOutput{}, ErrNotFound
	}

	patch := input.Patch()
	if err := user.ApplyPatch(patch); err != nil {
		var validationErr *domain.UserValidationError
		if errors.As(err, &validationErr) {
			return UpdateUserOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return UpdateUserOutput{}, err
	}

	if patch.IsEmpty() {
		return UpdateUserOutput{ID: user.ID()}, nil
	}

	if err := uc.repo.Update(ctx, user); err != nil {
		return UpdateUserOutput{}, fmt.Errorf("update user %d: %w", input.ID(), err)
	}
	return UpdateUserOutput{ID: user.ID()}, nil
}
