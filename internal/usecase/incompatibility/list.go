package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type ListIncompatibilitiesInput struct {
	Level *domain.IncompatibilityLevel
}

type ListIncompatibilitiesOutput struct {
	Incompatibilities []*domain.Incompatibility
}

type ListIncompatibilitiesUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewListIncompatibilitiesUseCase(repo domain.IncompatibilityRepository) *ListIncompatibilitiesUseCase {
	return &ListIncompatibilitiesUseCase{repo: repo}
}

func (uc *ListIncompatibilitiesUseCase) Execute(ctx context.Context, input ListIncompatibilitiesInput) (ListIncompatibilitiesOutput, error) {
	if input.Level != nil && !input.Level.IsValid() {
		return ListIncompatibilitiesOutput{}, &ValidationError{Field: "level"}
	}
	out, err := uc.repo.List(ctx, input.Level)
	if err != nil {
		return ListIncompatibilitiesOutput{}, fmt.Errorf("list incompatibilities: %w", err)
	}
	return ListIncompatibilitiesOutput{Incompatibilities: out}, nil
}
