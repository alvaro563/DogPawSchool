package reservation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func validListByUserInput() ListByUserReservationsInput {
	return ListByUserReservationsInput{UserID: 1, Limit: 50, Offset: 0}
}

func TestListByUserReservationsUseCase_Success(t *testing.T) {
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
	_, err := uc.Execute(context.Background(), ListByUserReservationsInput{
		UserID: 1, Status: &status, From: &from, To: &to, Limit: 50, Offset: 0,
	})
	require.NoError(t, err)
}

func TestListByUserReservationsUseCase_ValidationErrors(t *testing.T) {
	base := validListByUserInput()
	tests := []struct {
		name      string
		mutate    func(*ListByUserReservationsInput)
		wantField string
	}{
		{
			name:      "zero_user_id",
			mutate:    func(i *ListByUserReservationsInput) { i.UserID = 0 },
			wantField: "user_id",
		},
		{
			name:      "invalid_status",
			mutate:    func(i *ListByUserReservationsInput) { bad := domain.ReservationStatus("BOGUS"); i.Status = &bad },
			wantField: "status",
		},
		{
			name: "from_after_to",
			mutate: func(i *ListByUserReservationsInput) {
				from := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
				to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				i.From = &from
				i.To = &to
			},
			wantField: "from",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)
			repo := &mockReservationRepository{
				listByUserView: func(context.Context, int, *domain.ReservationStatus, *time.Time, *time.Time, int, int) ([]*domain.ReservationView, error) {
					t.Fatal("ListByUserView should not be called on validation error")
					return nil, nil
				},
			}
			uc := NewListByUserReservationsUseCase(repo)
			_, err := uc.Execute(context.Background(), input)
			assertValidationError(t, err, tt.wantField)
		})
	}
}
