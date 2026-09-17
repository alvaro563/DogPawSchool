package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// RegisterDogInput is the validated command to create a new dog.
// All fields are private: the only way to obtain a value is
// NewRegisterDogInput, which guarantees every invariant holds.
type RegisterDogInput struct {
	name                string
	breed               string
	passport            string
	ageInMonths         int
	sex                 domain.Sex
	weightKg            float64
	userID              int
	hasSpecialCondition bool
}

func (in RegisterDogInput) Name() string      { return in.name }
func (in RegisterDogInput) Breed() string     { return in.breed }
func (in RegisterDogInput) Passport() string  { return in.passport }
func (in RegisterDogInput) AgeInMonths() int  { return in.ageInMonths }
func (in RegisterDogInput) Sex() domain.Sex   { return in.sex }
func (in RegisterDogInput) WeightKg() float64          { return in.weightKg }
func (in RegisterDogInput) UserID() int                { return in.userID }
func (in RegisterDogInput) HasSpecialCondition() bool  { return in.hasSpecialCondition }

// NewRegisterDogInput is the validating factory. It returns the first
// *ValidationError encountered (single-error policy). On success, the
// returned input is, by construction, always valid.
func NewRegisterDogInput(
	name, breed, passport string,
	ageInMonths int,
	sex domain.Sex,
	weightKg float64,
	userID int,
	hasSpecialCondition bool,
) (RegisterDogInput, error) {
	if name == "" {
		return RegisterDogInput{}, &ValidationError{Field: "name"}
	}
	if breed == "" {
		return RegisterDogInput{}, &ValidationError{Field: "breed"}
	}
	if passport == "" {
		return RegisterDogInput{}, &ValidationError{Field: "passport"}
	}
	if ageInMonths <= 0 {
		return RegisterDogInput{}, &ValidationError{Field: "age_in_months"}
	}
	if weightKg <= 0 {
		return RegisterDogInput{}, &ValidationError{Field: "weight_kg"}
	}
	if !sex.IsValid() {
		return RegisterDogInput{}, &ValidationError{Field: "sex"}
	}
	if userID <= 0 {
		return RegisterDogInput{}, &ValidationError{Field: "user_id"}
	}
	return RegisterDogInput{
		name: name, breed: breed, passport: passport,
		ageInMonths: ageInMonths, sex: sex, weightKg: weightKg, userID: userID,
		hasSpecialCondition: hasSpecialCondition,
	}, nil
}

// MustNewRegisterDogInput is like NewRegisterDogInput but panics on
// error. Intended for tests and seed data where inputs are known valid.
func MustNewRegisterDogInput(
	name, breed, passport string,
	ageInMonths int,
	sex domain.Sex,
	weightKg float64,
	userID int,
	hasSpecialCondition bool,
) RegisterDogInput {
	in, err := NewRegisterDogInput(name, breed, passport, ageInMonths, sex, weightKg, userID, hasSpecialCondition)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterDogOutput is the result of a successful create.
type RegisterDogOutput struct {
	ID int
}

// RegisterDogUseCase creates a new dog owned by UserID. The input is
// trusted to be valid (validated by NewRegisterDogInput at the
// boundary); Execute is pure orchestration.
type RegisterDogUseCase struct {
	repo domain.DogRepository
}

func NewRegisterDogUseCase(repo domain.DogRepository) *RegisterDogUseCase {
	return &RegisterDogUseCase{repo: repo}
}

func (uc *RegisterDogUseCase) Execute(ctx context.Context, input RegisterDogInput) (RegisterDogOutput, error) {
	dog, err := domain.NewDog(0, input.Name(), input.Breed(), input.Passport(),
		input.AgeInMonths(), input.Sex(), input.WeightKg(), input.UserID())
	if err != nil {
		return RegisterDogOutput{}, err
	}
	if input.HasSpecialCondition() {
		_ = dog.ApplyPatch(domain.DogPatch{HasSpecialCondition: boolPtr(true)})
	}
	id, err := uc.repo.Create(ctx, dog)
	if err != nil {
		return RegisterDogOutput{}, fmt.Errorf("register dog: %w", err)
	}
	return RegisterDogOutput{ID: id}, nil
}

func boolPtr(v bool) *bool { return &v }
