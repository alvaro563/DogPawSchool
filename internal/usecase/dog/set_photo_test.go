package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewSetDogPhotoInput(t *testing.T) {
	t.Parallel()
	t.Run("zero_id", func(t *testing.T) {
		_, err := NewSetDogPhotoInput(0, "")
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
	t.Run("negative_id", func(t *testing.T) {
		_, err := NewSetDogPhotoInput(-5, "https://example.com/dog.jpg")
		assert.Error(t, err)
		var verr *ValidationError
		assert.True(t, errors.As(err, &verr))
		assert.Equal(t, "id", verr.Field)
	})
}

func TestSetDogPhotoUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("not_found", func(t *testing.T) {
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewSetDogPhotoUseCase(&stubTransactor{}, mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogPhotoInput(9999, "https://example.com/dog.jpg"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("set_photo_url", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(42, "Luna", "Labrador", "ES-1", 24,
			domain.SexFemale, 22.5, 1)
		var capturedDog *domain.Dog
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				capturedDog = dog
				return nil
			},
		}
		uc := NewSetDogPhotoUseCase(&stubTransactor{}, mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogPhotoInput(42, "https://example.com/luna.jpg"))
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/luna.jpg", out.PhotoURL)
		assert.Equal(t, "https://example.com/luna.jpg", capturedDog.PhotoURL())
		assert.Equal(t, 42, out.ID)
	})

	t.Run("clear_photo_url", func(t *testing.T) {
		loadedDog, _ := domain.NewDog(7, "Toby", "Beagle", "ES-2", 36,
			domain.SexMale, 12.0, 1)
		loadedDog.SetPhotoURL("https://old.com/toby.jpg")
		mock := &mockDogRepository{
			getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
				return loadedDog, nil
			},
			update: func(ctx context.Context, dog *domain.Dog) error {
				return nil
			},
		}
		uc := NewSetDogPhotoUseCase(&stubTransactor{}, mock)
		out, err := uc.Execute(context.Background(), MustNewSetDogPhotoInput(7, ""))
		assert.NoError(t, err)
		assert.Equal(t, "", out.PhotoURL)
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
		uc := NewSetDogPhotoUseCase(&stubTransactor{}, mock)
		_, err := uc.Execute(context.Background(), MustNewSetDogPhotoInput(1, "https://example.com/dog.jpg"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
