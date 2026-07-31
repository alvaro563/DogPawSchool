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
	return NewRegisterReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo)
}

func TestNewRegisterReservationInput(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	assert.True(t, now.Before(pass.PendingMovements()[0].CreatedAt().Add(time.Second)),
		"movement createdAt should be ~now")
}

func TestRegisterReservationUseCase_ActivityNotFound(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	existing := []*domain.Reservation{
		mustNewReservation(1, 10, 100, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(2, 10, 101, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(3, 10, 102, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(4, 10, 103, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(5, 10, 104, 30, domain.StatusConfirmed, fixedNow),
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
	t.Parallel()
	activity := validFutureActivity(10)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return activity, nil
		},
	}
	existing := []*domain.Reservation{
		mustNewReservation(1, 10, 100, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(2, 10, 101, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(3, 10, 102, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(4, 10, 103, 30, domain.StatusCancelledInTime, fixedNow),
		mustNewReservation(5, 10, 104, 30, domain.StatusCancelledLate, fixedNow),
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	// PassRepository.Update now persists the counter and the audit
	// movement together, so a failed audit insert surfaces as a failed
	// Update. No reservation may be created afterwards.
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			return pass, nil
		},
		update: func(context.Context, *domain.Pass) error {
			return errors.New("add pass movement: movement insert failed")
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			t.Fatal("reservation Create should not be called after the pass Update fails")
			return 0, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "movement insert failed")
}

func TestRegisterReservationUseCase_ActivityRepoErrorIsWrapped(t *testing.T) {
	t.Parallel()
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

// mustTrigger builds a TRIGGER pointing at a target trait code.
func mustTrigger(id int, name string, level domain.IncompatibilityLevel, target string) *domain.Incompatibility {
	trigger, err := domain.NewTriggerIncompatibility(id, name, level, target)
	if err != nil {
		panic(err)
	}
	return trigger
}

// mustTrait builds a TRAIT identified by its code.
func mustTrait(id int, code, name string, level domain.IncompatibilityLevel) *domain.Incompatibility {
	trait, err := domain.NewTraitIncompatibility(id, code, name, level)
	if err != nil {
		panic(err)
	}
	return trait
}

// dogWithTrigger returns a dog carrying the given trigger.
func dogWithTrigger(id, userID int, trigger *domain.Incompatibility) *domain.Dog {
	dog := validDog(id, userID)
	_, _ = dog.AddIncompatibility(trigger)
	return dog
}

// dogWithTrait returns a dog presenting the given trait.
func dogWithTrait(id, userID int, trait *domain.Incompatibility) *domain.Dog {
	dog := validDog(id, userID)
	_, _ = dog.AddTrait(trait)
	return dog
}

// registerFlowStubs returns the repos for a valid register flow where
// one existing confirmed reservation (dog 21) already holds a slot and
// other is the dog that holds it.
func registerFlowStubs(t *testing.T, dog, other *domain.Dog) (
	*stubActivityRepository, *stubDogRepository, *stubPassRepository, *mockReservationRepository,
) {
	t.Helper()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return validFutureActivity(10), nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return dog, nil
		},
		getByIDs: func(_ context.Context, ids []int) ([]*domain.Dog, error) {
			assert.Equal(t, []int{21}, ids, "slot holders must be loaded for the compatibility check")
			return []*domain.Dog{other}, nil
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
			return []*domain.Reservation{
				mustNewReservation(1, 10, 21, 30, domain.StatusConfirmed, fixedNow),
			}, nil
		},
	}
	return activityRepo, dogRepo, passRepo, reservationRepo
}

func TestRegisterReservationUseCase_AbsoluteConflictBlocks(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
	var createCalled bool
	reservationRepo.create = func(context.Context, *domain.Reservation) (int, error) {
		createCalled = true
		t.Fatal("reservation Create must not be called when an ABSOLUTA conflict blocks")
		return 0, nil
	}
	var passUpdated bool
	passRepo.update = func(context.Context, *domain.Pass) error {
		passUpdated = true
		return nil
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	require.Error(t, err)
	var incompatErr *IncompatibleDogsError
	assert.True(t, errors.As(err, &incompatErr), "expected IncompatibleDogsError, got %T", err)
	assert.Len(t, incompatErr.Conflicts, 1)
	assert.False(t, createCalled, "no reservation may be created")
	assert.False(t, passUpdated, "the pass session must not be consumed when the booking is blocked")
}

func TestRegisterReservationUseCase_MediumConflictCreatesPending(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
	var capturedStatus domain.ReservationStatus
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		capturedStatus = r.Status()
		return 99, nil
	}
	var passUpdated bool
	passRepo.update = func(context.Context, *domain.Pass) error {
		passUpdated = true
		return nil
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPendingToConfirm, output.Status, "a MEDIA conflict holds the slot pending")
	assert.Equal(t, domain.StatusPendingToConfirm, capturedStatus)
	assert.True(t, passUpdated, "the pass session is still consumed: the slot is held")
}

func TestRegisterReservationUseCase_BajaConflictAlsoPending(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelBaja, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		assert.Equal(t, domain.StatusPendingToConfirm, r.Status())
		return 99, nil
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPendingToConfirm, output.Status)
}

func TestRegisterReservationUseCase_NoConflictStaysConfirmed(t *testing.T) {
	t.Parallel()
	// The candidate carries a trigger, but the dog holding the slot
	// presents no matching trait, so the evaluation finds no conflict.
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := validDog(21, 1)
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		assert.Equal(t, domain.StatusConfirmed, r.Status())
		return 99, nil
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, output.Status)
}

func TestRegisterReservationUseCase_PendingReservationAlsoHoldsSlot(t *testing.T) {
	t.Parallel()
	// A PENDING_TO_CONFIRM reservation occupies its slot for the capacity
	// check AND its dog is loaded for the compatibility check.
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return validFutureActivity(10), nil
		},
	}
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) {
			return candidate, nil
		},
		getByIDs: func(_ context.Context, ids []int) ([]*domain.Dog, error) {
			assert.Equal(t, []int{21}, ids)
			return []*domain.Dog{other}, nil
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
			return []*domain.Reservation{
				mustNewReservation(1, 10, 21, 30, domain.StatusPendingToConfirm, fixedNow),
			}, nil
		},
		create: func(_ context.Context, r *domain.Reservation) (int, error) {
			assert.Equal(t, domain.StatusPendingToConfirm, r.Status())
			return 99, nil
		},
	}
	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPendingToConfirm, output.Status)
}
