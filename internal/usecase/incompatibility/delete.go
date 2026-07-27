package incompatibility

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// DeleteIncompatibilityInput is the validated input for deleting
// an incompatibility by id.
type DeleteIncompatibilityInput struct {
	id int
}

func (in DeleteIncompatibilityInput) ID() int { return in.id }

// NewDeleteIncompatibilityInput validates id > 0.
func NewDeleteIncompatibilityInput(id int) (DeleteIncompatibilityInput, error) {
	if id <= 0 {
		return DeleteIncompatibilityInput{}, &ValidationError{Field: "id"}
	}
	return DeleteIncompatibilityInput{id: id}, nil
}

// MustNewDeleteIncompatibilityInput panics on validation error. For tests.
func MustNewDeleteIncompatibilityInput(id int) DeleteIncompatibilityInput {
	in, err := NewDeleteIncompatibilityInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// DeleteIncompatibilityOutput returns the deleted id.
type DeleteIncompatibilityOutput struct {
	ID int
}

// DeleteIncompatibilityUseCase deletes an incompatibility by id.
type DeleteIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewDeleteIncompatibilityUseCase(repo domain.IncompatibilityRepository) *DeleteIncompatibilityUseCase {
	return &DeleteIncompatibilityUseCase{repo: repo}
}

func (uc *DeleteIncompatibilityUseCase) Execute(ctx context.Context, input DeleteIncompatibilityInput) (DeleteIncompatibilityOutput, error) {
	if err := uc.repo.Delete(ctx, input.ID()); err != nil {
		if errors.Is(err, domain.ErrIncompatibilityInUse) {
			return DeleteIncompatibilityOutput{}, ErrInUse
		}
		if errors.Is(err, domain.ErrNotFound) {
			return DeleteIncompatibilityOutput{}, ErrNotFound
		}
		return DeleteIncompatibilityOutput{}, fmt.Errorf("delete incompatibility %d: %w", input.ID(), err)
	}
	return DeleteIncompatibilityOutput{ID: input.ID()}, nil
}
