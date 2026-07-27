package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestListByPassReservationsUseCase_Success(t *testing.T) {
	views := []*domain.ReservationView{makeOwnedView(1, 1)}
	repo := &mockReservationRepository{
		listByPassView: func(_ context.Context, passID, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 30, passID)
			return views, nil
		},
	}
	uc := NewListByPassReservationsUseCase(repo)
	output, err := uc.Execute(context.Background(), MustNewListByPassReservationsInput(30, 50, 0))
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestNewListByPassReservationsInput_ZeroPassID(t *testing.T) {
	_, err := NewListByPassReservationsInput(0, 50, 0)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "pass_id", verr.Field)
}
