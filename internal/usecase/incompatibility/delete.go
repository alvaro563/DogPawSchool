package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type DeleteIncompatibilityInput struct {
	ID int
}

type DeleteIncompatibilityOutput struct {
	ID int
}

type DeleteIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewDeleteIncompatibilityUseCase(repo domain.IncompatibilityRepository) *DeleteIncompatibilityUseCase {
	return &DeleteIncompatibilityUseCase{repo: repo}
}

func (uc *DeleteIncompatibilityUseCase) Execute(ctx context.Context, input DeleteIncompatibilityInput) (DeleteIncompatibilityOutput, error) {
	if input.ID <= 0 {
		return DeleteIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}

	if err := uc.repo.Delete(ctx, input.ID); err != nil {
		return DeleteIncompatibilityOutput{}, fmt.Errorf("delete incompatibility %d: %w", input.ID, err)
	}
	return DeleteIncompatibilityOutput{ID: input.ID}, nil
}
