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

// validCancelInput returns a known-good input. Tests mutate one
// field at a time to cover the negative cases.
func validCancelInput() CancelReservationInput {
	return CancelReservationInput{
		UserID:        1,
		ReservationID: 10,
	}
}

// validConfirmedReservation returns a confirmed reservation at the
// given id, pointing at the given activity/dog/pass. Tests use it
// to set up the GetByID stub for the reservation repo.
func validConfirmedReservation(id, activityID, dogID, passID int) *domain.Reservation {
	return mustNewReservation(id, activityID, dogID, passID, domain.StatusConfirmed, time.Now())
}

// farFutureActivity returns an activity 7 days in the future, with
// room for at least one more booking.
func farFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, time.Now().Add(7*24*time.Hour))
}

// nearFutureActivity returns an activity 1 hour in the future. The
// cancellation late window is 2h, so this counts as a LATE cancel
// when the use case runs "now".
func nearFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, time.Now().Add(1*time.Hour))
}

// pastActivity returns an activity 24h in the past. Used to verify
// the activity-in-past guard.
func pastActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, time.Now().Add(-24*time.Hour))
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

func TestCancelReservationUseCase_SuccessInTime(t *testing.T) {
	userID := 1
	activity := farFutureActivity(10)
	dog := validDog(20, userID)
	// Start with a pass whose remainingSessions is 1 less than
	// numOfSessions (so CanRefund() returns true).
	pass := validPass(30, userID, 1)
	originalPassRemaining := pass.RemainingSessions()
	originalMovementCount := len(pass.Movements())

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
		addMovement: func(_ context.Context, m *domain.PassMovement) error {
			assert.Equal(t, 1, m.Amount(), "movement amount should be +1")
			assert.Contains(t, m.Reason(), "cancelled in time")
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
	assert.Equal(t, originalMovementCount+1, len(pass.Movements()),
		"in-memory pass should have a new movement")
}

func TestCancelReservationUseCase_SuccessLateDoesNotRefund(t *testing.T) {
	// Activity is 1h in the future. The cancellation late window
	// is 2h, so the use case classifies this as a LATE cancel.
	// The pass must NOT be refunded.
	userID := 1
	activity := nearFutureActivity(10)
	dog := validDog(20, userID)
	pass := validPass(30, userID, 1)
	originalPassRemaining := pass.RemainingSessions()
	originalMovementCount := len(pass.Movements())

	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(_ context.Context, _ int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, _ int) (*domain.Pass, error) { return pass, nil },
		// No Update or AddMovement should be called for a late cancel.
		update: func(_ context.Context, _ *domain.Pass) error {
			t.Fatal("pass Update should not be called for a late cancel")
			return nil
		},
		addMovement: func(_ context.Context, _ *domain.PassMovement) error {
			t.Fatal("pass AddMovement should not be called for a late cancel")
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
	assert.Equal(t, domain.StatusCancelledLate, output.Reservation.Status())
	assert.Equal(t, originalPassRemaining, pass.RemainingSessions(),
		"late cancel must NOT change pass remaining")
	assert.Equal(t, originalMovementCount, len(pass.Movements()),
		"late cancel must NOT add a new movement")
}

func TestCancelReservationUseCase_ValidationErrors(t *testing.T) {
	base := validCancelInput()
	tests := []struct {
		name      string
		mutate    func(input *CancelReservationInput)
		wantField string
	}{
		{
			name:      "zero_user_id",
			mutate:    func(i *CancelReservationInput) { i.UserID = 0 },
			wantField: "user_id",
		},
		{
			name:      "negative_user_id",
			mutate:    func(i *CancelReservationInput) { i.UserID = -1 },
			wantField: "user_id",
		},
		{
			name:      "zero_reservation_id",
			mutate:    func(i *CancelReservationInput) { i.ReservationID = 0 },
			wantField: "reservation_id",
		},
		{
			name:      "negative_reservation_id",
			mutate:    func(i *CancelReservationInput) { i.ReservationID = -5 },
			wantField: "reservation_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)
			// All repos should not be called on validation error.
			reservationRepo := &mockReservationRepository{
				getByID: func(context.Context, int) (*domain.Reservation, error) {
					t.Fatal("reservationRepo.GetByID should not be called")
					return nil, nil
				},
			}
			uc := newCancelUseCase(nil, nil, nil, reservationRepo, nil)
			_, err := uc.Execute(context.Background(), input)
			assertValidationError(t, err, tt.wantField)
		})
	}
}

func TestCancelReservationUseCase_ReservationNotFound(t *testing.T) {
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, postgres.ErrReservationNotFound
		},
	}
	uc := newCancelUseCase(nil, nil, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	assert.ErrorIs(t, err, ErrInvalidReservation)
}

func TestCancelReservationUseCase_AlreadyCancelled(t *testing.T) {
	cancelledInTime := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledInTime, time.Now())
	cancelledLate := mustNewReservation(99, 10, 20, 30, domain.StatusCancelledLate, time.Now())
	completed := mustNewReservation(99, 10, 20, 30, domain.StatusCompleted, time.Now())
	forgiven := mustNewReservation(99, 10, 20, 30, domain.StatusForgiven, time.Now())
	noShow := mustNewReservation(99, 10, 20, 30, domain.StatusNoShow, time.Now())

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
			// Re-capture for closure safety.
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
	activity := farFutureActivity(10)
	reservation := validConfirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return nil, postgres.ErrNotFound
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
			return nil, postgres.ErrPassNotFound
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
	// The pass AddMovement step fails. The transactor must roll
	// back so the reservation status is NOT persisted as
	// CANCELLED_IN_TIME. Use case returns the underlying error
	// wrapped.
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
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update:  func(context.Context, *domain.Pass) error { return nil },
		addMovement: func(context.Context, *domain.PassMovement) error {
			return errors.New("movement insert failed")
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		// Update should not be called because the tx rolls back
		// before the reservation is updated.
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("reservation Update should not be called after AddMovement fails")
			return nil
		},
	}
	uc := newCancelUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCancelInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add movement")
}

func TestCancelReservationUseCase_ReservationRepoErrorIsWrapped(t *testing.T) {
	// Non-sentinel errors are wrapped (no mapping), so the
	// handler surfaces them as 500.
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
	// The activity is far in the future (in-time cancel), but
	// the pass is fully available (remaining == num) so
	// CanRefund() returns false. The use case must NOT call
	// RefundSession (which would refuse) and must NOT call
	// Update or AddMovement on the pass. The reservation is
	// still cancelled in time, just without a refund.
	userID := 1
	activity := farFutureActivity(10)
	dog := validDog(20, userID)
	// validPass(30, 1, 5) starts with 10, consumes 5 → remaining = 5 == num.
	// Hmm, that's still refundable. To get a non-refundable
	// pass, we need remaining == num, which means never consumed.
	// Easiest: build with a custom constructor.
	now := time.Now()
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
		addMovement: func(context.Context, *domain.PassMovement) error {
			t.Fatal("pass AddMovement should not be called when CanRefund() is false")
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
	// In-memory pass state is unchanged (no refund applied).
	assert.Equal(t, 5, pass.RemainingSessions())
	assert.Empty(t, pass.Movements())
}
