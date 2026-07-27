package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterDogInput {
	return MustNewRegisterDogInput("Buddy", "Labrador", "ES12345", 24, domain.SexMale, 25.0, 1)
}

func TestNewRegisterDogInput(t *testing.T) {
	scenarios := []struct {
		name          string
		factory       func() (RegisterDogInput, error)
		expectedField string
	}{
		{"empty_name", func() (RegisterDogInput, error) { return NewRegisterDogInput("", "x", "x", 1, domain.SexMale, 1, 1) }, "name"},
		{"empty_breed", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "", "x", 1, domain.SexMale, 1, 1) }, "breed"},
		{"zero_age", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "x", 0, domain.SexMale, 1, 1) }, "age_in_months"},
		{"empty_sex", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "x", 1, domain.Sex(""), 1, 1) }, "sex"},
		{"zero_weight", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "x", 1, domain.SexMale, 0, 1) }, "weight_kg"},
		{"empty_passport", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "", 1, domain.SexMale, 1, 1) }, "passport"},
		{"zero_user_id", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "x", 1, domain.SexMale, 1, 0) }, "user_id"},
		{"negative_user_id", func() (RegisterDogInput, error) { return NewRegisterDogInput("x", "x", "x", 1, domain.SexMale, 1, -5) }, "user_id"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := s.factory()
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr), "expected ValidationError, got %T", err)
			assert.Equal(t, s.expectedField, verr.Field)
		})
	}
}

func TestRegisterDogUseCase_Execute(t *testing.T) {
	t.Run("factory_blocks_invalid_before_use_case_runs", func(t *testing.T) {
		// The factory rejects invalid input; the use case never sees it.
		_, err := NewRegisterDogInput("", "", "", 0, domain.Sex(""), 0, 0)
		assert.Error(t, err)
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
