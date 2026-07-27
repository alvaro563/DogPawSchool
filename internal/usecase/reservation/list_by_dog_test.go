package reservation

import (
	"context"
	"errors"
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
	output, err := uc.Execute(context.Background(), MustNewListByDogReservationsInput(20, 50, 0))
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestNewListByDogReservationsInput_ZeroDogID(t *testing.T) {
	_, err := NewListByDogReservationsInput(0, 50, 0)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "dog_id", verr.Field)
}
