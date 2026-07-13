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

func (uc *GetIncompatibilityUseCase) Execute(ctx context.Context, in GetIncompatibilityInput) (GetIncompatibilityOutput, error) {
	if in.ID <= 0 {
		return GetIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}
	incomp, err := uc.repo.GetIncompatibilityByID(ctx, in.ID)
	if err != nil {
		return GetIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", in.ID, err)
	}
	if incomp == nil {
		return GetIncompatibilityOutput{}, ErrNotFound
	}
	return GetIncompatibilityOutput{Incompatibility: incomp}, nil
}
