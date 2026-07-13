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

func (uc *ModifyIncompatibilityUseCase) Execute(ctx context.Context, in ModifyIncompatibilityInput) (ModifyIncompatibilityOutput, error) {
	if in.ID <= 0 {
		return ModifyIncompatibilityOutput{}, &ValidationError{Field: "id"}
	}

	incomp, err := uc.repo.GetIncompatibilityByID(ctx, in.ID)
	if err != nil {
		return ModifyIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", in.ID, err)
	}
	if incomp == nil {
		return ModifyIncompatibilityOutput{}, ErrNotFound
	}

	if err := incomp.ApplyPatch(in.Patch); err != nil {
		var dverr *domain.IncompatibilityValidationError
		if errors.As(err, &dverr) {
			return ModifyIncompatibilityOutput{}, &ValidationError{Field: dverr.Field}
		}
		return ModifyIncompatibilityOutput{}, err
	}

	if isEmptyIncompatibilityPatch(in.Patch) {
		return ModifyIncompatibilityOutput{Incompatibility: incomp}, nil
	}

	if err := uc.repo.Update(ctx, incomp); err != nil {
		return ModifyIncompatibilityOutput{}, fmt.Errorf("update incompatibility %d: %w", in.ID, err)
	}
	return ModifyIncompatibilityOutput{Incompatibility: incomp}, nil
}

func isEmptyIncompatibilityPatch(p domain.IncompatibilityPatch) bool {
	return p.Name == nil && p.Level == nil
}
