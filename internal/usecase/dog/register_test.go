package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterDogInput {
	return RegisterDogInput{
		Name:        "Buddy",
		Breed:       "Labrador",
		AgeInMonths: 24,
		Sex:         domain.SexMale,
		WeightKg:    25.0,
		Passport:    "ES12345",
		UserID:      1,
	}
}

func TestRegisterDogUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         RegisterDogInput
			expectedField string
		}{
			{"empty_name", RegisterDogInput{Breed: "x", AgeInMonths: 1, Sex: domain.SexMale, WeightKg: 1, Passport: "x", UserID: 1}, "name"},
			{"empty_breed", RegisterDogInput{Name: "x", AgeInMonths: 1, Sex: domain.SexMale, WeightKg: 1, Passport: "x", UserID: 1}, "breed"},
			{"zero_age", RegisterDogInput{Name: "x", Breed: "x", Sex: domain.SexMale, WeightKg: 1, Passport: "x", UserID: 1}, "age_in_months"},
			{"empty_sex", RegisterDogInput{Name: "x", Breed: "x", AgeInMonths: 1, WeightKg: 1, Passport: "x", UserID: 1}, "sex"},
			{"zero_weight", RegisterDogInput{Name: "x", Breed: "x", AgeInMonths: 1, Sex: domain.SexMale, Passport: "x", UserID: 1}, "weight_kg"},
			{"empty_passport", RegisterDogInput{Name: "x", Breed: "x", AgeInMonths: 1, Sex: domain.SexMale, WeightKg: 1, UserID: 1}, "passport"},
			{"zero_user_id", RegisterDogInput{Name: "x", Breed: "x", AgeInMonths: 1, Sex: domain.SexMale, WeightKg: 1, Passport: "x"}, "user_id"},
			{"negative_user_id", RegisterDogInput{Name: "x", Breed: "x", AgeInMonths: 1, Sex: domain.SexMale, WeightKg: 1, Passport: "x", UserID: -5}, "user_id"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewRegisterDogUseCase(mock)

				_, err := uc.Execute(context.Background(), s.input)

				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr), "expected ValidationError, got %T", err)
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("validation_does_not_call_repo", func(t *testing.T) {
		called := false
		mock := &mockDogRepository{
			create: func(ctx context.Context, dog *domain.Dog) (int, error) {
				called = true
				return 0, nil
			},
		}
		uc := NewRegisterDogUseCase(mock)

		_, err := uc.Execute(context.Background(), RegisterDogInput{Name: ""})

		assert.Error(t, err)
		assert.False(t, called, "repo should not be called when validation fails")
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedDog *domain.Dog
		mock := &mockDogRepository{
			create: func(ctx context.Context, dog *domain.Dog) (int, error) {
				capturedDog = dog
				dog.Activate()
				return 42, nil
			},
		}
		uc := NewRegisterDogUseCase(mock)

		out, err := uc.Execute(context.Background(), validRegisterInput())

		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.NotNil(t, capturedDog)
		assert.Equal(t, "Buddy", capturedDog.Name())
		assert.Equal(t, "Labrador", capturedDog.Breed())
		assert.Equal(t, 24, capturedDog.AgeInMonths())
		assert.Equal(t, domain.SexMale, capturedDog.Sex())
		assert.Equal(t, 25.0, capturedDog.WeightKg())
		assert.Equal(t, "ES12345", capturedDog.Passport())
		assert.Equal(t, 1, capturedDog.UserID())
		assert.True(t, capturedDog.IsActive())
		assert.False(t, capturedDog.Neutered())
		assert.False(t, capturedDog.Heat())
		assert.Empty(t, capturedDog.PhotoURL())
		assert.Empty(t, capturedDog.MedicalNotes())
		assert.Empty(t, capturedDog.EducatorNotes())
		assert.Empty(t, capturedDog.Incompatibilities())
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		repoErr := errors.New("database connection lost")
		mock := &mockDogRepository{
			create: func(ctx context.Context, dog *domain.Dog) (int, error) {
				return 0, repoErr
			},
		}
		uc := NewRegisterDogUseCase(mock)

		_, err := uc.Execute(context.Background(), validRegisterInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr), "expected wrapped error to contain original")
	})
}
