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

func (uc *RegisterIncompatibilityUseCase) Execute(ctx context.Context, input RegisterIncompatibilityInput) (RegisterIncompatibilityOutput, error) {
	if err := input.validate(); err != nil {
		return RegisterIncompatibilityOutput{}, err
	}

	incompat, err := domain.NewIncompatibility(0, input.Name, input.Level)
	if err != nil {
		return RegisterIncompatibilityOutput{}, err
	}

	id, err := uc.repo.Create(ctx, incompat)
	if err != nil {
		return RegisterIncompatibilityOutput{}, fmt.Errorf("register incompatibility: %w", err)
	}
	return RegisterIncompatibilityOutput{ID: id}, nil
}

func (input RegisterIncompatibilityInput) validate() error {
	if input.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if !input.Level.IsValid() {
		return &ValidationError{Field: "level"}
	}
	return nil
}
