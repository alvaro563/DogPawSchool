package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestListByActivityReservationsUseCase_Success(t *testing.T) {
	t.Parallel()
	views := []*domain.ReservationView{makeOwnedView(1, 1)}
	repo := &mockReservationRepository{
		listByActivityView: func(_ context.Context, activityID, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 10, activityID)
			return views, nil
		},
	}
	uc := NewListByActivityReservationsUseCase(repo)
	output, err := uc.Execute(context.Background(), MustNewListByActivityReservationsInput(10, 50, 0))
	require.NoError(t, err)
	assert.Len(t, output.Views, 1)
}

func TestNewListByActivityReservationsInput_ZeroActivityID(t *testing.T) {
	t.Parallel()
	_, err := NewListByActivityReservationsInput(0, 50, 0)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "activity_id", verr.Field)
}
