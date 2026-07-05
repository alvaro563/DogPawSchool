package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validAddInput() AddDogIncompatibilityInput {
	return AddDogIncompatibilityInput{
		DogID:           42,
		Incompatibility: string(domain.IncompatibilityReactivoMachos),
	}
}

func TestAddDogIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         AddDogIncompatibilityInput
			expectedField string
		}{
			{"zero_dog_id", AddDogIncompatibilityInput{Incompatibility: string(domain.IncompatibilityReactivoMachos)}, "dog_id"},
			{"negative_dog_id", AddDogIncompatibilityInput{DogID: -1, Incompatibility: string(domain.IncompatibilityReactivoMachos)}, "dog_id"},
			{"empty_incompatibility", AddDogIncompatibilityInput{DogID: 1}, "incompatibility"},
			{"unknown_incompatibility", AddDogIncompatibilityInput{DogID: 1, Incompatibility: "DOES_NOT_EXIST"}, "incompatibility"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewAddDogIncompatibilityUseCase(mock)

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
		uc := NewAddDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), AddDogIncompatibilityInput{DogID: 0, Incompatibility: string(domain.IncompatibilityReactivoMachos)})

		assert.Error(t, err)
		assert.False(t, called, "repo should not be called when validation fails")
	})

	t.Run("happy_path_adds_when_not_present", func(t *testing.T) {
		existingDog := &domain.Dog{
			ID: 42,
			Incompatibilities: []domain.Incompatibility{
				domain.IncompatibilityNoToleraCachorros,
			},
		}
		updateCalled := false
		var updatedDog *domain.Dog
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				updatedDog = dog
				return nil
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		out, err := uc.Execute(context.Background(), validAddInput())

		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.True(t, out.Added)
		assert.True(t, updateCalled, "update must be called when a change is made")
		assert.Equal(t, []domain.Incompatibility{
			domain.IncompatibilityNoToleraCachorros,
			domain.IncompatibilityReactivoMachos,
		}, out.Incompatibilities)
		assert.Same(t, existingDog, updatedDog)
	})

	t.Run("idempotent_no_op_when_already_present", func(t *testing.T) {
		existingDog := &domain.Dog{
			ID: 42,
			Incompatibilities: []domain.Incompatibility{
				domain.IncompatibilityReactivoMachos,
			},
		}
		updateCalled := false
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				return nil
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		out, err := uc.Execute(context.Background(), validAddInput())

		assert.NoError(t, err)
		assert.False(t, out.Added, "Added must be false when the value was already present")
		assert.False(t, updateCalled, "update must NOT be called when no state change is needed")
		assert.Equal(t, []domain.Incompatibility{
			domain.IncompatibilityReactivoMachos,
		}, out.Incompatibilities, "the slice must not be mutated on a no-op")
	})

	t.Run("idempotent_double_call_produces_same_state", func(t *testing.T) {
		existingDog := &domain.Dog{ID: 42, Incompatibilities: nil}
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		out1, err := uc.Execute(context.Background(), validAddInput())
		assert.NoError(t, err)
		assert.True(t, out1.Added)
		assert.Len(t, out1.Incompatibilities, 1)

		out2, err := uc.Execute(context.Background(), validAddInput())
		assert.NoError(t, err)
		assert.False(t, out2.Added)
		assert.Len(t, out2.Incompatibilities, 1)
	})

	t.Run("get_by_id_returns_error", func(t *testing.T) {
		repoErr := errors.New("database timeout")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_by_id_returns_nil_dog", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return &domain.Dog{ID: id, Incompatibilities: nil}, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		uc := NewAddDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
