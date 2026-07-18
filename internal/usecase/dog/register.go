package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type RegisterDogInput struct {
	Name        string
	Breed       string
	AgeInMonths int
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

func (uc *RegisterDogUseCase) Execute(ctx context.Context, input RegisterDogInput) (RegisterDogOutput, error) {
	if err := input.validate(); err != nil {
		return RegisterDogOutput{}, err
	}

	dog, err := domain.NewDog(0, input.Name, input.Breed, input.Passport, input.AgeInMonths, input.Sex, input.WeightKg, input.UserID)
	if err != nil {
		return RegisterDogOutput{}, err
	}

	id, err := uc.repo.Create(ctx, dog)
	if err != nil {
		return RegisterDogOutput{}, fmt.Errorf("register dog: %w", err)
	}

	return RegisterDogOutput{ID: id}, nil
}

func (input RegisterDogInput) validate() error {
	if input.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if input.Breed == "" {
		return &ValidationError{Field: "breed"}
	}
	if input.AgeInMonths <= 0 {
		return &ValidationError{Field: "age_in_months"}
	}
	if input.Sex == "" {
		return &ValidationError{Field: "sex"}
	}
	if input.WeightKg <= 0 {
		return &ValidationError{Field: "weight_kg"}
	}
	if input.Passport == "" {
		return &ValidationError{Field: "passport"}
	}
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	return nil
}
