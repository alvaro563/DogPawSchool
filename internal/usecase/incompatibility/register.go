package incompatibility

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// RegisterIncompatibilityInput is the validated command to create
// a new incompatibility. All fields are private: only
// NewRegisterIncompatibilityInput can construct one.
type RegisterIncompatibilityInput struct {
	name  string
	level domain.IncompatibilityLevel
}

func (in RegisterIncompatibilityInput) Name() string                       { return in.name }
func (in RegisterIncompatibilityInput) Level() domain.IncompatibilityLevel { return in.level }

// NewRegisterIncompatibilityInput is the validating factory.
// Returns the first *ValidationError encountered. The returned
// input is, by construction, always valid.
func NewRegisterIncompatibilityInput(name string, level domain.IncompatibilityLevel) (RegisterIncompatibilityInput, error) {
	if name == "" {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "name"}
	}
	if !level.IsValid() {
		return RegisterIncompatibilityInput{}, &ValidationError{Field: "level"}
	}
	return RegisterIncompatibilityInput{name: name, level: level}, nil
}

// MustNewRegisterIncompatibilityInput panics on validation error. For tests.
func MustNewRegisterIncompatibilityInput(name string, level domain.IncompatibilityLevel) RegisterIncompatibilityInput {
	in, err := NewRegisterIncompatibilityInput(name, level)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterIncompatibilityOutput is the result of a successful create.
type RegisterIncompatibilityOutput struct {
	ID int
}

// RegisterIncompatibilityUseCase creates a new incompatibility
// category.
type RegisterIncompatibilityUseCase struct {
	repo domain.IncompatibilityRepository
}

func NewRegisterIncompatibilityUseCase(repo domain.IncompatibilityRepository) *RegisterIncompatibilityUseCase {
	return &RegisterIncompatibilityUseCase{repo: repo}
}

func (uc *RegisterIncompatibilityUseCase) Execute(ctx context.Context, input RegisterIncompatibilityInput) (RegisterIncompatibilityOutput, error) {
	incompat, err := domain.NewIncompatibility(0, input.Name(), input.Level())
	if err != nil {
		return RegisterIncompatibilityOutput{}, err
	}
	id, err := uc.repo.Create(ctx, incompat)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateIncompatibilityName) {
			return RegisterIncompatibilityOutput{}, ErrDuplicateName
		}
		return RegisterIncompatibilityOutput{}, fmt.Errorf("create incompatibility: %w", err)
	}
	return RegisterIncompatibilityOutput{ID: id}, nil
}
