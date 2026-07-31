package reservation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func rejectClock() func() time.Time { return func() time.Time { return fixedNow } }

func TestRejectPendingReservationUseCase_SuccessRefundsPass(t *testing.T) {
	t.Parallel()
	pending := mustNewReservation(42, 10, 20, 30, domain.StatusPendingToConfirm, fixedNow)
	pass := validPass(30, 1, 4) // 10 initial, 6 consumed
	var passUpdated bool
	var reservationUpdated bool
	reservationRepo := &mockReservationRepository{
		getByID: func(_ context.Context, id int) (*domain.Reservation, error) {
			assert.Equal(t, 42, id)
			return pending, nil
		},
		update: func(_ context.Context, r *domain.Reservation) error {
			reservationUpdated = true
			assert.Equal(t, domain.StatusCancelledInTime, r.Status())
			return nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) {
			assert.Equal(t, 30, id)
			return pass, nil
		},
		update: func(_ context.Context, p *domain.Pass) error {
			passUpdated = true
			assert.Equal(t, 5, p.RemainingSessions(), "rejecting refunds the consumed session")
			return nil
		},
	}
	uc := NewRejectPendingReservationUseCase(&stubTransactor{}, passRepo, reservationRepo)
	output, err := uc.Execute(context.Background(), MustNewRejectPendingReservationInput(42, rejectClock()))
	require.NoError(t, err)
	assert.True(t, reservationUpdated)
	assert.True(t, passUpdated, "reject follows the cancel in-time policy: session refunded")
	assert.Equal(t, domain.StatusCancelledInTime, output.Reservation.Status())
}

func TestRejectPendingReservationUseCase_NotFound(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := NewRejectPendingReservationUseCase(&stubTransactor{}, &stubPassRepository{}, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewRejectPendingReservationInput(42, rejectClock()))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRejectPendingReservationUseCase_NotPending(t *testing.T) {
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
	uc := NewRejectPendingReservationUseCase(&stubTransactor{}, &stubPassRepository{}, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewRejectPendingReservationInput(42, rejectClock()))
	assert.ErrorIs(t, err, ErrNotPending)
}

func TestRejectPendingReservationUseCase_Validation(t *testing.T) {
	t.Parallel()
	_, err := NewRejectPendingReservationInput(0, nil)
	assertValidationError(t, err, "reservation_id")
}

func TestRejectPendingReservationUseCase_MissingPassStillRejects(t *testing.T) {
	t.Parallel()
	pending := mustNewReservation(42, 10, 20, 30, domain.StatusPendingToConfirm, fixedNow)
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return pending, nil
		},
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCancelledInTime, r.Status())
			return nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := NewRejectPendingReservationUseCase(&stubTransactor{}, passRepo, reservationRepo)
	_, err := uc.Execute(context.Background(), MustNewRejectPendingReservationInput(42, rejectClock()))
	assert.NoError(t, err, "reject should still succeed when the pass row is gone")
}
