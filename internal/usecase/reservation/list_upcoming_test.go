package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func validListUpcomingInput() ListUpcomingByUserInput {
	return MustNewListUpcomingByUserInput(1, 50, 0)
}

func TestListUpcomingByUserUseCase_Success(t *testing.T) {
	t.Parallel()
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

func TestNewListUpcomingByUserInput_ZeroUserID(t *testing.T) {
	t.Parallel()
	_, err := NewListUpcomingByUserInput(0, 50, 0)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "user_id", verr.Field)
}
