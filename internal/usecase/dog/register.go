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

func (uc *RegisterDogUseCase) Execute(ctx context.Context, in RegisterDogInput) (RegisterDogOutput, error) {
	if err := in.validate(); err != nil {
		return RegisterDogOutput{}, err
	}

	d, err := domain.NewDog(0, in.Name, in.Breed, in.Passport, in.AgeInMonths, in.Sex, in.WeightKg, in.UserID)
	if err != nil {
		return RegisterDogOutput{}, err
	}

	if err := uc.repo.Create(ctx, d); err != nil {
		return RegisterDogOutput{}, fmt.Errorf("register dog: %w", err)
	}

	return RegisterDogOutput{ID: d.ID()}, nil
}

func (in RegisterDogInput) validate() error {
	if in.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if in.Breed == "" {
		return &ValidationError{Field: "breed"}
	}
	if in.AgeInMonths <= 0 {
		return &ValidationError{Field: "age_in_months"}
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
