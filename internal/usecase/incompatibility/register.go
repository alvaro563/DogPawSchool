package incompatibility

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type RegisterIncompatibilityInput struct {
	Name  string
	Level domain.IncompatibilityLevel
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

func (uc *RegisterIncompatibilityUseCase) Execute(ctx context.Context, in RegisterIncompatibilityInput) (RegisterIncompatibilityOutput, error) {
	if err := in.validate(); err != nil {
		return RegisterIncompatibilityOutput{}, err
	}

	incomp, err := domain.NewIncompatibility(0, in.Name, in.Level)
	if err != nil {
		return RegisterIncompatibilityOutput{}, err
	}

	id, err := uc.repo.Create(ctx, incomp)
	if err != nil {
		return RegisterIncompatibilityOutput{}, fmt.Errorf("register incompatibility: %w", err)
	}
	return RegisterIncompatibilityOutput{ID: id}, nil
}

func (in RegisterIncompatibilityInput) validate() error {
	if in.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if !in.Level.IsValid() {
		return &ValidationError{Field: "level"}
	}
	return nil
}
