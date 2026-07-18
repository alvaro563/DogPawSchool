package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestSetDogHeatUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		t.Run("zero_id", func(t *testing.T) {
			uc := NewSetDogHeatUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 0, Heat: true})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "id", verr.Field)
		})
		t.Run("negative_id", func(t *testing.T) {
			uc := NewSetDogHeatUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), SetDogHeatInput{ID: -1, Heat: true})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
		})
	})

	t.Run("not_found", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		_, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 9999, Heat: true})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
	})

	t.Run("happy_path_female_heat_true", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
		var setCalled bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogHeat: func(ctx context.Context, id int, heat bool) error {
				setCalled = true
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		out, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 2, Heat: true})
		assert.NoError(t, err)
		assert.True(t, setCalled, "repo SetDogHeat must be called")
		assert.Equal(t, SetDogHeatOutput{ID: 2, Heat: true, Sex: domain.SexFemale}, out)
	})

	t.Run("happy_path_female_heat_false", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogHeat: func(ctx context.Context, id int, heat bool) error {
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		out, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 2, Heat: false})
		assert.NoError(t, err)
		assert.False(t, out.Heat)
	})

	t.Run("happy_path_male_heat_false", func(t *testing.T) {
		// Male + heat=false is fine (the validation only blocks heat=true on males).
		loadedDog, _ := domain.NewDog(9, "Toby", "Cocker Spaniel", "ES-9", 18,
			domain.SexMale, 9.0, 1)
		var setCalled bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogHeat: func(ctx context.Context, id int, heat bool) error {
				setCalled = true
				assert.False(t, heat)
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		out, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 9, Heat: false})
		assert.NoError(t, err)
		assert.True(t, setCalled)
		assert.Equal(t, domain.SexMale, out.Sex)
	})

	t.Run("rejects_heat_true_on_male", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(7, "Toby", "Beagle", "ES-7", 36,
			domain.SexMale, 12.0, 1)
		var setCalled bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogHeat: func(ctx context.Context, id int, heat bool) error {
				setCalled = true
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		_, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 7, Heat: true})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidHeatForSex),
			"expected ErrInvalidHeatForSex, got %v", err)
		assert.False(t, setCalled, "SetDogHeat must NOT be called when validation fails")
	})

	t.Run("set_heat_error", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
		repoErr := errors.New("connection lost")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogHeat: func(ctx context.Context, id int, heat bool) error {
				return repoErr
			},
		}
		uc := NewSetDogHeatUseCase(mock)
		_, err := uc.Execute(context.Background(), SetDogHeatInput{ID: 2, Heat: true})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
