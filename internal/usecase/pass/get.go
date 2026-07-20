package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetPassInput is the input for fetching a single pass by id.
type GetPassInput struct {
	ID int
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
	if input.ID <= 0 {
		return GetPassOutput{}, &ValidationError{Field: "id"}
	}
	pass, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return GetPassOutput{}, fmt.Errorf("get pass %d: %w", input.ID, err)
	}
	if pass == nil {
		return GetPassOutput{}, ErrNotFound
	}
	return GetPassOutput{Pass: pass}, nil
}
