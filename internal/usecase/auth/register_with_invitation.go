package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterWithInvitationInput is the validated command to register a
// new user from a pending invitation. All fields are private: the
// only way to obtain a value is NewRegisterWithInvitationInput.
type RegisterWithInvitationInput struct {
	token    string
	name     string
	password string
	now      time.Time
}

func (in RegisterWithInvitationInput) Token() string    { return in.token }
func (in RegisterWithInvitationInput) Name() string     { return in.name }
func (in RegisterWithInvitationInput) Password() string { return in.password }
func (in RegisterWithInvitationInput) Now() time.Time   { return in.now }

// NewRegisterWithInvitationInput validates the fields. Password must
// be at least 60 characters (matching the DB CHECK on users.password).
// A nil now provider defaults to time.Now.
func NewRegisterWithInvitationInput(token, name, password string, now func() time.Time) (RegisterWithInvitationInput, error) {
	if token == "" {
		return RegisterWithInvitationInput{}, &ValidationError{Field: "token"}
	}
	if name == "" {
		return RegisterWithInvitationInput{}, &ValidationError{Field: "name"}
	}
	if len(password) < 60 {
		return RegisterWithInvitationInput{}, &ValidationError{Field: "password"}
	}
	if now == nil {
		now = time.Now
	}
	return RegisterWithInvitationInput{
		token:    token,
		name:     name,
		password: password,
		now:      now(),
	}, nil
}

// MustNewRegisterWithInvitationInput is like
// NewRegisterWithInvitationInput but panics on error. Intended for
// tests where inputs are known valid.
func MustNewRegisterWithInvitationInput(token, name, password string, now func() time.Time) RegisterWithInvitationInput {
	in, err := NewRegisterWithInvitationInput(token, name, password, now)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterWithInvitationOutput is the result of a successful
// registration via invitation.
type RegisterWithInvitationOutput struct {
	User *domain.User
}

// RegisterWithInvitationUseCase registers a new user from a pending
// invitation. The flow is:
//  1. Load the invitation by token.
//  2. Validate that it can still be used (PENDING, not expired).
//  3. Hash the password with bcrypt.
//  4. Create the User entity with the invitation's email and role.
//  5. In a single transaction: save the user, accept the invitation.
type RegisterWithInvitationUseCase struct {
	transactor     Transactor
	invitationRepo domain.InvitationRepository
	userRepo       domain.UserRepository
	hasher         PasswordHasher
}

func NewRegisterWithInvitationUseCase(
	transactor Transactor,
	invitationRepo domain.InvitationRepository,
	userRepo domain.UserRepository,
	hasher PasswordHasher,
) *RegisterWithInvitationUseCase {
	return &RegisterWithInvitationUseCase{
		transactor:     transactor,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		hasher:         hasher,
	}
}

func (uc *RegisterWithInvitationUseCase) Execute(ctx context.Context, input RegisterWithInvitationInput) (RegisterWithInvitationOutput, error) {
	// 1. Load invitation by token.
	inv, err := uc.invitationRepo.GetByToken(ctx, input.Token())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RegisterWithInvitationOutput{}, ErrNotFound
		}
		return RegisterWithInvitationOutput{}, fmt.Errorf("get invitation by token: %w", err)
	}

	// 2. Validate invitation can be used.
	if !inv.CanBeUsed(input.Now()) {
		return RegisterWithInvitationOutput{}, domain.ErrInvitationInvalid
	}

	// 3. Hash the password. The algorithm is an infrastructure
	// decision injected as PasswordHasher; this layer only knows that
	// the plaintext must never reach the repository.
	hashedPw, err := uc.hasher.Hash(input.Password())
	if err != nil {
		return RegisterWithInvitationOutput{}, fmt.Errorf("hash password: %w", err)
	}

	// 4. Create user entity.
	user, err := domain.NewUser(0, input.Name(), inv.Email(), hashedPw, inv.Role())
	if err != nil {
		return RegisterWithInvitationOutput{}, fmt.Errorf("build user: %w", err)
	}

	// 5. Persist user + accept invitation in a transaction.
	err = uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Create(txCtx, user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := inv.Accept(); err != nil {
			return fmt.Errorf("accept invitation: %w", err)
		}
		if err := uc.invitationRepo.Update(txCtx, inv); err != nil {
			return fmt.Errorf("update invitation: %w", err)
		}
		return nil
	})
	if err != nil {
		return RegisterWithInvitationOutput{}, err
	}

	return RegisterWithInvitationOutput{User: user}, nil
}
