package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required field: %s", e.Field)
}

func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

type RegisterDogInput struct {
	Name        string
	Breed       string
	AgeinMonths int
	Sex         domain.Sex
	WeightKg    float64
	Passport    string
	UserID      int
}

type RegisterDogOutput struct {
	ID int
}

type RegisterDogUseCase struct {
	repo domain.DogRepository
}

func NewRegisterDogUseCase(repo domain.DogRepository) *RegisterDogUseCase {
	return &RegisterDogUseCase{repo: repo}
}

func (uc *RegisterDogUseCase) Execute(ctx context.Context, in RegisterDogInput) (RegisterDogOutput, error) {
	if err := in.validate(); err != nil {
		return RegisterDogOutput{}, err
	}

	d := &domain.Dog{
		Name:        in.Name,
		Breed:       in.Breed,
		AgeinMonths: in.AgeinMonths,
		Sex:         in.Sex,
		WeightKg:    in.WeightKg,
		Passport:    in.Passport,
		UserID:      in.UserID,
		IsActive:    true,
	}

	if err := uc.repo.Create(ctx, d); err != nil {
		return RegisterDogOutput{}, fmt.Errorf("register dog: %w", err)
	}

	return RegisterDogOutput{ID: d.ID}, nil
}

func (in RegisterDogInput) validate() error {
	if in.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if in.Breed == "" {
		return &ValidationError{Field: "breed"}
	}
	if in.AgeinMonths <= 0 {
		return &ValidationError{Field: "agein_months"}
	}
	if in.Sex == "" {
		return &ValidationError{Field: "sex"}
	}
	if in.WeightKg <= 0 {
		return &ValidationError{Field: "weight_kg"}
	}
	if in.Passport == "" {
		return &ValidationError{Field: "passport"}
	}
	if in.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	return nil
}
