package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type ListIncompatibilitiesInput struct {
	level *domain.IncompatibilityLevel
}

func (in ListIncompatibilitiesInput) Level() *domain.IncompatibilityLevel { return in.level }

func NewListIncompatibilitiesInput(level *domain.IncompatibilityLevel) (ListIncompatibilitiesInput, error) {
	if level != nil && !level.IsValid() {
		return ListIncompatibilitiesInput{}, &ValidationError{Field: "level"}
	}
	return ListIncompatibilitiesInput{level: level}, nil
}

func MustNewListIncompatibilitiesInput(level *domain.IncompatibilityLevel) ListIncompatibilitiesInput {
	in, err := NewListIncompatibilitiesInput(level)
	if err != nil {
		panic(err)
	}
	return in
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
	out, err := uc.repo.List(ctx, input.Level())
	if err != nil {
		return ListIncompatibilitiesOutput{}, fmt.Errorf("list incompatibilities: %w", err)
	}
	return ListIncompatibilitiesOutput{Incompatibilities: out}, nil
}
