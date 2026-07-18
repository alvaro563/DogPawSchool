package incompatibility

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type ModifyIncompatibilityInput struct {
	ID    int
	Patch domain.IncompatibilityPatch
}

type ModifyIncompatibilityOutput struct {
	Incompatibility *domain.Incompatibility
}

type ModifyIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewModifyIncompatibilityUseCase(repo domain.IncompatibilityRepository) *ModifyIncompatibilityUseCase {
	return &ModifyIncompatibilityUseCase{repo: repo}
}

func (uc *ModifyIncompatibilityUseCase) Execute(ctx context.Context, input ModifyIncompatibilityInput) (ModifyIncompatibilityOutput, error) {
	if input.ID <= 0 {
		return ModifyIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}

	incompat, err := uc.repo.GetIncompatibilityByID(ctx, input.ID)
	if err != nil {
		return ModifyIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.ID, err)
	}
	if incompat == nil {
		return ModifyIncompatibilityOutput{}, ErrNotFound
	}

	if err := incompat.ApplyPatch(input.Patch); err != nil {
		var validationErr *domain.IncompatibilityValidationError
		if errors.As(err, &validationErr) {
			return ModifyIncompatibilityOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyIncompatibilityOutput{}, err
	}

	if isEmptyIncompatibilityPatch(input.Patch) {
		return ModifyIncompatibilityOutput{Incompatibility: incompat}, nil
	}

	if err := uc.repo.Update(ctx, incompat); err != nil {
		return ModifyIncompatibilityOutput{}, fmt.Errorf("update incompatibility %d: %w", input.ID, err)
	}
	return ModifyIncompatibilityOutput{Incompatibility: incompat}, nil
}

func isEmptyIncompatibilityPatch(patch domain.IncompatibilityPatch) bool {
	return patch.Name == nil && patch.Level == nil
}
