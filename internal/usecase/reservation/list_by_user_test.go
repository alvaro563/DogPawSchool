package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func validListByUserInput() ListByUserReservationsInput {
	return MustNewListByUserReservationsInput(1, nil, nil, nil, 50, 0)
}

func TestListByUserReservationsUseCase_Success(t *testing.T) {
	t.Parallel()
	views := []*domain.ReservationView{
		makeOwnedView(1, 1),
		makeOwnedView(2, 1),
	}
	repo := &mockReservationRepository{
		listByUserView: func(_ context.Context, userID int, status *domain.ReservationStatus, from, to *time.Time, limit, offset int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 1, userID)
			assert.Nil(t, status, "no status filter")
			assert.Nil(t, from, "no from filter")
			assert.Nil(t, to, "no to filter")
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return views, nil
		},
	}
	uc := NewListByUserReservationsUseCase(repo)
	output, err := uc.Execute(context.Background(), validListByUserInput())
	require.NoError(t, err)
	assert.Len(t, output.Views, 2)
}

func TestListByUserReservationsUseCase_WithFilters(t *testing.T) {
	t.Parallel()
	status := domain.StatusConfirmed
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	repo := &mockReservationRepository{
		listByUserView: func(_ context.Context, userID int, st *domain.ReservationStatus, f, toPtr *time.Time, _, _ int) ([]*domain.ReservationView, error) {
			assert.Equal(t, 1, userID)
			require.NotNil(t, st)
			assert.Equal(t, domain.StatusConfirmed, *st)
			require.NotNil(t, f)
			assert.Equal(t, from.Unix(), f.Unix())
			require.NotNil(t, toPtr)
			assert.Equal(t, to.Unix(), toPtr.Unix())
			return nil, nil
		},
	}
	uc := NewListByUserReservationsUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewListByUserReservationsInput(1, &status, &from, &to, 50, 0))
	require.NoError(t, err)
}

func TestNewListByUserReservationsInput(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bad := domain.ReservationStatus("BOGUS")

	scenarios := []struct {
		name   string
		userID int
		status *domain.ReservationStatus
		from   *time.Time
		to     *time.Time
		field  string
	}{
		{"zero_user_id", 0, nil, nil, nil, "user_id"},
		{"invalid_status", 1, &bad, nil, nil, "status"},
		{"from_after_to", 1, nil, &from, &to, "from"},
	}
	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewListByUserReservationsInput(tt.userID, tt.status, tt.from, tt.to, 50, 0)
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.field, verr.Field)
		})
	}
}
