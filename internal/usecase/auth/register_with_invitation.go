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
// be at least 8 characters. A nil now provider defaults to time.Now.
func NewRegisterWithInvitationInput(token, name, password string, now func() time.Time) (RegisterWithInvitationInput, error) {
	if token == "" {
		return RegisterWithInvitationInput{}, &ValidationError{Field: "token"}
	}
	if name == "" {
		return RegisterWithInvitationInput{}, &ValidationError{Field: "name"}
	}
	if len(password) < 8 {
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
	// 1. Hash the password outside the transaction (bcrypt is expensive
	// and would hold the row lock unnecessarily).
	hashedPw, err := uc.hasher.Hash(input.Password())
	if err != nil {
		return RegisterWithInvitationOutput{}, fmt.Errorf("hash password: %w", err)
	}

	var user *domain.User

	// 2. Persist user + accept invitation in a single transaction.
	// The invitation row is locked with FOR UPDATE so that a concurrent
	// request with the same token blocks here instead of racing past
	// CanBeUsed.
	err = uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		// a. Load invitation by token with row-level lock.
		inv, err := uc.invitationRepo.GetByTokenForUpdate(txCtx, input.Token())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("get invitation by token: %w", err)
		}

		// b. Validate invitation can still be used. Because the row
		// is locked, no concurrent request can accept or revoke it
		// between this check and the Update below.
		if !inv.CanBeUsed(input.Now()) {
			return domain.ErrInvitationInvalid
		}

		// c. Create the user entity.
		user, err = domain.NewUser(0, input.Name(), inv.Email(), hashedPw, inv.Role())
		if err != nil {
			return fmt.Errorf("build user: %w", err)
		}
		userID, err := uc.userRepo.Create(txCtx, user)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		user, err = domain.NewUser(userID, user.Name(), user.Email(), user.Password(), user.Role())
		if err != nil {
			return fmt.Errorf("rebuild user with db id: %w", err)
		}

		// d. Accept invitation (in-memory) and persist.
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
