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

func (uc *ListIncompatibilitiesUseCase) Execute(ctx context.Context, in ListIncompatibilitiesInput) (ListIncompatibilitiesOutput, error) {
	if in.Level != nil && !in.Level.IsValid() {
		return ListIncompatibilitiesOutput{}, &ValidationError{Field: "level"}
	}
	out, err := uc.repo.List(ctx, in.Level)
	if err != nil {
		return ListIncompatibilitiesOutput{}, fmt.Errorf("list incompatibilities: %w", err)
	}
	return ListIncompatibilitiesOutput{Incompatibilities: out}, nil
}
