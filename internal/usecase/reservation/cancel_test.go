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

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func validCancelInput() CancelReservationInput {
	return MustNewCancelReservationInput(1, 10, func() time.Time { return fixedNow })
}

// validConfirmedReservation returns a confirmed reservation at the
// given id, pointing at the given activity/dog/pass. Tests use it
// to set up the GetByID stub for the reservation repo.
func validConfirmedReservation(id, activityID, dogID, passID int) *domain.Reservation {
	return mustNewReservation(id, activityID, dogID, passID, domain.StatusConfirmed, fixedNow)
}

// farFutureActivity returns an activity 7 days in the future, with
// room for at least one more booking. Anchored to fixedNow so the
// test is deterministic regardless of the wall clock.
func farFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(7*24*time.Hour))
}

// nearFutureActivity returns an activity 1 hour in the future. The
// cancellation late window is 2h, so this counts as a LATE cancel
// when the use case runs at fixedNow.
func nearFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(1*time.Hour))
}

// pastActivity returns an activity 24h in the past relative to
// fixedNow. Used to verify the activity-in-past guard.
func pastActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-24*time.Hour))
}

// newCancelUseCase wires the use case with default no-op mocks for
// every dependency. Tests override only the fields they care
// about.
func newCancelUseCase(
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
	transactor Transactor,
) *CancelReservationUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewCancelReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo)
}

func TestNewCancelReservationInput(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		name   string
		userID int
		resID  int
		field  string
	}{
		{"zero_user_id", 0, 10, "user_id"},
		{"negative_user_id", -1, 10, "user_id"},
		{"zero_reservation_id", 1, 0, "reservation_id"},
		{"negative_reservation_id", 1, -5, "reservation_id"},
	}
	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCancelReservationInput(tt.userID, tt.resID, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.field, verr.Field)
		})
	}
}

func TestCancelReservationUseCase_SuccessInTime(t *testing.T) {
	t.Parallel()
	userID := 1
	activity := farFutureActivity(10)
	dog := validDog(20, userID)
	pass := validPass(30, userID, 1)
	originalPassRemaining := pass.RemainingSessions()
	originalMovementCount := len(pass.PendingMovements())

	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(_ context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 10, id)
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) { return pass, nil },
		update: func(_ context.Context, p *domain.Pass) error {
			assert.Equal(t, originalPassRemaining+1, p.RemainingSessions(),
				"pass should be refunded (remaining + 1)")
			return nil
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(_ context.Context, id int) (*domain.Reservation, error) { return reservation, nil },
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCancelledInTime, r.Status(),
				"reservation should be CANCELLED_IN_TIME")
			return nil
		},
	}

	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validCancelInput())

	require.NoError(t, err)
	require.NotNil(t, output.Reservation)
	assert.Equal(t, domain.StatusCancelledInTime, output.Reservation.Status(),
		"output should reflect the in-time cancel")
	assert.Equal(t, originalPassRemaining+1, pass.RemainingSessions(),
		"in-memory pass should reflect the refund")
	assert.Equal(t, originalMovementCount+1, len(pass.PendingMovements()),
		"in-memory pass should have a new movement")
}

func TestCancelReservationUseCase_SuccessLateDoesNotRefund(t *testing.T) {
	t.Parallel()
	// Activity is 1h in the future. The cancellation late window
	// is 2h, so the use case classifies this as a LATE cancel.
	// The pass must NOT be refunded.
	userID := 1
	activity := nearFutureActivity(10)
	dog := validDog(20, userID)
	pass := validPass(30, userID, 1)
	originalPassRemaining := pass.RemainingSessions()
	originalMovementCount := len(pass.PendingMovements())

	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(_ context.Context, _ int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, _ int) (*domain.Pass, error) { return pass, nil },
		update: func(_ context.Context, _ *domain.Pass) error {
			t.Fatal("pass Update should not be called for a late cancel")
			return nil
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(_ context.Context, _ int) (*domain.Reservation, error) { return reservation, nil },
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCancelledLate, r.Status())
			return nil
		},
	}

	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validCancelInput())

	require.NoError(t, err)
	require.NotNil(t, output.Reservation)
	assert.Equal(t, domain.StatusCancelledLate, output.Reservation.Status())
	assert.Equal(t, originalPassRemaining, pass.RemainingSessions(),
		"late cancel must NOT change pass remaining")
	assert.Equal(t, originalMovementCount, len(pass.PendingMovements()),
		"late cancel must NOT add a new movement")
}

func TestCancelReservationUseCase_ReservationNotFound(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newCancelUseCase(nil, nil, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidReservation)
}

func TestCancelReservationUseCase_AlreadyCancelled(t *testing.T) {
	t.Parallel()
	cancelledInTime := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledInTime, fixedNow)
	cancelledLate := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, fixedNow)
	completed := mustNewReservation(99, 10, 20, 30, domain.StatusCompleted, fixedNow)
	forgiven := mustNewReservation(99, 10, 20, 30, domain.StatusForgiven, fixedNow)
	noShow := mustNewReservation(99, 10, 20, 30, domain.StatusNoShow, fixedNow)

	cases := []struct {
		name        string
		reservation *domain.Reservation
	}{
		{"cancelled_in_time", cancelledInTime},
		{"cancelled_late", cancelledLate},
		{"completed", completed},
		{"forgiven", forgiven},
		{"no_show", noShow},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.reservation
			reservationRepo := &mockReservationRepository{
				getByID: func(context.Context, int) (*domain.Reservation, error) { return r, nil },
			}
			uc := newCancelUseCase(nil, nil, nil, reservationRepo, nil)
			_, err := uc.Execute(context.Background(), validCancelInput())
			assert.ErrorIs(t, err, ErrAlreadyCancelled)
		})
	}
}

func TestCancelReservationUseCase_ActivityInPast(t *testing.T) {
	t.Parallel()
	activity := pastActivity(10)
	dog := validDog(20, 1)
	pass := validPass(30, 1, 1)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("reservation Update should not be called when activity is in the past")
			return nil
		},
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrActivityInPast)
}

func TestCancelReservationUseCase_DogNotFound(t *testing.T) {
	t.Parallel()
	activity := farFutureActivity(10)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return nil, domain.ErrNotFound
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
	}
	uc := newCancelUseCase(activityRepo, dogRepo, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidDog)
}

func TestCancelReservationUseCase_DogNotOwnedByUser(t *testing.T) {
	t.Parallel()
	activity := farFutureActivity(10)
	// Dog belongs to user 99, but the request is for user 1.
	dog := validDog(20, 99)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
	}
	uc := newCancelUseCase(activityRepo, dogRepo, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidDog, "should not leak dog ownership")
}

func TestCancelReservationUseCase_PassNotFound(t *testing.T) {
	t.Parallel()
	activity := farFutureActivity(10)
	dog := validDog(20, 1)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return nil, domain.ErrNotFound
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidPass)
}

func TestCancelReservationUseCase_PassNotOwnedByUser(t *testing.T) {
	t.Parallel()
	activity := farFutureActivity(10)
	dog := validDog(20, 1)
	// Pass belongs to user 99, but the request is for user 1.
	pass := validPass(30, 99, 1)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidPass, "should not leak pass ownership")
}

func TestCancelReservationUseCase_TransactorRollsBackOnMovementFailure(t *testing.T) {
	t.Parallel()
	userID := 1
	activity := farFutureActivity(10)
	dog := validDog(20, userID)
	pass := validPass(30, userID, 1)

	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	// PassRepository.Update now persists the counter and the audit
	// movement together, so a failed audit insert surfaces as a failed
	// Update. The reservation must not be touched afterwards.
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update: func(context.Context, *domain.Pass) error {
			return errors.New("add pass movement: movement insert failed")
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("reservation Update should not be called after the pass Update fails")
			return nil
		},
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "movement insert failed")
}

func TestCancelReservationUseCase_ReservationRepoErrorIsWrapped(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := newCancelUseCase(nil, nil, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get reservation")
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestCancelReservationUseCase_InTimeButPassNotRefundable(t *testing.T) {
	t.Parallel()
	// The activity is far in the future (in-time cancel), but
	// the pass is fresh (remaining == num) so CanRefund() returns
	// false. The use case must NOT call RefundSession and must
	// NOT call Update/AddMovement on the pass. The reservation
	// is still cancelled in time, just without a refund.
	userID := 1
	activity := farFutureActivity(10)
	dog := validDog(20, userID)
	now := fixedNow
	pass := domain.MustNewPass(30, 5, 5, 5, domain.PassGeneric, userID, now, now, nil)
	require.Equal(t, 5, pass.RemainingSessions())
	require.False(t, pass.CanRefund(), "fresh pass should not be refundable")

	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update: func(context.Context, *domain.Pass) error {
			t.Fatal("pass Update should not be called when CanRefund() is false")
			return nil
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCancelledInTime, r.Status())
			return nil
		},
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validCancelInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelledInTime, output.Reservation.Status())
	assert.Equal(t, 5, pass.RemainingSessions())
	assert.Empty(t, pass.PendingMovements())
}
