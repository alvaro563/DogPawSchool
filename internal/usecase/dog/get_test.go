package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestGetDogInput_Factory(t *testing.T) {
	in, err := NewGetDogInput(5)
	require.NoError(t, err)
	assert.Equal(t, 5, in.ID())
}

func TestGetDogInput_InvalidID(t *testing.T) {
	for _, id := range []int{0, -1} {
		_, err := NewGetDogInput(id)
		assert.Error(t, err)
		assert.True(t, IsValidationError(err))
	}
}

func TestGetDogUseCase_Success(t *testing.T) {
	dog := newTestDog(1)
	repo := &mockDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			assert.Equal(t, 1, id)
			return dog, nil
		},
	}
	uc := NewGetDogUseCase(repo)
	out, err := uc.Execute(context.Background(), MustNewGetDogInput(1))
	require.NoError(t, err)
	assert.Equal(t, dog, out.Dog)
}

func TestGetDogUseCase_NotFound(t *testing.T) {
	repo := &mockDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			return nil, nil
		},
	}
	uc := NewGetDogUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewGetDogInput(99))
	assert.Equal(t, ErrNotFound, err)
}

func TestGetDogUseCase_RepoError(t *testing.T) {
	repo := &mockDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			return nil, errors.New("db down")
		},
	}
	uc := NewGetDogUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewGetDogInput(1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func newTestDog(id int) *domain.Dog {
	d, _ := domain.NewDog(id, "Luna", "Labrador", "ES-"+string(rune('0'+id)), 24, domain.SexFemale, 22.5, 1)
	return d
}
