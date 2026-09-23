package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"dogpaw/internal/domain"
)

var dummyBcryptHash string

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-mitigation-dummy-password-constant-value"), bcrypt.DefaultCost)
	if err != nil {
		panic("precompute bcrypt dummy hash: " + err.Error())
	}
	dummyBcryptHash = string(h)
}

// LoginInput is the validated command to authenticate a user. All fields
// are private: the only way to obtain a value is NewLoginInput.
type LoginInput struct {
	email    string
	password string
	remoteIP string
	now      time.Time
}

func (in LoginInput) Email() string    { return in.email }
func (in LoginInput) Password() string { return in.password }
func (in LoginInput) RemoteIP() string { return in.remoteIP }
func (in LoginInput) Now() time.Time   { return in.now }

// NewLoginInput validates the fields. Email and password must be
// non-empty. A nil now provider defaults to time.Now.
//
// remoteIP is the TCP peer of the request (RemoteAddr). The use
// case passes it to the AccountLoginLimiter so brute-force
// attempts against the same source can be throttled even when the
// attacker rotates emails. Empty string is allowed in tests but
// MUST be populated by the handler in production.
func NewLoginInput(email, password, remoteIP string, now func() time.Time) (LoginInput, error) {
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
		remoteIP: remoteIP,
		now:      now(),
	}, nil
}

// MustNewLoginInput is like NewLoginInput but panics on error. Intended
// for tests where inputs are known valid.
func MustNewLoginInput(email, password, remoteIP string, now func() time.Time) LoginInput {
	in, err := NewLoginInput(email, password, remoteIP, now)
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
// returns a signed token and the user profile. The flow:
//
//  0. Account lockout check (per email + per remote IP).
//  1. Look up the user by email.
//  2. Verify the password against the stored hash.
//  3. Check the account is active (CanLogin).
//  4. Generate a signed token.
//  5. Record the attempt (success: reset email failures; failure: count).
//
// Step 0 short-circuits the bcrypt check (~250ms saved per blocked
// attempt) and step 5 keeps the counter honest regardless of which
// step the failure occurred at.
type LoginUseCase struct {
	userRepo domain.UserRepository
	verifier PasswordVerifier
	tokenGen TokenGenerator
	lockout  domain.AccountLoginLimiter
}

func NewLoginUseCase(
	userRepo domain.UserRepository,
	verifier PasswordVerifier,
	tokenGen TokenGenerator,
	lockout domain.AccountLoginLimiter,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo: userRepo,
		verifier: verifier,
		tokenGen: tokenGen,
		lockout:  lockout,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	if err := uc.lockout.Check(ctx, input.Email(), input.RemoteIP()); err != nil {
		// Lockout is a domain-level rejection that the handler
		// will map to 429 + Retry-After. We still record the
		// attempt (as a failure) so the limiter sees a steady
		// attack and refuses to release the bucket early.
		_ = uc.lockout.Record(ctx, input.Email(), input.RemoteIP(), false)
		return LoginOutput{}, err
	}

	user, err := uc.userRepo.GetByEmail(ctx, input.Email())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = uc.verifier.Verify(dummyBcryptHash, "timing-mitigation-padding")
			_ = uc.lockout.Record(ctx, input.Email(), input.RemoteIP(), false)
			return LoginOutput{}, ErrInvalidCredentials
		}
		return LoginOutput{}, fmt.Errorf("lookup user: %w", err)
	}

	if err := uc.verifier.Verify(user.Password(), input.Password()); err != nil {
		_ = uc.lockout.Record(ctx, input.Email(), input.RemoteIP(), false)
		return LoginOutput{}, ErrInvalidCredentials
	}

	if !user.CanLogin() {
		_ = uc.lockout.Record(ctx, input.Email(), input.RemoteIP(), false)
		return LoginOutput{}, ErrInvalidCredentials
	}

	token, err := uc.tokenGen.Generate(user)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("generate token: %w", err)
	}

	if err := uc.lockout.Record(ctx, input.Email(), input.RemoteIP(), true); err != nil {
		// We have already authenticated the user — a failed
		// audit-write must not surface as a login error. Log
		// and move on. (Logging is the caller's job; the use
		// case just returns the token.)
		_ = err
	}
	if err := uc.lockout.ResetByEmail(ctx, input.Email()); err != nil {
		_ = err
	}

	return LoginOutput{Token: token, User: user}, nil
}
