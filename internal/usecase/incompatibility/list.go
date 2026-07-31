package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListIncompatibilitiesInput is the input for listing every
// incompatibility, optionally filtered by level and/or kind.
type ListIncompatibilitiesInput struct {
	level *domain.IncompatibilityLevel
	kind  *domain.IncompatibilityKind
}

func (in ListIncompatibilitiesInput) Level() *domain.IncompatibilityLevel { return in.level }
func (in ListIncompatibilitiesInput) Kind() *domain.IncompatibilityKind   { return in.kind }

// NewListIncompatibilitiesInput validates the optional level and kind
// filters. Passing nil means "no filter".
func NewListIncompatibilitiesInput(level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) (ListIncompatibilitiesInput, error) {
	if level != nil && !level.IsValid() {
		return ListIncompatibilitiesInput{}, &ValidationError{Field: "level"}
	}
	if kind != nil && !kind.IsValid() {
		return ListIncompatibilitiesInput{}, &ValidationError{Field: "kind"}
	}
	return ListIncompatibilitiesInput{level: level, kind: kind}, nil
}

// MustNewListIncompatibilitiesInput panics on validation error. For tests.
func MustNewListIncompatibilitiesInput(level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ListIncompatibilitiesInput {
	in, err := NewListIncompatibilitiesInput(level, kind)
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
// optional level and kind filters.
type ListIncompatibilitiesUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewListIncompatibilitiesUseCase(repo domain.IncompatibilityRepository) *ListIncompatibilitiesUseCase {
	return &ListIncompatibilitiesUseCase{repo: repo}
}

func (uc *ListIncompatibilitiesUseCase) Execute(ctx context.Context, input ListIncompatibilitiesInput) (ListIncompatibilitiesOutput, error) {
	out, err := uc.repo.List(ctx, input.Level(), input.Kind())
	if err != nil {
		return ListIncompatibilitiesOutput{}, fmt.Errorf("list incompatibilities: %w", err)
	}
	return ListIncompatibilitiesOutput{Incompatibilities: out}, nil
}
