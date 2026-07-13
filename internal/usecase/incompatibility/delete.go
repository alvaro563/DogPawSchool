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

func (uc *DeleteIncompatibilityUseCase) Execute(ctx context.Context, in DeleteIncompatibilityInput) (DeleteIncompatibilityOutput, error) {
	if in.ID <= 0 {
		return DeleteIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}

	if err := uc.repo.Delete(ctx, in.ID); err != nil {
		return DeleteIncompatibilityOutput{}, fmt.Errorf("delete incompatibility %d: %w", in.ID, err)
	}
	return DeleteIncompatibilityOutput{ID: in.ID}, nil
}
