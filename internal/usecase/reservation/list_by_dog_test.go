package reservation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestListByDogReservationsUseCase_Success(t *testing.T) {
	views := []*domain.ReservationView{
		makeOwnedView(1, 1),
	}
	repo := &mockReservationRepository{
		listByDogView: func(_ context.Context, dogID, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 20, dogID)
			assert.Equal(t, 50, limit)
			return views, nil
		},
	}
	uc := NewListByDogReservationsUseCase(repo)
	output, err := uc.Execute(context.Background(), ListByDogReservationsInput{DogID: 20, Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestListByDogReservationsUseCase_ZeroDogID(t *testing.T) {
	uc := NewListByDogReservationsUseCase(&mockReservationRepository{
		listByDogView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			t.Fatal("ListByDogView should not be called on validation error")
			return nil, nil
		},
	})
	_, err := uc.Execute(context.Background(), ListByDogReservationsInput{DogID: 0, Limit: 50, Offset: 0})
	assertValidationError(t, err, "dog_id")
}
