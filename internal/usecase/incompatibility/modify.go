package incompatibility

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type ModifyIncompatibilityInput struct {
	id    int
	patch domain.IncompatibilityPatch
}

func (in ModifyIncompatibilityInput) ID() int                            { return in.id }
func (in ModifyIncompatibilityInput) Patch() domain.IncompatibilityPatch { return in.patch }

func NewModifyIncompatibilityInput(id int, patch domain.IncompatibilityPatch) (ModifyIncompatibilityInput, error) {
	if id <= 0 {
		return ModifyIncompatibilityInput{}, &ValidationError{Field: "id"}
	}
	return ModifyIncompatibilityInput{id: id, patch: patch}, nil
}

func MustNewModifyIncompatibilityInput(id int, patch domain.IncompatibilityPatch) ModifyIncompatibilityInput {
	in, err := NewModifyIncompatibilityInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
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
	incompat, err := uc.repo.GetIncompatibilityByID(ctx, input.ID())
	if err != nil {
		return ModifyIncompatibilityOutput{}, fmt.Errorf("get incompatibility %d: %w", input.ID(), err)
	}
	if incompat == nil {
		return ModifyIncompatibilityOutput{}, ErrNotFound
	}

	patch := input.Patch()
	if patch.TargetTraitCode != nil && incompat.TargetTraitCode() != "" {
		if err := validateTriggerTarget(ctx, uc.repo, *patch.TargetTraitCode); err != nil {
			return ModifyIncompatibilityOutput{}, err
		}
	}
	if err := incompat.ApplyPatch(patch); err != nil {
		var validationErr *domain.IncompatibilityValidationError
		if errors.As(err, &validationErr) {
			return ModifyIncompatibilityOutput{}, &ValidationError{Field: validationErr.Field}
		}
		return ModifyIncompatibilityOutput{}, err
	}

	if isEmptyIncompatibilityPatch(patch) {
		return ModifyIncompatibilityOutput{Incompatibility: incompat}, nil
	}

	if err := uc.repo.Update(ctx, incompat); err != nil {
		if errors.Is(err, domain.ErrDuplicateIncompatibilityName) {
			return ModifyIncompatibilityOutput{}, ErrDuplicateName
		}
		if errors.Is(err, domain.ErrDuplicateIncompatibilityCode) {
			return ModifyIncompatibilityOutput{}, ErrDuplicateCode
		}
		return ModifyIncompatibilityOutput{}, fmt.Errorf("update incompatibility %d: %w", input.ID(), err)
	}
	return ModifyIncompatibilityOutput{Incompatibility: incompat}, nil
}

func isEmptyIncompatibilityPatch(patch domain.IncompatibilityPatch) bool {
	return patch.Name == nil && patch.Level == nil &&
		patch.Code == nil && patch.TargetTraitCode == nil
}
