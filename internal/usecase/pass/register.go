package pass

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterPassInput is the validated command to create a new
// prepaid pass for a user. All fields are private: only
// NewRegisterPassInput can construct one.
type RegisterPassInput struct {
	numOfSessions int
	price         int
	passType      domain.PassType
	userID        int
	expiresAt     *time.Time
}

func (in RegisterPassInput) NumOfSessions() int        { return in.numOfSessions }
func (in RegisterPassInput) Price() int                { return in.price }
func (in RegisterPassInput) PassType() domain.PassType { return in.passType }
func (in RegisterPassInput) UserID() int               { return in.userID }
func (in RegisterPassInput) ExpiresAt() *time.Time     { return in.expiresAt }

// NewRegisterPassInput is the validating factory. Returns the
// first *ValidationError encountered. The returned input is, by
// construction, always valid.
func NewRegisterPassInput(
	numOfSessions, price int,
	passType domain.PassType,
	userID int,
	expiresAt *time.Time,
) (RegisterPassInput, error) {
	if numOfSessions <= 0 {
		return RegisterPassInput{}, &ValidationError{Field: "num_of_sessions"}
	}
	if price < 0 {
		return RegisterPassInput{}, &ValidationError{Field: "price"}
	}
	if !passType.IsValid() {
		return RegisterPassInput{}, &ValidationError{Field: "pass_type"}
	}
	if userID <= 0 {
		return RegisterPassInput{}, &ValidationError{Field: "user_id"}
	}
	if expiresAt != nil && expiresAt.IsZero() {
		return RegisterPassInput{}, &ValidationError{Field: "expires_at"}
	}
	return RegisterPassInput{
		numOfSessions: numOfSessions, price: price, passType: passType,
		userID: userID, expiresAt: expiresAt,
	}, nil
}

// MustNewRegisterPassInput panics on validation error. For tests.
func MustNewRegisterPassInput(
	numOfSessions, price int,
	passType domain.PassType,
	userID int,
	expiresAt *time.Time,
) RegisterPassInput {
	in, err := NewRegisterPassInput(numOfSessions, price, passType, userID, expiresAt)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterPassOutput is the result of a successful create.
type RegisterPassOutput struct {
	ID int
}

// RegisterPassUseCase creates a new pass for the user identified
// by UserID. id=0 lets the DB assign the new id; createdAt and
// updatedAt are both the server's wall-clock at the moment of the
// API call.
type RegisterPassUseCase struct {
	repo domain.PassRepository
}

func NewRegisterPassUseCase(repo domain.PassRepository) *RegisterPassUseCase {
	return &RegisterPassUseCase{repo: repo}
}

func (uc *RegisterPassUseCase) Execute(ctx context.Context, input RegisterPassInput) (RegisterPassOutput, error) {
	now := time.Now()
	pass, err := domain.NewPass(0, input.NumOfSessions(), input.NumOfSessions(),
		input.Price(), input.PassType(), input.UserID(), now, now, input.ExpiresAt())
	if err != nil {
		return RegisterPassOutput{}, err
	}
	id, err := uc.repo.Create(ctx, pass)
	if err != nil {
		return RegisterPassOutput{}, fmt.Errorf("register pass: %w", err)
	}
	return RegisterPassOutput{ID: id}, nil
}
