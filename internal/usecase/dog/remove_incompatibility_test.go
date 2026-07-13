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
		DogID:             42,
		IncompatibilityID: 1,
	}
}

func newTestDogForRemove(t *testing.T, incompats ...*domain.Incompatibility) *domain.Dog {
	t.Helper()
	d, err := domain.NewDog(42, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 25.0, 1)
	if err != nil {
		t.Fatalf("newTestDogForRemove: %v", err)
	}
	for _, in := range incompats {
		if _, err := d.AddIncompatibility(in); err != nil {
			t.Fatalf("newTestDogForRemove: %v", err)
		}
	}
	return d
}

func TestRemoveDogIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         RemoveDogIncompatibilityInput
			expectedField string
		}{
			{"zero_dog_id", RemoveDogIncompatibilityInput{IncompatibilityID: 1}, "dog_id"},
			{"negative_dog_id", RemoveDogIncompatibilityInput{DogID: -1, IncompatibilityID: 1}, "dog_id"},
			{"zero_incompatibility_id", RemoveDogIncompatibilityInput{DogID: 1}, "incompatibility_id"},
			{"negative_incompatibility_id", RemoveDogIncompatibilityInput{DogID: 1, IncompatibilityID: -5}, "incompatibility_id"},
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

		_, err := uc.Execute(context.Background(), RemoveDogIncompatibilityInput{DogID: 0, IncompatibilityID: 1})

		assert.Error(t, err)
		assert.False(t, called, "repo should not be called when validation fails")
	})

	t.Run("happy_path_removes_when_present", func(t *testing.T) {
		existingDog := newTestDogForRemove(t,
			validIncompatibility(),
			newIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia),
		)
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
		assert.Len(t, out.Incompatibilities, 1)
		assert.Same(t, existingDog, updatedDog)
	})

	t.Run("idempotent_no_op_when_not_present", func(t *testing.T) {
		existingDog := newTestDogForRemove(t,
			newIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia),
		)
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
		assert.Len(t, out.Incompatibilities, 1)
	})

	t.Run("idempotent_double_call_produces_same_state", func(t *testing.T) {
		existingDog := newTestDogForRemove(t, validIncompatibility())
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
		assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %T", err)
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return newTestDogForRemove(t, validIncompatibility()), nil
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
