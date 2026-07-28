package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	"dogpaw/internal/domain"
)

// basicEmailRegex mirrors the DB-level CHECK on users.email so the
// use case can reject malformed addresses before they reach the
// repository. Same pattern as domain.basicEmailRegex.
var basicEmailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// CreateInvitationInput is the validated command to create a new user
// invitation. All fields are private: the only way to obtain a value
// is NewCreateInvitationInput, which guarantees every invariant holds.
type CreateInvitationInput struct {
	createdBy int
	email     string
	role      domain.UserRole
	now       time.Time
}

func (in CreateInvitationInput) CreatedBy() int        { return in.createdBy }
func (in CreateInvitationInput) Email() string         { return in.email }
func (in CreateInvitationInput) Role() domain.UserRole { return in.role }
func (in CreateInvitationInput) Now() time.Time        { return in.now }

// NewCreateInvitationInput validates createdBy, email (non-empty, format)
// and role. A nil now provider defaults to time.Now.
func NewCreateInvitationInput(createdBy int, email string, role domain.UserRole, now func() time.Time) (CreateInvitationInput, error) {
	if createdBy <= 0 {
		return CreateInvitationInput{}, &ValidationError{Field: "created_by"}
	}
	if email == "" {
		return CreateInvitationInput{}, &ValidationError{Field: "email"}
	}
	if !basicEmailRegex.MatchString(email) {
		return CreateInvitationInput{}, &ValidationError{Field: "email"}
	}
	if !role.IsValid() {
		return CreateInvitationInput{}, &ValidationError{Field: "role"}
	}
	if now == nil {
		now = time.Now
	}
	return CreateInvitationInput{createdBy: createdBy, email: email, role: role, now: now()}, nil
}

// MustNewCreateInvitationInput is like NewCreateInvitationInput but
// panics on error. Intended for tests where inputs are known valid.
func MustNewCreateInvitationInput(createdBy int, email string, role domain.UserRole, now func() time.Time) CreateInvitationInput {
	in, err := NewCreateInvitationInput(createdBy, email, role, now)
	if err != nil {
		panic(err)
	}
	return in
}

// CreateInvitationOutput is the result of a successful invitation
// creation. The Token is the unique, URL-safe token the invitee will
// use to complete registration.
type CreateInvitationOutput struct {
	ID    int
	Token string
}

// invitationTokenSize is the number of random bytes used to generate
// the invitation token. 32 bytes = 256 bits of entropy, hex-encoded
// to 64 characters.
const invitationTokenSize = 32

// CreateInvitationUseCase creates a new user invitation in PENDING
// status with a cryptographically random token and a 48-hour expiry.
// The input is trusted to be valid (validated by
// NewCreateInvitationInput at the boundary); Execute is pure
// orchestration.
type CreateInvitationUseCase struct {
	repo domain.InvitationRepository
}

func NewCreateInvitationUseCase(repo domain.InvitationRepository) *CreateInvitationUseCase {
	return &CreateInvitationUseCase{repo: repo}
}

func (uc *CreateInvitationUseCase) Execute(ctx context.Context, input CreateInvitationInput) (CreateInvitationOutput, error) {
	tokenBytes := make([]byte, invitationTokenSize)
	if _, err := rand.Read(tokenBytes); err != nil {
		return CreateInvitationOutput{}, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := input.Now().Add(48 * time.Hour)

	inv, err := domain.NewPendingInvitation(input.CreatedBy(), input.Email(), token, input.Role(), expiresAt, input.Now())
	if err != nil {
		return CreateInvitationOutput{}, fmt.Errorf("create invitation: %w", err)
	}

	id, err := uc.repo.Create(ctx, inv)
	if err != nil {
		return CreateInvitationOutput{}, fmt.Errorf("create invitation: %w", err)
	}

	return CreateInvitationOutput{ID: id, Token: token}, nil
}
