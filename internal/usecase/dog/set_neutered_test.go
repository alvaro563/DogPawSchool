package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewSetDogNeuteredInput(t *testing.T) {
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewSetDogNeuteredInput(0, true)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
	t.Run("negative_id", func(t *testing.T) {
		_, err := NewSetDogNeuteredInput(-5, true)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestSetDogNeuteredUseCase_Execute(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogNeuteredInput(9999, true))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("happy_path_true", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(42, "Luna", "Labrador", "ES-1", 24,
			domain.SexFemale, 22.5, 1)
		var updateCalled bool
		var capturedDog *domain.Dog
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				capturedDog = dog
				return nil
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogNeuteredInput(42, true))
		assert.NoError(t, err)
		assert.True(t, updateCalled, "repo Update must be called")
		assert.True(t, capturedDog.Neutered(), "the persisted aggregate must carry the new flag")
		assert.Equal(t, 42, capturedDog.ID())
		assert.Equal(t, SetDogNeuteredOutput{ID: 42, Neutered: true, Sex: domain.SexFemale}, out)
	})

	t.Run("happy_path_false", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(7, "Toby", "Beagle", "ES-2", 36,
			domain.SexMale, 12.0, 1)
		loadedDog.SetNeutered(true) // start neutered so the toggle down is observable
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogNeuteredInput(7, false))
		assert.NoError(t, err)
		assert.False(t, out.Neutered)
		assert.Equal(t, domain.SexMale, out.Sex)
	})

	t.Run("update_error", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(1, "Luna", "Labrador", "ES-1", 24,
			domain.SexFemale, 22.5, 1)
		repoErr := errors.New("connection lost")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		uc := NewSetDogNeuteredUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogNeuteredInput(1, true))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
