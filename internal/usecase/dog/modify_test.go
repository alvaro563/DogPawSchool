package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestDogForModify(t *testing.T) *domain.Dog {
	t.Helper()
	d, err := domain.NewDog(42, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 20.0, 1)
	if err != nil {
		t.Fatalf("newTestDogForModify: %v", err)
	}
	return d
}

func validPatch() domain.DogPatch {
	name := "Buddie"
	return domain.DogPatch{Name: &name}
}

func TestModifyDogUseCase_Execute(t *testing.T) {
	t.Run("validation_zero_id", func(t *testing.T) {
		mock := &mockDogRepository{}
		uc := NewModifyDogUseCase(mock)
		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: 0, Patch: validPatch()})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})

	t.Run("validation_negative_id", func(t *testing.T) {
		mock := &mockDogRepository{}
		uc := NewModifyDogUseCase(mock)
		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: -1, Patch: validPatch()})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
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
		_, _ = uc.Execute(context.Background(), ModifyDogInput{ID: 0})
		assert.False(t, called)
	})

	t.Run("empty_patch_is_noop", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		updateCalled := false
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return existingDog, nil },
			update:  func(ctx context.Context, dog *domain.Dog) error { updateCalled = true; return nil },
		}
		uc := NewModifyDogUseCase(mock)
		out, err := uc.Execute(context.Background(), ModifyDogInput{ID: 42, Patch: domain.DogPatch{}})
		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.False(t, updateCalled, "empty patch must not call repo.Update")
	})

	t.Run("partial_patch_changes_only_targeted_field", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		var updatedDog *domain.Dog
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return existingDog, nil },
			update:  func(ctx context.Context, dog *domain.Dog) error { updatedDog = dog; return nil },
		}
		uc := NewModifyDogUseCase(mock)
		out, err := uc.Execute(context.Background(), ModifyDogInput{ID: 42, Patch: validPatch()})
		assert.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.NotNil(t, updatedDog)
		assert.Equal(t, "Buddie", updatedDog.Name(), "name was patched")
		assert.Equal(t, "Labrador", updatedDog.Breed(), "breed preserved")
		assert.Equal(t, 24, updatedDog.AgeInMonths(), "age preserved")
		assert.Equal(t, domain.SexMale, updatedDog.Sex(), "sex preserved")
		assert.Equal(t, "ES12345", updatedDog.Passport(), "passport preserved")
		assert.Equal(t, 20.0, updatedDog.WeightKg(), "weight preserved")
	})

	t.Run("multi_field_patch", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return existingDog, nil },
			update:  func(ctx context.Context, dog *domain.Dog) error { return nil },
		}
		uc := NewModifyDogUseCase(mock)
		newName := "Luna"
		newBreed := "Husky"
		neutered := true
		_, err := uc.Execute(context.Background(), ModifyDogInput{
			ID: 42,
			Patch: domain.DogPatch{
				Name:     &newName,
				Breed:    &newBreed,
				Neutered: &neutered,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, "Luna", existingDog.Name())
		assert.Equal(t, "Husky", existingDog.Breed())
		assert.True(t, existingDog.Neutered())
		assert.Equal(t, 24, existingDog.AgeInMonths(), "age not in patch, preserved")
	})

	t.Run("patch_with_invalid_name_returns_validation_error", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return existingDog, nil },
		}
		uc := NewModifyDogUseCase(mock)
		empty := ""
		_, err := uc.Execute(context.Background(), ModifyDogInput{
			ID:    42,
			Patch: domain.DogPatch{Name: &empty},
		})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "name", verr.Field)
	})

	t.Run("patch_with_invalid_sex_returns_validation_error", func(t *testing.T) {
		existingDog := newTestDogForModify(t)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return existingDog, nil },
		}
		uc := NewModifyDogUseCase(mock)
		invalidSex := domain.Sex("OTHER")
		_, err := uc.Execute(context.Background(), ModifyDogInput{
			ID:    42,
			Patch: domain.DogPatch{Sex: &invalidSex},
		})
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "sex", verr.Field)
	})

	t.Run("get_by_id_returns_error", func(t *testing.T) {
		repoErr := errors.New("database timeout")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return nil, repoErr },
		}
		uc := NewModifyDogUseCase(mock)
		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: 42, Patch: validPatch()})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("get_by_id_returns_nil_dog", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return nil, nil },
		}
		uc := NewModifyDogUseCase(mock)
		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: 42, Patch: validPatch()})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %T", err)
	})

	t.Run("update_returns_error", func(t *testing.T) {
		repoErr := errors.New("concurrent modification")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) { return newTestDogForModify(t), nil },
			update:  func(ctx context.Context, dog *domain.Dog) error { return repoErr },
		}
		uc := NewModifyDogUseCase(mock)
		_, err := uc.Execute(context.Background(), ModifyDogInput{ID: 42, Patch: validPatch()})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
