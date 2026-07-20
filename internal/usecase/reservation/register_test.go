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
	"dogpaw/internal/repository/postgres"
)

// validRegisterInput returns a known-good input. Tests mutate one
// field at a time to cover the negative cases.
func validRegisterInput() RegisterReservationInput {
	return RegisterReservationInput{
		UserID:     1,
		ActivityID: 10,
		DogID:      20,
		PassID:     30,
	}
}

// validFutureActivity returns an activity in the future, with room
// for at least one more booking.
func validFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, time.Now().Add(7*24*time.Hour))
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
// remaining sessions, no expiry. It always starts from
// numOfSessions=10 and consumes the difference to land at the
// requested remaining (so callers can pass remaining=0 without
// tripping the constructor's numOfSessions>0 check).
func validPass(id, userID, remaining int) *domain.Pass {
	now := time.Now()
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
	return NewRegisterReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo)
}

func TestRegisterReservationUseCase_Success(t *testing.T) {
	now := time.Now()
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
			return nil, nil // no existing bookings → capacity available
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
	assert.Equal(t, 99, output.ID, "use case should return the DB-assigned id from the create stub")
	require.NotNil(t, capturedReservation)
	// capturedReservation is the in-memory object BEFORE the DB
	// assigns the id, so it has id=0. The post-insert id is in
	// output.ID. We still verify the rest of the fields here.
	assert.Equal(t, 0, capturedReservation.ID(), "in-memory reservation has id=0 before DB insert")
	assert.Equal(t, 10, capturedReservation.ActivityID())
	// The pass update was observed in the stub; the in-memory pass
	// now reflects the consumption.
	assert.Equal(t, 4, pass.RemainingSessions())
	assert.True(t, now.Before(pass.Movements()[0].CreatedAt().Add(time.Second)),
		"movement createdAt should be ~now")
}

func TestRegisterReservationUseCase_ValidationErrors(t *testing.T) {
	base := validRegisterInput()
	tests := []struct {
		name      string
		mutate    func(input *RegisterReservationInput)
		wantField string
	}{
		{
			name:      "zero_user_id",
			mutate:    func(i *RegisterReservationInput) { i.UserID = 0 },
			wantField: "user_id",
		},
		{
			name:      "negative_user_id",
			mutate:    func(i *RegisterReservationInput) { i.UserID = -1 },
			wantField: "user_id",
		},
		{
			name:      "zero_activity_id",
			mutate:    func(i *RegisterReservationInput) { i.ActivityID = 0 },
			wantField: "activity_id",
		},
		{
			name:      "zero_dog_id",
			mutate:    func(i *RegisterReservationInput) { i.DogID = 0 },
			wantField: "dog_id",
		},
		{
			name:      "zero_pass_id",
			mutate:    func(i *RegisterReservationInput) { i.PassID = 0 },
			wantField: "pass_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)
			// All repos wired with asserts that the use case does
			// not reach the repo layer on validation failure.
			activityRepo := &stubActivityRepository{
				getByID: func(context.Context, int) (*domain.Activity, error) {
					t.Fatal("activityRepo.GetByID should not be called")
					return nil, nil
				},
			}
			dogRepo := &stubDogRepository{
				getByID: func(context.Context, int) (*domain.Dog, error) {
					t.Fatal("dogRepo.GetByID should not be called")
					return nil, nil
				},
			}
			passRepo := &stubPassRepository{
				getByID: func(context.Context, int) (*domain.Pass, error) {
					t.Fatal("passRepo.GetByID should not be called")
					return nil, nil
				},
			}
			reservationRepo := &mockReservationRepository{
				listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
					t.Fatal("reservationRepo.ListByActivity should not be called")
					return nil, nil
				},
			}
			uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
			_, err := uc.Execute(context.Background(), input)
			assertValidationError(t, err, tt.wantField)
		})
	}
}

func TestRegisterReservationUseCase_ActivityNotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, postgres.ErrActivityNotFound
		},
	}
	uc := newRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestRegisterReservationUseCase_ActivityInPast(t *testing.T) {
	pastActivity := domain.MustNewActivity(10, "Paseo", "Central", domain.TypeRoute, 5, 1,
		time.Now().Add(-24*time.Hour))
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
	// 5 max capacity, 5 CONFIRMED → full
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
	// 5 max capacity, 3 CONFIRMED + 2 CANCELLED → not full
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
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	pass := validPass(30, 1, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
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
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return nil, postgres.ErrNotFound
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidDog)
}

func TestRegisterReservationUseCase_DogNotOwnedByUser(t *testing.T) {
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	// Dog belongs to user 99, but the request is for user 1.
	dog := validDog(20, 99)
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidDog, "should not leak dog ownership")
}

func TestRegisterReservationUseCase_PassNotFound(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
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
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidPass)
}

func TestRegisterReservationUseCase_PassNotOwnedByUser(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	// Pass belongs to user 99.
	pass := validPass(30, 99, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidPass, "should not leak pass ownership")
}

func TestRegisterReservationUseCase_PassExhausted(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	pass := validPass(30, 1, 0) // 0 remaining
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrPassExhausted)
}

func TestRegisterReservationUseCase_PassExpired(t *testing.T) {
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	now := time.Now()
	expiry := now.Add(-24 * time.Hour) // expired yesterday
	pass := domain.MustNewPass(30, 5, 5, 1000, domain.PassGeneric, 1, now.Add(-48*time.Hour), now.Add(-48*time.Hour), &expiry)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
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
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			return 0, postgres.ErrDuplicateReservation
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.ErrorIs(t, err, ErrDuplicateReservationForDog)
}

func TestRegisterReservationUseCase_TransactorRollsBackOnRepoError(t *testing.T) {
	// The pass AddMovement step fails; the transactor must roll back
	// the transaction so no partial state is persisted. The use case
	// returns the underlying error wrapped.
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
	pass := validPass(30, 1, 5)
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
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		// create should not be called because the tx rolls back
		// before the reservation is inserted.
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
	// Non-sentinel errors are wrapped (no mapping), so the handler
	// surfaces them as 500.
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
