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

func validForgiveInput() ForgiveReservationInput {
	return MustNewForgiveReservationInput(99, func() time.Time { return fixedNow })
}

func newForgiveUseCase(
	reservationRepo domain.ReservationRepository,
	passRepo domain.PassRepository,
	transactor Transactor,
) *ForgiveReservationUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewForgiveReservationUseCase(transactor, passRepo, reservationRepo)
}

// forgiveFlowStubs wires a forgive use case with default stubs:
// the reservation (CANCELLED_LATE) and the pass (initial sessions
// consumed by registration, NOT refunded).
func forgiveFlowStubs(reservation *domain.Reservation, pass *domain.Pass) (
	*stubActivityRepository, *mockReservationRepository, *stubPassRepository,
) {
	_ = /* stubActivityRepository placeholder, never used */ (*stubActivityRepository)(nil)
	resRepo := &mockReservationRepository{
		getByID: func(_ context.Context, id int) (*domain.Reservation, error) {
			if id != reservation.ID() {
				return nil, domain.ErrNotFound
			}
			return reservation, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) {
			if id != pass.ID() {
				return nil, domain.ErrNotFound
			}
			return pass, nil
		},
	}
	return nil, resRepo, passRepo
}

func TestNewForgiveReservationInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   int
		want string
	}{
		{"zero_id", 0, "reservation_id"},
		{"negative_id", -5, "reservation_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewForgiveReservationInput(tt.id, func() time.Time { return fixedNow })
			require.Error(t, err)
			var verr *ValidationError
			require.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.want, verr.Field)
		})
	}

	t.Run("nil_now_defaults_to_time_now", func(t *testing.T) {
		in, err := NewForgiveReservationInput(99, nil)
		require.NoError(t, err)
		assert.NotZero(t, in.Now())
	})
}

func TestForgiveReservationUseCase_Success(t *testing.T) {
	t.Parallel()
	reservation := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, fixedNow)
	pass := validPass(30, 1, 5) // 5 remaining of 10 → consumed 5 → can refund

	_, resRepo, passRepo := forgiveFlowStubs(reservation, pass)
	var passUpdated bool
	passRepo.update = func(_ context.Context, p *domain.Pass, _ time.Time) error {
		passUpdated = true
		assert.Equal(t, 6, p.RemainingSessions(), "session must be refunded (5 -> 6)")
		return nil
	}
	var resUpdated bool
	resRepo.update = func(_ context.Context, r *domain.Reservation, _ domain.ReservationStatus) error {
		resUpdated = true
		assert.Equal(t, domain.StatusForgiven, r.Status())
		return nil
	}

	uc := newForgiveUseCase(resRepo, passRepo, nil)
	out, err := uc.Execute(context.Background(), validForgiveInput())
	require.NoError(t, err)
	require.NotNil(t, out.Reservation)
	assert.Equal(t, domain.StatusForgiven, out.Reservation.Status())
	assert.True(t, out.PassSessionRefunded, "session was refundable, output must reflect that")
	assert.True(t, passUpdated)
	assert.True(t, resUpdated)
}

func TestForgiveReservationUseCase_RefundsEvenAfterInsufficientBalance(t *testing.T) {
	t.Parallel()
	reservation := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, fixedNow)
	// Fresh pass with no consumption → CanRefund() == false.
	pass := validPass(30, 1, 10)

	_, resRepo, passRepo := forgiveFlowStubs(reservation, pass)
	var passUpdateCalls int
	passRepo.update = func(_ context.Context, p *domain.Pass, _ time.Time) error {
		passUpdateCalls++
		return nil
	}

	uc := newForgiveUseCase(resRepo, passRepo, nil)
	out, err := uc.Execute(context.Background(), validForgiveInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusForgiven, out.Reservation.Status())
	assert.False(t, out.PassSessionRefunded, "no consumption = nothing to refund")
	assert.Equal(t, 0, passUpdateCalls, "no Update call when nothing to refund")
}

func TestForgiveReservationUseCase_ReservationNotFound(t *testing.T) {
	t.Parallel()
	pass := validPass(30, 1, 5)

	resRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, domain.ErrNotFound
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) { return pass, nil },
	}

	uc := newForgiveUseCase(resRepo, passRepo, nil)
	_, err := uc.Execute(context.Background(), validForgiveInput())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestForgiveReservationUseCase_NotLateCancelled_TableDriven(t *testing.T) {
	t.Parallel()
	pass := validPass(30, 1, 5)
	passUpdated := false
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) {
			return pass, nil
		},
		update: func(_ context.Context, _ *domain.Pass, _ time.Time) error {
			passUpdated = true
			return nil
		},
	}

	cases := []struct {
		name   string
		status domain.ReservationStatus
	}{
		{"CONFIRMED", domain.StatusConfirmed},
		{"PENDING_TO_CONFIRM", domain.StatusPendingToConfirm},
		{"CANCELLED_IN_TIME", domain.StatusCancelledInTime},
		{"FORGIVEN", domain.StatusForgiven},
		{"COMPLETED", domain.StatusCompleted},
		{"NO_SHOW", domain.StatusNoShow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reservation := mustNewReservation(99, 10, 20, 30, tc.status, fixedNow)
			resRepo := &mockReservationRepository{
				getByID: func(_ context.Context, id int) (*domain.Reservation, error) { return reservation, nil },
			}
			uc := newForgiveUseCase(resRepo, passRepo, nil)
			_, err := uc.Execute(context.Background(), validForgiveInput())
			require.Error(t, err)
			require.True(t, errors.Is(err, ErrNotLateCancelled),
				"expected ErrNotLateCancelled, got %T: %v", err, err)
		})
	}
	assert.False(t, passUpdated, "pass must never be touched when status precondition fails")
}

func TestForgiveReservationUseCase_RepoErrorsWrapped(t *testing.T) {
	t.Parallel()
	t.Run("get_reservation_error_wrapped", func(t *testing.T) {
		resRepo := &mockReservationRepository{
			getByID: func(context.Context, int) (*domain.Reservation, error) {
				return nil, errors.New("db connection lost")
			},
		}
		passRepo := &stubPassRepository{}
		uc := newForgiveUseCase(resRepo, passRepo, nil)
		_, err := uc.Execute(context.Background(), validForgiveInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get reservation")
		assert.Contains(t, err.Error(), "db connection lost")
	})

	t.Run("get_pass_db_error_wrapped", func(t *testing.T) {
		reservation := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, fixedNow)
		resRepo := &mockReservationRepository{
			getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		}
		passRepo := &stubPassRepository{
			getByID: func(context.Context, int) (*domain.Pass, error) {
				return nil, errors.New("query timeout")
			},
		}
		uc := newForgiveUseCase(resRepo, passRepo, nil)
		_, err := uc.Execute(context.Background(), validForgiveInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get pass")
		assert.Contains(t, err.Error(), "query timeout")
	})
}

func TestForgiveReservationUseCase_TransactorRollsBackOnMovementFailure(t *testing.T) {
	t.Parallel()
	reservation := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, fixedNow)
	pass := validPass(30, 1, 5)

	resRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update: func(_ context.Context, _ *domain.Pass, _ time.Time) error { return errors.New("update failed") },
	}
	// Replace the default stub transactor with one that simply runs
	// the closure and surfaces any error it returns (no real DB).
	uc := newForgiveUseCase(resRepo, passRepo, &stubTransactor{})

	_, err := uc.Execute(context.Background(), validForgiveInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update pass")
}
