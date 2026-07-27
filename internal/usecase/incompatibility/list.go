package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListIncompatibilitiesInput is the input for listing every
// incompatibility, optionally filtered by level.
type ListIncompatibilitiesInput struct {
	level *domain.IncompatibilityLevel
}

func (in ListIncompatibilitiesInput) Level() *domain.IncompatibilityLevel { return in.level }

// NewListIncompatibilitiesInput validates the optional level
// filter. Passing nil means "no filter".
func NewListIncompatibilitiesInput(level *domain.IncompatibilityLevel) (ListIncompatibilitiesInput, error) {
	if level != nil && !level.IsValid() {
		return ListIncompatibilitiesInput{}, &ValidationError{Field: "level"}
	}
	return ListIncompatibilitiesInput{level: level}, nil
}

// MustNewListIncompatibilitiesInput panics on validation error. For tests.
func MustNewListIncompatibilitiesInput(level *domain.IncompatibilityLevel) ListIncompatibilitiesInput {
	in, err := NewListIncompatibilitiesInput(level)
	if err != nil {
		panic(err)
	}
	return in
}

// ListIncompatibilitiesOutput carries the result list.
type ListIncompatibilitiesOutput struct {
	Incompatibilities []*domain.Incompatibility
}

// ListIncompatibilitiesUseCase returns all incompatibilities, with
// an optional level filter.
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
