package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteDogUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		t.Run("zero_id", func(t *testing.T) {
			uc := NewDeleteDogUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), DeleteDogInput{ID: 0})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr), "expected *ValidationError, got %T", err)
			assert.Equal(t, "id", verr.Field)
		})
		t.Run("negative_id", func(t *testing.T) {
			uc := NewDeleteDogUseCase(&mockDogRepository{})
			_, err := uc.Execute(context.Background(), DeleteDogInput{ID: -5})
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "id", verr.Field)
		})
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedID int
		mock := &mockDogRepository{
			delete: func(ctx context.Context, id int) error {
				capturedID = id
				return nil
			},
		}
		uc := NewDeleteDogUseCase(mock)
		out, err := uc.Execute(context.Background(), DeleteDogInput{ID: 42})
		assert.NoError(t, err)
		assert.Equal(t, DeleteDogOutput{}, out)
		assert.Equal(t, 42, capturedID)
	})

	t.Run("not_found_passes_through", func(t *testing.T) {
		repoErr := errors.New("postgres: dog not found")
		mock := &mockDogRepository{
			delete: func(ctx context.Context, id int) error {
				return repoErr
			},
		}
		uc := NewDeleteDogUseCase(mock)
		_, err := uc.Execute(context.Background(), DeleteDogInput{ID: 9999})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr), "expected wrapped repoErr")
	})

	t.Run("repo_error_is_wrapped", func(t *testing.T) {
		repoErr := errors.New("connection lost")
		mock := &mockDogRepository{
			delete: func(ctx context.Context, id int) error {
				return repoErr
			},
		}
		uc := NewDeleteDogUseCase(mock)
		_, err := uc.Execute(context.Background(), DeleteDogInput{ID: 1})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})
}
