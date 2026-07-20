package reservation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestListByActivityReservationsUseCase_Success(t *testing.T) {
	views := []*domain.ReservationView{makeOwnedView(1, 1)}
	repo := &mockReservationRepository{
		listByActivityView: func(_ context.Context, activityID, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 10, activityID)
			return views, nil
		},
	}
	uc := NewListByActivityReservationsUseCase(repo)
	output, err := uc.Execute(context.Background(), ListByActivityReservationsInput{ActivityID: 10, Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestListByActivityReservationsUseCase_ZeroActivityID(t *testing.T) {
	uc := NewListByActivityReservationsUseCase(&mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			t.Fatal("ListByActivityView should not be called on validation error")
			return nil, nil
		},
	})
	_, err := uc.Execute(context.Background(), ListByActivityReservationsInput{ActivityID: 0, Limit: 50, Offset: 0})
	assertValidationError(t, err, "activity_id")
}
