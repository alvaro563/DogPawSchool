package pass

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterPassInput is the validated payload for creating a new
// prepaid pass (bono) for a specific user.
type RegisterPassInput struct {
	NumOfSessions int
	Price         int
	PassType      domain.PassType
	UserID        int
	ExpiresAt     *time.Time
}

// RegisterPassOutput is the result of a successful create.
type RegisterPassOutput struct {
	ID int
}

// RegisterPassUseCase creates a new pass for the user identified by
// UserID. It validates the input, builds a domain.Pass (with the DB
// generating the id and createdAt timestamp), and asks the
// repository to persist it.
type RegisterPassUseCase struct {
	repo domain.PassRepository
}

func NewRegisterPassUseCase(repo domain.PassRepository) *RegisterPassUseCase {
	return &RegisterPassUseCase{repo: repo}
}

func (uc *RegisterPassUseCase) Execute(ctx context.Context, input RegisterPassInput) (RegisterPassOutput, error) {
	if err := input.validate(); err != nil {
		return RegisterPassOutput{}, err
	}

	// id=0 lets the DB assign the new id; createdAt and updatedAt
	// are both the server's wall-clock at the moment of the API call
	// (the DB trigger keeps updatedAt in sync on future UPDATEs).
	now := time.Now()
	pass, err := domain.NewPass(0, input.NumOfSessions, input.NumOfSessions, input.Price, input.PassType, input.UserID, now, now, input.ExpiresAt)
	if err != nil {
		return RegisterPassOutput{}, err
	}

	id, err := uc.repo.Create(ctx, pass)
	if err != nil {
		return RegisterPassOutput{}, fmt.Errorf("register pass: %w", err)
	}
	return RegisterPassOutput{ID: id}, nil
}

func (input RegisterPassInput) validate() error {
	if input.NumOfSessions <= 0 {
		return &ValidationError{Field: "num_of_sessions"}
	}
	if input.Price < 0 {
		return &ValidationError{Field: "price"}
	}
	if !input.PassType.IsValid() {
		return &ValidationError{Field: "pass_type"}
	}
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	if input.ExpiresAt != nil && input.ExpiresAt.IsZero() {
		return &ValidationError{Field: "expires_at"}
	}
	return nil
}
