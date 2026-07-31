package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestConfirmPendingReservationUseCase_Success(t *testing.T) {
	t.Parallel()
	pending := mustNewReservation(42, 10, 20, 30, domain.StatusPendingToConfirm, fixedNow)
	var updated bool
	reservationRepo := &mockReservationRepository{
		getByID: func(_ context.Context, id int) (*domain.Reservation, error) {
			assert.Equal(t, 42, id)
			return pending, nil
		},
		update: func(_ context.Context, r *domain.Reservation) error {
			updated = true
			assert.Equal(t, domain.StatusConfirmed, r.Status())
			return nil
		},
	}
	uc := NewConfirmPendingReservationUseCase(&stubTransactor{}, reservationRepo)
	output, err := uc.Execute(context.Background(), MustNewConfirmPendingReservationInput(42))
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, domain.StatusConfirmed, output.Reservation.Status())
}

func TestConfirmPendingReservationUseCase_NotFound(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := NewConfirmPendingReservationUseCase(&stubTransactor{}, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewConfirmPendingReservationInput(42))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestConfirmPendingReservationUseCase_NotPending(t *testing.T) {
	t.Parallel()
	confirmed := mustNewReservation(42, 10, 20, 30, domain.StatusConfirmed, fixedNow)
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmed, nil
		},
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("update must not be called for a non-pending reservation")
			return nil
		},
	}
	uc := NewConfirmPendingReservationUseCase(&stubTransactor{}, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewConfirmPendingReservationInput(42))
	assert.ErrorIs(t, err, ErrNotPending)
}

func TestConfirmPendingReservationUseCase_Validation(t *testing.T) {
	t.Parallel()
	_, err := NewConfirmPendingReservationInput(0)
	assertValidationError(t, err, "reservation_id")
}

func TestConfirmPendingReservationUseCase_RepoErrorWrapped(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, errors.New("db down")
		},
	}
	uc := NewConfirmPendingReservationUseCase(&stubTransactor{}, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewConfirmPendingReservationInput(42))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get reservation")
}
