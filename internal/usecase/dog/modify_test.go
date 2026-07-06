package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validModifyInput() ModifyDogInput {
	return ModifyDogInput{
		ID:            42,
		Neutered:      true,
		Heat:          false,
		WeightKg:      25.0,
		PhotoURL:      "http://example.com/photo.jpg",
		MedicalNotes:  "Healthy",
		EducatorNotes: "Well behaved",
		IsActive:      true,
	}
}

func newTestDogForModify(t *testing.T) *domain.Dog {
	t.Helper()
	d, err := domain.NewDog(42, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 20.0, 1)
	if err != nil {
		t.Fatalf("newTestDogForModify: %v", err)
	}
	return d
}

func TestModifyDogUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ModifyDogInput
			expectedField string
		}{
			{"zero_id", ModifyDogInput{WeightKg: 1.0}, "id"},
			{"negative_id", ModifyDogInput{ID: -1, WeightKg: 1.0}, "id"},
			{"zero_weight", ModifyDogInput{ID: 1}, "weight_kg"},
			{"negative_weight", ModifyDogInput{ID: 1, WeightKg: -5.0}, "weight_kg"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewModifyDogUseCase(mock)

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
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				called = true
				return nil, nil
			},
		}
		uc := NewModifyDogUseCase(mock)

		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: 0, WeightKg: 10})

		assert.Error(t, err)
		assert.False(t, called, "repo should not be called when validation fails")
	})

	t.Run("happy_path_updates_fields_and_preserves_others", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		var updatedDog *domain.Dog
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				assert.Equal(t, 42, id)
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updatedDog = dog
				return nil
			},
		}
		uc := NewModifyDogUseCase(mock)

		out, err := uc.Execute(context.Background(), validModifyInput())

		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.NotNil(t, updatedDog)
		assert.Same(t, existingDog, updatedDog, "the use case should mutate the same entity it fetched")
		assert.True(t, updatedDog.Neutered())
		assert.False(t, updatedDog.Heat())
		assert.Equal(t, 25.0, updatedDog.WeightKg())
		assert.Equal(t, "http://example.com/photo.jpg", updatedDog.PhotoURL())
		assert.Equal(t, "Healthy", updatedDog.MedicalNotes())
		assert.Equal(t, "Well behaved", updatedDog.EducatorNotes())
		assert.True(t, updatedDog.IsActive())
		assert.Equal(t, "Buddy", updatedDog.Name(), "Name should be preserved")
		assert.Equal(t, "Labrador", updatedDog.Breed(), "Breed should be preserved")
		assert.Equal(t, 1, updatedDog.UserID(), "UserID should be preserved")
	})

	t.Run("get_by_id_returns_error", func(t *testing.T) {
		repoErr := errors.New("database timeout")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewModifyDogUseCase(mock)

		_, err := uc.Execute(context.Background(), validModifyInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_by_id_returns_nil_dog", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewModifyDogUseCase(mock)

		_, err := uc.Execute(context.Background(), validModifyInput())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return newTestDogForModify(t), nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		uc := NewModifyDogUseCase(mock)

		_, err := uc.Execute(context.Background(), validModifyInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
