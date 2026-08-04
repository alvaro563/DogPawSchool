package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ChangePasswordInput is the validated command to change a user's
// password. All fields are private: the only way to obtain a value
// is NewChangePasswordInput.
type ChangePasswordInput struct {
	userID      int
	oldPassword string
	newPassword string
	now         time.Time
}

func (in ChangePasswordInput) UserID() int      { return in.userID }
func (in ChangePasswordInput) OldPassword() string { return in.oldPassword }
func (in ChangePasswordInput) NewPassword() string { return in.newPassword }
func (in ChangePasswordInput) Now() time.Time      { return in.now }

// NewChangePasswordInput validates the fields. Old password must be
// non-empty; new password must be at least 8 characters. A nil now
// provider defaults to time.Now.
func NewChangePasswordInput(userID int, oldPassword, newPassword string, now func() time.Time) (ChangePasswordInput, error) {
	if userID <= 0 {
		return ChangePasswordInput{}, &ValidationError{Field: "user_id"}
	}
	if oldPassword == "" {
		return ChangePasswordInput{}, &ValidationError{Field: "old_password"}
	}
	if len(newPassword) < 8 {
		return ChangePasswordInput{}, &ValidationError{Field: "new_password"}
	}
	if len(newPassword) > 72 {
		return ChangePasswordInput{}, &ValidationError{Field: "new_password"}
	}
	if now == nil {
		now = time.Now
	}
	return ChangePasswordInput{
		userID:      userID,
		oldPassword: oldPassword,
		newPassword: newPassword,
		now:         now(),
	}, nil
}

// MustNewChangePasswordInput is like NewChangePasswordInput but panics
// on error. Intended for tests where inputs are known valid.
func MustNewChangePasswordInput(userID int, oldPassword, newPassword string, now func() time.Time) ChangePasswordInput {
	in, err := NewChangePasswordInput(userID, oldPassword, newPassword, now)
	if err != nil {
		panic(err)
	}
	return in
}

// ChangePasswordOutput is the (empty) result of a successful password
// change. The handler translates this to a 200 with a message.
type ChangePasswordOutput struct{}

// ChangePasswordUseCase verifies the current password and replaces it
// with a new one. The flow is:
//
//  1. Load the user by ID.
//  2. Verify the old password against the stored hash.
//  3. Check the account is active (CanLogin).
//  4. Reject if the new password equals the old one.
//  5. Hash the new password.
//  6. Update the user's password and bump updatedAt.
//  7. Persist via repository.
type ChangePasswordUseCase struct {
	userRepo domain.UserRepository
	verifier PasswordVerifier
	hasher   PasswordHasher
}

func NewChangePasswordUseCase(
	userRepo domain.UserRepository,
	verifier PasswordVerifier,
	hasher PasswordHasher,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		userRepo: userRepo,
		verifier: verifier,
		hasher:   hasher,
	}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) (ChangePasswordOutput, error) {
	user, err := uc.userRepo.GetByID(ctx, input.UserID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ChangePasswordOutput{}, ErrInvalidCredentials
		}
		return ChangePasswordOutput{}, fmt.Errorf("lookup user: %w", err)
	}

	if err := uc.verifier.Verify(user.Password(), input.OldPassword()); err != nil {
		return ChangePasswordOutput{}, ErrInvalidCredentials
	}

	if !user.CanLogin() {
		return ChangePasswordOutput{}, ErrInvalidCredentials
	}

	if err := uc.verifier.Verify(user.Password(), input.NewPassword()); err == nil {
		return ChangePasswordOutput{}, ErrSamePassword
	}

	hashedPw, err := uc.hasher.Hash(input.NewPassword())
	if err != nil {
		return ChangePasswordOutput{}, fmt.Errorf("hash new password: %w", err)
	}

	user.SetPassword(hashedPw)
	user.IncrementTokenVersion()
	user.MarkUpdated(input.Now())

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return ChangePasswordOutput{}, fmt.Errorf("update user: %w", err)
	}

	return ChangePasswordOutput{}, nil
}
