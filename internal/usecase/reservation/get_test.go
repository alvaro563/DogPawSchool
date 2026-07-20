package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

func validGetInput() GetReservationInput {
	return GetReservationInput{UserID: 1, ReservationID: 99}
}

// makeOwnedView returns a ReservationView whose dog is owned by
// the given userID.
func makeOwnedView(id, userID int) *domain.ReservationView {
	return mustNewReservationView(
		id, 10, 20, userID, 30, userID,
		domain.StatusConfirmed, time.Now(),
		"Paseo", "Park", time.Now().Add(7*24*time.Hour),
		"Luna", 5,
	)
}

func TestGetReservationUseCase_Success(t *testing.T) {
	view := makeOwnedView(99, 1)
	repo := &mockReservationRepository{
		getView: func(_ context.Context, id int) (*domain.ReservationView, error) {
			assert.Equal(t, 99, id)
			return view, nil
		},
	}
	uc := NewGetReservationUseCase(repo)
	output, err := uc.Execute(context.Background(), validGetInput())
	require.NoError(t, err)
	assert.Same(t, view, output.View)
}

func TestGetReservationUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(input *GetReservationInput)
		wantField string
	}{
		{
			name:      "zero_user_id",
			mutate:    func(i *GetReservationInput) { i.UserID = 0 },
			wantField: "user_id",
		},
		{
			name:      "zero_reservation_id",
			mutate:    func(i *GetReservationInput) { i.ReservationID = 0 },
			wantField: "reservation_id",
		},
		{
			name:      "negative_reservation_id",
			mutate:    func(i *GetReservationInput) { i.ReservationID = -1 },
			wantField: "reservation_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validGetInput()
			tt.mutate(&input)
			repo := &mockReservationRepository{
				getView: func(context.Context, int) (*domain.ReservationView, error) {
					t.Fatal("GetView should not be called on validation error")
					return nil, nil
				},
			}
			uc := NewGetReservationUseCase(repo)
			_, err := uc.Execute(context.Background(), input)
			assertValidationError(t, err, tt.wantField)
		})
	}
}

func TestGetReservationUseCase_NotFound(t *testing.T) {
	repo := &mockReservationRepository{
		getView: func(context.Context, int) (*domain.ReservationView, error) {
			return nil, postgres.ErrReservationNotFound
		},
	}
	uc := NewGetReservationUseCase(repo)
	_, err := uc.Execute(context.Background(), validGetInput())
	assert.ErrorIs(t, err, ErrInvalidReservation)
}

func TestGetReservationUseCase_NotOwnedByUser(t *testing.T) {
	// The reservation exists but the dog is owned by user 99, not
	// user 1 (the path). Must surface as not_found (no leak).
	view := makeOwnedView(99, 99)
	repo := &mockReservationRepository{
		getView: func(context.Context, int) (*domain.ReservationView, error) {
			return view, nil
		},
	}
	uc := NewGetReservationUseCase(repo)
	_, err := uc.Execute(context.Background(), validGetInput())
	assert.ErrorIs(t, err, ErrReservationNotOwned)
}

func TestGetReservationUseCase_RepoErrorIsWrapped(t *testing.T) {
	repo := &mockReservationRepository{
		getView: func(context.Context, int) (*domain.ReservationView, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := NewGetReservationUseCase(repo)
	_, err := uc.Execute(context.Background(), validGetInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get reservation view")
	assert.Contains(t, err.Error(), "db connection lost")
}
