package reservation

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterReservationInput {
	return MustNewRegisterReservationInput(1, 10, 20, 30, func() time.Time { return fixedNow })
}

// validFutureActivity returns an activity in the future, with room
// for at least one more booking. Anchored to fixedNow so the test
// is deterministic regardless of the wall clock.
func validFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(7*24*time.Hour))
}

// validDog returns a dog owned by the given user.
func validDog(id, userID int) *domain.Dog {
	dog, err := domain.NewDog(id, "Luna", "Labrador", "ES-"+strconv.Itoa(id), 24, domain.SexFemale, 22.5, userID)
	if err != nil {
		panic(err)
	}
	return dog
}

// validPass returns a pass owned by the given user with the given
// remaining sessions, no expiry. Anchored to fixedNow.
func validPass(id, userID, remaining int) *domain.Pass {
	now := fixedNow
	const initialSessions = 10
	pass := domain.MustNewPass(id, initialSessions, initialSessions, 1000, domain.PassGeneric, userID, now, now, nil)
	for i := 0; i < initialSessions-remaining; i++ {
		_, _ = pass.ConsumeSession("seed", now)
	}
	return pass
}

// newRegisterUseCase wires the use case with default no-op mocks for
// every dependency. Tests override only the fields they care about.
func newRegisterUseCase(
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
	transactor Transactor,
) *RegisterReservationUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewRegisterReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo, func() time.Time { return fixedNow })
}

func TestNewRegisterReservationInput(t *testing.T) {
	tests := []struct {
		name   string
		userID int
		actID  int
		dogID  int
		passID int
		field  string
	}{
		{"zero_user_id", 0, 10, 20, 30, "user_id"},
		{"negative_user_id", -1, 10, 20, 30, "user_id"},
		{"zero_activity_id", 1, 0, 20, 30, "activity_id"},
		{"zero_dog_id", 1, 10, 0, 30, "dog_id"},
		{"zero_pass_id", 1, 10, 20, 0, "pass_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRegisterReservationInput(tt.userID, tt.actID, tt.dogID, tt.passID, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.field, verr.Field)
		})
	}
}

func TestRegisterReservationUseCase_Success(t *testing.T) {
	now := fixedNow
	userID := 1
	activity := validFutureActivity(10)
	dog := validDog(20, userID)
	pass := validPass(30, userID, 5)

	var capturedReservation *domain.Reservation
	activityRepo := &stubActivityRepository{
		getByID: func(_ context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 10, id)
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			assert.Equal(t, 20, id)
			return dog, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(_ context.Context, id int) (*domain.Pass, error) {
			assert.Equal(t, 30, id)
			return pass, nil
		},
		update: func(_ context.Context, p *domain.Pass) error {
			assert.Equal(t, 4, p.RemainingSessions(), "pass should be decremented by 1")
			return nil
		},
		addMovement: func(_ context.Context, m *domain.PassMovement) error {
			assert.Equal(t, -1, m.Amount(), "movement amount should be -1")
			assert.Contains(t, m.Reason(), "activity 10")
			return nil
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(_ context.Context, id int) ([]*domain.Reservation, error) {
			assert.Equal(t, 10, id)
			return nil, nil
		},
		create: func(_ context.Context, r *domain.Reservation) (int, error) {
			capturedReservation = r
			assert.Equal(t, 10, r.ActivityID())
			assert.Equal(t, 20, r.DogID())
			assert.Equal(t, 30, r.PassID())
			assert.Equal(t, domain.StatusConfirmed, r.Status())
			assert.False(t, r.CreatedAt().IsZero())
			return 99, nil
		},
	}

	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validRegisterInput())

	require.NoError(t, err)
	assert.Equal(t, 99, output.ID)
	require.NotNil(t, capturedReservation)
	assert.Equal(t, 0, capturedReservation.ID(), "in-memory reservation has id=0 before DB insert")
	assert.Equal(t, 10, capturedReservation.ActivityID())
	assert.Equal(t, 4, pass.RemainingSessions())
	assert.True(t, now.Before(pass.Movements()[0].CreatedAt().Add(time.Second)),
		"movement createdAt should be ~now")
}

func TestRegisterReservationUseCase_ActivityNotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestRegisterReservationUseCase_ActivityInPast(t *testing.T) {
	pastActivity := domain.MustNewActivity(10, "Paseo", "Central", domain.TypeRoute, 5, 1,
		fixedNow.Add(-24*time.Hour))
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return pastActivity, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrActivityInPast)
}

func TestRegisterReservationUseCase_ActivityFull(t *testing.T) {
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	existing := []*domain.Reservation{
		mustNewReservation(1, 10, 100, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(2, 10, 101, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(3, 10, 102, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(4, 10, 103, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(5, 10, 104, 30, domain.StatusConfirmed, time.Now()),
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return existing, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, nil, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrActivityFull)
}

func TestRegisterReservationUseCase_ActivityCancellationsFreeCapacity(t *testing.T) {
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	existing := []*domain.Reservation{
		mustNewReservation(1, 10, 100, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(2, 10, 101, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(3, 10, 102, 30, domain.StatusConfirmed, time.Now()),
		mustNewReservation(4, 10, 103, 30, domain.StatusCancelledInTime, time.Now()),
		mustNewReservation(5, 10, 104, 30, domain.StatusCancelledLate, time.Now()),
	}
	dog := validDog(20, 1)
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	pass := validPass(30, 1, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return existing, nil
		},
		create: func(context.Context, *domain.Reservation) (int, error) { return 99, nil },
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.NoError(t, err, "cancellations should free capacity (3 < 5)")
}

// noListActivity returns a non-nil mockReservationRepository that
// answers ListByActivity with an empty list. Used by tests that
// expect to fail at a check BEFORE the activity list (e.g., dog
// ownership); we still need a non-nil repo so the use case does
// not panic on a nil method call.
func noListActivity() *mockReservationRepository {
	return &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return nil, nil
		},
	}
}

func TestRegisterReservationUseCase_DogNotFound(t *testing.T) {
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidDog)
}

func TestRegisterReservationUseCase_DogNotOwnedByUser(t *testing.T) {
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dog := validDog(20, 99)
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidDog, "should not leak dog ownership")
}

func TestRegisterReservationUseCase_PassNotFound(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidPass)
}

func TestRegisterReservationUseCase_PassNotOwnedByUser(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	pass := validPass(30, 99, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidPass, "should not leak pass ownership")
}

func TestRegisterReservationUseCase_PassExhausted(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	pass := validPass(30, 1, 0)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrPassExhausted)
}

func TestRegisterReservationUseCase_PassExpired(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	now := fixedNow
	expiry := now.Add(-24 * time.Hour)
	pass := domain.MustNewPass(30, 5, 5, 1000, domain.PassGeneric, 1, now.Add(-48*time.Hour), now.Add(-48*time.Hour), &expiry)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrPassExpired)
}

func TestRegisterReservationUseCase_DuplicateReservation(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	pass := validPass(30, 1, 5)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			return 0, domain.ErrDuplicateReservation
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrDuplicateReservationForDog)
}

func TestRegisterReservationUseCase_TransactorRollsBackOnRepoError(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	pass := validPass(30, 1, 5)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
		update: func(context.Context, *domain.Pass) error { return nil },
		addMovement: func(context.Context, *domain.PassMovement) error {
			return errors.New("movement insert failed")
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			t.Fatal("reservation Create should not be called after AddMovement fails")
			return 0, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add movement")
}

func TestRegisterReservationUseCase_ActivityRepoErrorIsWrapped(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := newRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get activity")
	assert.Contains(t, err.Error(), "db connection lost")
}
