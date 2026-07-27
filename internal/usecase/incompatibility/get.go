package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetIncompatibilityInput is the validated input for fetching a
// single incompatibility by id.
type GetIncompatibilityInput struct {
	id int
}

func (in GetIncompatibilityInput) ID() int { return in.id }

// NewGetIncompatibilityInput validates id > 0.
func NewGetIncompatibilityInput(id int) (GetIncompatibilityInput, error) {
	if id <= 0 {
		return GetIncompatibilityInput{}, &ValidationError{Field: "id"}
	}
	return GetIncompatibilityInput{id: id}, nil
}

// MustNewGetIncompatibilityInput panics on validation error. For tests.
func MustNewGetIncompatibilityInput(id int) GetIncompatibilityInput {
	in, err := NewGetIncompatibilityInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// GetIncompatibilityOutput carries the requested incompatibility.
type GetIncompatibilityOutput struct {
	Incompatibility *domain.Incompatibility
}

// GetIncompatibilityUseCase returns a single incompatibility or
// ErrNotFound.
type GetIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewGetIncompatibilityUseCase(repo domain.IncompatibilityRepository) *GetIncompatibilityUseCase {
	return &GetIncompatibilityUseCase{repo: repo}
}

func (uc *GetIncompatibilityUseCase) Execute(ctx context.Context, input GetIncompatibilityInput) (GetIncompatibilityOutput, error) {
	incompat, err := uc.repo.GetIncompatibilityByID(ctx, input.ID())
	if err != nil {
		return GetIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.ID(), err)
	}
	if incompat == nil {
		return GetIncompatibilityOutput{}, ErrNotFound
	}
	return GetIncompatibilityOutput{Incompatibility: incompat}, nil
}
