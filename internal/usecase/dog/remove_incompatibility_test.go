package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRemoveInput() RemoveDogIncompatibilityInput {
	return RemoveDogIncompatibilityInput{
		DogID:           42,
		Incompatibility: string(domain.IncompatibilityReactivoMachos),
	}
}

func TestRemoveDogIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         RemoveDogIncompatibilityInput
			expectedField string
		}{
			{"zero_dog_id", RemoveDogIncompatibilityInput{Incompatibility: string(domain.IncompatibilityReactivoMachos)}, "dog_id"},
			{"negative_dog_id", RemoveDogIncompatibilityInput{DogID: -1, Incompatibility: string(domain.IncompatibilityReactivoMachos)}, "dog_id"},
			{"empty_incompatibility", RemoveDogIncompatibilityInput{DogID: 1}, "incompatibility"},
			{"unknown_incompatibility", RemoveDogIncompatibilityInput{DogID: 1, Incompatibility: "DOES_NOT_EXIST"}, "incompatibility"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewRemoveDogIncompatibilityUseCase(mock)

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
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), RemoveDogIncompatibilityInput{DogID: 0, Incompatibility: string(domain.IncompatibilityReactivoMachos)})

		assert.Error(t, err)
		assert.False(t, called, "repo should not be called when validation fails")
	})

	t.Run("happy_path_removes_when_present", func(t *testing.T) {
		existingDog := &domain.Dog{
			ID: 42,
			Incompatibilities: []domain.Incompatibility{
				domain.IncompatibilityReactivoMachos,
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
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		out, err := uc.Execute(context.Background(), validRemoveInput())

		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.True(t, out.Removed)
		assert.True(t, updateCalled, "update must be called when a change is made")
		assert.Equal(t, []domain.Incompatibility{
			domain.IncompatibilityNoToleraCachorros,
		}, out.Incompatibilities)
		assert.Same(t, existingDog, updatedDog)
	})

	t.Run("idempotent_no_op_when_not_present", func(t *testing.T) {
		existingDog := &domain.Dog{
			ID: 42,
			Incompatibilities: []domain.Incompatibility{
				domain.IncompatibilityNoToleraCachorros,
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
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		out, err := uc.Execute(context.Background(), validRemoveInput())

		assert.NoError(t, err)
		assert.False(t, out.Removed, "Removed must be false when the value was not present")
		assert.False(t, updateCalled, "update must NOT be called when no state change is needed")
		assert.Equal(t, []domain.Incompatibility{
			domain.IncompatibilityNoToleraCachorros,
		}, out.Incompatibilities, "the slice must not be mutated on a no-op")
	})

	t.Run("idempotent_double_call_produces_same_state", func(t *testing.T) {
		existingDog := &domain.Dog{
			ID: 42,
			Incompatibilities: []domain.Incompatibility{
				domain.IncompatibilityReactivoMachos,
			},
		}
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		out1, err := uc.Execute(context.Background(), validRemoveInput())
		assert.NoError(t, err)
		assert.True(t, out1.Removed)
		assert.Len(t, out1.Incompatibilities, 0)

		out2, err := uc.Execute(context.Background(), validRemoveInput())
		assert.NoError(t, err)
		assert.False(t, out2.Removed)
		assert.Len(t, out2.Incompatibilities, 0)
	})

	t.Run("get_by_id_returns_error", func(t *testing.T) {
		repoErr := errors.New("database timeout")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validRemoveInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_by_id_returns_nil_dog", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validRemoveInput())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return &domain.Dog{ID: id, Incompatibilities: []domain.Incompatibility{domain.IncompatibilityReactivoMachos}}, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		uc := NewRemoveDogIncompatibilityUseCase(mock)

		_, err := uc.Execute(context.Background(), validRemoveInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
