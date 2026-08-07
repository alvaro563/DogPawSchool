package incompatibility

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// RegisterIncompatibilityInput is the validated command to create a new
// incompatibility. A trait is identified by code != ""; a trigger by
// targetTraitCode != "". All fields are private: only
// NewRegisterIncompatibilityInput can construct one.
type RegisterIncompatibilityInput struct {
	name            string
	level           domain.IncompatibilityLevel
	code            string
	targetTraitCode string
}

func (in RegisterIncompatibilityInput) Name() string                       { return in.name }
func (in RegisterIncompatibilityInput) Level() domain.IncompatibilityLevel { return in.level }
func (in RegisterIncompatibilityInput) Code() string                       { return in.code }
func (in RegisterIncompatibilityInput) TargetTraitCode() string            { return in.targetTraitCode }

// NewRegisterIncompatibilityInput is the validating factory.
func NewRegisterIncompatibilityInput(
	name string,
	level domain.IncompatibilityLevel,
	code, targetTraitCode string,
) (RegisterIncompatibilityInput, error) {
	if name == "" {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "name"}
	}
	if !level.IsValid() {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "level"}
	}
	if code != "" && targetTraitCode != "" {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "code"}
	}
	if code == "" && targetTraitCode == "" {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "code"}
	}
	return RegisterIncompatibilityInput{
		name:            name,
		level:           level,
		code:            code,
		targetTraitCode: targetTraitCode,
	}, nil
}

// MustNewRegisterIncompatibilityInput panics on validation error. For tests.
func MustNewRegisterIncompatibilityInput(
	name string,
	level domain.IncompatibilityLevel,
	code, targetTraitCode string,
) RegisterIncompatibilityInput {
	in, err := NewRegisterIncompatibilityInput(name, level, code, targetTraitCode)
	if err != nil {
		panic(err)
	}
	return in
}

type RegisterIncompatibilityOutput struct {
	ID int
}

type RegisterIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewRegisterIncompatibilityUseCase(repo domain.IncompatibilityRepository) *RegisterIncompatibilityUseCase {
	return &RegisterIncompatibilityUseCase{repo: repo}
}

func (uc *RegisterIncompatibilityUseCase) Execute(ctx context.Context, input RegisterIncompatibilityInput) (RegisterIncompatibilityOutput, error) {
	if input.TargetTraitCode() != "" {
		if err := validateTriggerTarget(ctx, uc.repo, input.TargetTraitCode()); err != nil {
			return RegisterIncompatibilityOutput{}, err
		}
	}

	var incompat *domain.Incompatibility
	var err error
	if input.Code() != "" {
		incompat, err = domain.NewTraitIncompatibility(0, input.Code(), input.Name(), input.Level())
	} else {
		incompat, err = domain.NewTriggerIncompatibility(0, input.Name(), input.Level(), input.TargetTraitCode())
	}
	if err != nil {
		return RegisterIncompatibilityOutput{}, err
	}

	id, err := uc.repo.Create(ctx, incompat)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateIncompatibilityName) {
			return RegisterIncompatibilityOutput{}, ErrDuplicateName
		}
		if errors.Is(err, domain.ErrDuplicateIncompatibilityCode) {
			return RegisterIncompatibilityOutput{}, ErrDuplicateCode
		}
		return RegisterIncompatibilityOutput{}, fmt.Errorf("create incompatibility: %w", err)
	}
	return RegisterIncompatibilityOutput{ID: id}, nil
}

func validateTriggerTarget(ctx context.Context, repo domain.IncompatibilityRepository, code string) error {
	if code == "" {
		return &ValidationError{Field: "target_trait_code"}
	}
	trait, err := repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &ValidationError{Field: "target_trait_code"}
		}
		return fmt.Errorf("validate trigger target %q: %w", code, err)
	}
	if trait == nil || trait.Code() == "" {
		return &ValidationError{Field: "target_trait_code"}
	}
	return nil
}
