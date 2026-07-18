package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestSetDogNeuteredUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		t.Run("zero_id", func(t *testing.T) {
			uc := NewSetDogNeuteredUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: 0, Neutered: true})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "id", verr.Field)
		})
		t.Run("negative_id", func(t *testing.T) {
			uc := NewSetDogNeuteredUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: -5, Neutered: true})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "id", verr.Field)
		})
	})

	t.Run("not_found", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		_, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: 9999, Neutered: true})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
	})

	t.Run("happy_path_true", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(42, "Luna", "Labrador", "ES-1", 24,
			domain.SexFemale, 22.5, 1)
		var setCalled bool
		var capturedID int
		var capturedNeutered bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogNeutered: func(ctx context.Context, id int, neutered bool) error {
				setCalled = true
				capturedID = id
				capturedNeutered = neutered
				return nil
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		out, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: 42, Neutered: true})
		assert.NoError(t, err)
		assert.True(t, setCalled, "repo SetDogNeutered must be called")
		assert.Equal(t, 42, capturedID)
		assert.True(t, capturedNeutered)
		assert.Equal(t, SetDogNeuteredOutput{ID: 42, Neutered: true, Sex: domain.SexFemale}, out)
	})

	t.Run("happy_path_false", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(7, "Toby", "Beagle", "ES-2", 36,
			domain.SexMale, 12.0, 1)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogNeutered: func(ctx context.Context, id int, neutered bool) error {
				return nil
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		out, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: 7, Neutered: false})
		assert.NoError(t, err)
		assert.False(t, out.Neutered)
		assert.Equal(t, domain.SexMale, out.Sex)
	})

	t.Run("set_neutered_error", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(1, "Luna", "Labrador", "ES-1", 24,
			domain.SexFemale, 22.5, 1)
		repoErr := errors.New("connection lost")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			setDogNeutered: func(ctx context.Context, id int, neutered bool) error {
				return repoErr
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		_, err := uc.Execute(context.Background(), SetDogNeuteredInput{ID: 1, Neutered: true})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
