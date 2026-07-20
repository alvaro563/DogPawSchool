package reservation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func validListUpcomingInput() ListUpcomingByUserInput {
	return ListUpcomingByUserInput{UserID: 1, Limit: 50, Offset: 0}
}

func TestListUpcomingByUserUseCase_Success(t *testing.T) {
	views := []*domain.ReservationView{
		makeOwnedView(1, 1),
		makeOwnedView(2, 1),
	}
	repo := &mockReservationRepository{
		listByUserUpcoming: func(_ context.Context, userID, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 1, userID)
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return views, nil
		},
	}
	uc := NewListUpcomingByUserUseCase(repo)
	output, err := uc.Execute(context.Background(), validListUpcomingInput())
	require.NoError(t, err)
	assert.Len(t, output.Views, 2)
}

func TestListUpcomingByUserUseCase_ZeroUserID(t *testing.T) {
	uc := NewListUpcomingByUserUseCase(&mockReservationRepository{
		listByUserUpcoming: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			t.Fatal("ListByUserUpcomingView should not be called on validation error")
			return nil, nil
		},
	})
	_, err := uc.Execute(context.Background(), ListUpcomingByUserInput{UserID: 0, Limit: 50, Offset: 0})
	assertValidationError(t, err, "user_id")
}
