package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type GetIncompatibilityInput struct {
	ID int
}

type GetIncompatibilityOutput struct {
	Incompatibility *domain.Incompatibility
}

type GetIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewGetIncompatibilityUseCase(repo domain.IncompatibilityRepository) *GetIncompatibilityUseCase {
	return &GetIncompatibilityUseCase{repo: repo}
}

func (uc *GetIncompatibilityUseCase) Execute(ctx context.Context, input GetIncompatibilityInput) (GetIncompatibilityOutput, error) {
	if input.ID <= 0 {
		return GetIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}
	incompat, err := uc.repo.GetIncompatibilityByID(ctx, input.ID)
	if err != nil {
		return GetIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.ID, err)
	}
	if incompat == nil {
		return GetIncompatibilityOutput{}, ErrNotFound
	}
	return GetIncompatibilityOutput{Incompatibility: incompat}, nil
}
