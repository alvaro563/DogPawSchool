package reservation

import (
	"context"
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
	output, err := uc.Execute(context.Background(), ListByPassReservationsInput{PassID: 30, Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestListByPassReservationsUseCase_ZeroPassID(t *testing.T) {
	uc := NewListByPassReservationsUseCase(&mockReservationRepository{
		listByPassView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			t.Fatal("ListByPassView should not be called on validation error")
			return nil, nil
		},
	})
	_, err := uc.Execute(context.Background(), ListByPassReservationsInput{PassID: 0, Limit: 50, Offset: 0})
	assertValidationError(t, err, "pass_id")
}
