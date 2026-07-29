package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewSetDogHeatInput(t *testing.T) {
	t.Parallel()
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewSetDogHeatInput(0, true)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
	t.Run("negative_id", func(t *testing.T) {
		_, err := NewSetDogHeatInput(-1, true)
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
	})
}

func TestSetDogHeatUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("not_found", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(9999, true))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("happy_path_female_heat_true", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
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
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(2, true))
		assert.NoError(t, err)
		assert.True(t, updateCalled, "repo Update must be called")
		assert.True(t, capturedDog.Heat(), "the persisted aggregate must carry the new flag")
		assert.Equal(t, SetDogHeatOutput{ID: 2, Heat: true, Sex: domain.SexFemale}, out)
	})

	t.Run("happy_path_female_heat_false", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(2, false))
		assert.NoError(t, err)
		assert.False(t, out.Heat)
	})

	t.Run("happy_path_male_heat_false", func(t *testing.T) {
		// Male + heat=false is fine (the invariant only blocks heat=true on males).
		loadedDog, _ := domain.NewDog(9, "Toby", "Cocker Spaniel", "ES-9", 18,
			domain.SexMale, 9.0, 1)
		var updateCalled bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(9, false))
		assert.NoError(t, err)
		assert.True(t, updateCalled)
		assert.Equal(t, domain.SexMale, out.Sex)
	})

	t.Run("rejects_heat_true_on_male", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(7, "Toby", "Beagle", "ES-7", 36,
			domain.SexMale, 12.0, 1)
		var updateCalled bool
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				updateCalled = true
				return nil
			},
		}
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(7, true))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidHeatForSex),
			"expected ErrInvalidHeatForSex, got %v", err)
		assert.False(t, updateCalled, "Update must NOT be called when the domain invariant fails")
	})

	t.Run("update_error", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(2, "Leia", "Samoyed", "ES-2", 18,
			domain.SexFemale, 12.0, 1)
		repoErr := errors.New("connection lost")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return repoErr
			},
		}
		uc := NewSetDogHeatUseCase(&stubTransactor{}, mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogHeatInput(2, true))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
