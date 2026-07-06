package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

type mockIncompatibilityRepository struct {
	getIncompatibilityByID func(ctx context.Context, id int) (*domain.Incompatibility, error)
}

func (m *mockIncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	if m.getIncompatibilityByID != nil {
		return m.getIncompatibilityByID(ctx, id)
	}
	return nil, nil
}

func validAddInput() AddDogIncompatibilityInput {
	return AddDogIncompatibilityInput{
		DogID:             42,
		IncompatibilityID: 1,
	}
}

func newAddUseCase(dogRepo domain.DogRepository, incompatRepo domain.IncompatibilityRepository) *AddDogIncompatibilityUseCase {
	return NewAddDogIncompatibilityUseCase(dogRepo, incompatRepo)
}

func newTestDogWithIncompatibilities(t *testing.T, incompats ...*domain.Incompatibility) *domain.Dog {
	t.Helper()
	d, err := domain.NewDog(42, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 25.0, 1)
	if err != nil {
		t.Fatalf("newTestDogWithIncompatibilities: %v", err)
	}
	for _, in := range incompats {
		if _, err := d.AddIncompatibility(in); err != nil {
			t.Fatalf("newTestDogWithIncompatibilities: %v", err)
		}
	}
	return d
}

func TestAddDogIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         AddDogIncompatibilityInput
			expectedField string
		}{
			{"zero_dog_id", AddDogIncompatibilityInput{IncompatibilityID: 1}, "dog_id"},
			{"negative_dog_id", AddDogIncompatibilityInput{DogID: -1, IncompatibilityID: 1}, "dog_id"},
			{"zero_incompatibility_id", AddDogIncompatibilityInput{DogID: 1}, "incompatibility_id"},
			{"negative_incompatibility_id", AddDogIncompatibilityInput{DogID: 1, IncompatibilityID: -5}, "incompatibility_id"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				dogMock := &mockDogRepository{}
				incompatMock := &mockIncompatibilityRepository{}
				uc := newAddUseCase(dogMock, incompatMock)

				_, err := uc.Execute(context.Background(), s.input)

				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr), "expected ValidationError, got %T", err)
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("validation_does_not_call_repo", func(t *testing.T) {
		dogCalled := false
		incompatCalled := false
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				dogCalled = true
				return nil, nil
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				incompatCalled = true
				return nil, nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), AddDogIncompatibilityInput{DogID: 0, IncompatibilityID: 1})

		assert.Error(t, err)
		assert.False(t, dogCalled, "dog repo should not be called when validation fails")
		assert.False(t, incompatCalled, "incompatibility repo should not be called when validation fails")
	})

	t.Run("happy_path_adds_when_not_present", func(t *testing.T) {
		existingDog := newTestDogWithIncompatibilities(t,
			newIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia),
		)
		var fetchedIncompatID int
		updateCalled := false
		var updatedDog *domain.Dog
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				updatedDog = dog
				return nil
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				fetchedIncompatID = id
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		out, err := uc.Execute(context.Background(), validAddInput())

		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.True(t, out.Added)
		assert.True(t, updateCalled, "update must be called when a change is made")
		assert.Equal(t, 1, fetchedIncompatID, "should fetch the right IncompatibilityID")
		assert.Len(t, out.Incompatibilities, 2)
		assert.Same(t, existingDog, updatedDog)
	})

	t.Run("idempotent_no_op_when_already_present", func(t *testing.T) {
		existingDog := newTestDogWithIncompatibilities(t, validIncompatibility())
		updateCalled := false
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				return nil
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		out, err := uc.Execute(context.Background(), validAddInput())

		assert.NoError(t, err)
		assert.False(t, out.Added, "Added must be false when the value was already present")
		assert.False(t, updateCalled, "update must NOT be called when no state change is needed")
		assert.Len(t, out.Incompatibilities, 1)
	})

	t.Run("idempotent_double_call_produces_same_state", func(t *testing.T) {
		existingDog, _ := domain.NewDog(42, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 25.0, 1)
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return existingDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		out1, err := uc.Execute(context.Background(), validAddInput())
		assert.NoError(t, err)
		assert.True(t, out1.Added)
		assert.Len(t, out1.Incompatibilities, 1)

		out2, err := uc.Execute(context.Background(), validAddInput())
		assert.NoError(t, err)
		assert.False(t, out2.Added)
		assert.Len(t, out2.Incompatibilities, 1)
	})

	t.Run("get_incompatibility_returns_error", func(t *testing.T) {
		repoErr := errors.New("incompatibility db timeout")
		dogMock := &mockDogRepository{}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return nil, repoErr
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_incompatibility_returns_nil", func(t *testing.T) {
		dogMock := &mockDogRepository{}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return nil, nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incompatibility")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get_dog_returns_error", func(t *testing.T) {
		repoErr := errors.New("dog db timeout")
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, repoErr
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_dog_returns_nil", func(t *testing.T) {
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, nil
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dog")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		dogMock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				emptyDog, _ := domain.NewDog(id, "Buddy", "Lab", "ES1", 24, domain.SexMale, 10.0, 1)
				return emptyDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		incompatMock := &mockIncompatibilityRepository{
			getIncompatibilityByID: func(ctx context.Context, id int) (*domain.Incompatibility, error) {
				return validIncompatibility(), nil
			},
		}
		uc := newAddUseCase(dogMock, incompatMock)

		_, err := uc.Execute(context.Background(), validAddInput())

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
