package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetPassInput is the validated input for fetching a single pass
// by id.
type GetPassInput struct {
	id int
}

func (in GetPassInput) ID() int { return in.id }

// NewGetPassInput validates id > 0.
func NewGetPassInput(id int) (GetPassInput, error) {
	if id <= 0 {
		return GetPassInput{}, &ValidationError{Field: "id"}
	}
	return GetPassInput{id: id}, nil
}

// MustNewGetPassInput panics on validation error. For tests.
func MustNewGetPassInput(id int) GetPassInput {
	in, err := NewGetPassInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// GetPassOutput carries the requested pass.
type GetPassOutput struct {
	Pass *domain.Pass
}

// GetPassUseCase returns a single pass or ErrNotFound.
type GetPassUseCase struct {
	repo domain.PassRepository
}

func NewGetPassUseCase(repo domain.PassRepository) *GetPassUseCase {
	return &GetPassUseCase{repo: repo}
}

func (uc *GetPassUseCase) Execute(ctx context.Context, input GetPassInput) (GetPassOutput, error) {
	pass, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return GetPassOutput{}, fmt.Errorf("get pass %d: %w", input.ID(), err)
	}
	if pass == nil {
		return GetPassOutput{}, ErrNotFound
	}
	return GetPassOutput{Pass: pass}, nil
}
