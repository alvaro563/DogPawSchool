package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// LoginInput is the validated command to authenticate a user. All fields
// are private: the only way to obtain a value is NewLoginInput.
type LoginInput struct {
	email    string
	password string
	now      time.Time
}

func (in LoginInput) Email() string    { return in.email }
func (in LoginInput) Password() string { return in.password }
func (in LoginInput) Now() time.Time   { return in.now }

// NewLoginInput validates the fields. Email and password must be
// non-empty. A nil now provider defaults to time.Now.
func NewLoginInput(email, password string, now func() time.Time) (LoginInput, error) {
	if email == "" {
		return LoginInput{}, &ValidationError{Field: "email"}
	}
	if password == "" {
		return LoginInput{}, &ValidationError{Field: "password"}
	}
	if now == nil {
		now = time.Now
	}
	return LoginInput{
		email:    email,
		password: password,
		now:      now(),
	}, nil
}

// MustNewLoginInput is like NewLoginInput but panics on error. Intended
// for tests where inputs are known valid.
func MustNewLoginInput(email, password string, now func() time.Time) LoginInput {
	in, err := NewLoginInput(email, password, now)
	if err != nil {
		panic(err)
	}
	return in
}

// LoginOutput is the result of a successful authentication.
type LoginOutput struct {
	Token string
	User  *domain.User
}

// TokenGenerator creates a signed token that proves the bearer is a
// given user. Implementations live in internal/crypto (e.g. JWT).
type TokenGenerator interface {
	Generate(user *domain.User) (string, error)
}

// LoginUseCase authenticates a user by email + password. On success it
// returns a signed token and the user profile. The flow is:
//
//  1. Look up the user by email.
//  2. Verify the password against the stored hash.
//  3. Check the account is active (CanLogin).
//  4. Generate a signed token.
type LoginUseCase struct {
	userRepo domain.UserRepository
	verifier PasswordVerifier
	tokenGen TokenGenerator
}

func NewLoginUseCase(
	userRepo domain.UserRepository,
	verifier PasswordVerifier,
	tokenGen TokenGenerator,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo: userRepo,
		verifier: verifier,
		tokenGen: tokenGen,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.userRepo.GetByEmail(ctx, input.Email())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginOutput{}, ErrInvalidCredentials
		}
		return LoginOutput{}, fmt.Errorf("lookup user: %w", err)
	}

	if err := uc.verifier.Verify(user.Password(), input.Password()); err != nil {
		return LoginOutput{}, ErrInvalidCredentials
	}

	if !user.CanLogin() {
		return LoginOutput{}, ErrInvalidCredentials
	}

	token, err := uc.tokenGen.Generate(user)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("generate token: %w", err)
	}

	return LoginOutput{Token: token, User: user}, nil
}
