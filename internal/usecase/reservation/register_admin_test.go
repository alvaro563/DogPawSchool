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

func validAdminRegisterInput() RegisterAdminReservationInput {
	in, err := NewRegisterAdminReservationInput(10, 20, 30, func() time.Time { return fixedNow })
	if err != nil {
		panic(err)
	}
	return in
}

func newAdminRegisterUseCase(
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
	transactor Transactor,
) *RegisterAdminReservationUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewRegisterAdminReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo)
}

// ── Input validation ──────────────────────────────────────────────

func TestNewRegisterAdminReservationInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		actID  int
		dogID  int
		passID int
		field  string
	}{
		{"zero_activity_id", 0, 20, 30, "activity_id"},
		{"negative_activity_id", -1, 20, 30, "activity_id"},
		{"zero_dog_id", 10, 0, 30, "dog_id"},
		{"negative_dog_id", 10, -1, 30, "dog_id"},
		{"zero_pass_id", 10, 20, 0, "pass_id"},
		{"negative_pass_id", 10, 20, -1, "pass_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRegisterAdminReservationInput(tt.actID, tt.dogID, tt.passID, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.field, verr.Field)
		})
	}
}

func TestNewRegisterAdminReservationInput_NilNow(t *testing.T) {
	t.Parallel()
	in, err := NewRegisterAdminReservationInput(10, 20, 30, nil)
	require.NoError(t, err)
	assert.False(t, in.Now().IsZero(), "nil now provider should fall back to time.Now")
}

// ── Success ───────────────────────────────────────────────────────

func TestAdminRegister_Success(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)   // owned by user 99
	pass := validPass(30, 99, 5) // owned by the same user 99 (admin only acts on behalf)

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
			return 99, nil
		},
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())

	require.NoError(t, err)
	assert.Equal(t, 99, output.ID)
	assert.Equal(t, domain.StatusConfirmed, output.Status)
	require.NotNil(t, capturedReservation)
	assert.Equal(t, 0, capturedReservation.ID(), "in-memory reservation has id=0 before DB insert")
	assert.Equal(t, 4, pass.RemainingSessions())
}

// ── Ownership bypass ──────────────────────────────────────────────

func TestAdminRegister_DogOwnedByAnotherUser(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99) // owned by user 99, not the admin
	pass := validPass(30, 99, 5)

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
		create:         func(context.Context, *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.NoError(t, err, "admin must be allowed to book any dog regardless of ownership")
}

func TestAdminRegister_PassOwnedByAnotherUser(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	pass := validPass(30, 99, 5) // owned by user 99

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
		create:         func(context.Context, *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.NoError(t, err, "admin must be allowed to use any pass regardless of ownership")
}

// ── Compatibility conflicts: admin forces CONFIRMED ────────────────

func TestAdminRegister_MediumConflictForcesConfirmed(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 99, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := adminFlowStubs(t, candidate, other, 99)
	var capturedStatus domain.ReservationStatus
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		capturedStatus = r.Status()
		return 99, nil
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, output.Status, "admin must force CONFIRMED even with MEDIA conflict")
	assert.Equal(t, domain.StatusConfirmed, capturedStatus)
}

func TestAdminRegister_BajaConflictForcesConfirmed(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 99, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelBaja, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := adminFlowStubs(t, candidate, other, 99)
	var capturedStatus domain.ReservationStatus
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		capturedStatus = r.Status()
		return 99, nil
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, output.Status, "admin must force CONFIRMED even with Baja conflict")
	assert.Equal(t, domain.StatusConfirmed, capturedStatus)
}

func TestAdminRegister_AbsoluteConflictStillBypassed(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 99, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO"))
	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	activityRepo, dogRepo, passRepo, reservationRepo := adminFlowStubs(t, candidate, other, 99)
	var capturedStatus domain.ReservationStatus
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		capturedStatus = r.Status()
		return 99, nil
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err, "admin authority overrides ABSOLUTA conflicts")
	assert.Equal(t, domain.StatusConfirmed, output.Status, "admin must force CONFIRMED even with ABSOLUTA conflict")
	assert.Equal(t, domain.StatusConfirmed, capturedStatus)
}

// ── Edge cases ────────────────────────────────────────────────────

func TestAdminRegister_PassExhausted(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	pass := validPass(30, 99, 0) // exhausted

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrPassExhausted, "admin cannot use an exhausted pass")
}

func TestAdminRegister_PassExpired(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	now := fixedNow
	expiry := now.Add(-24 * time.Hour)
	pass := domain.MustNewPass(30, 5, 5, 1000, domain.PassGeneric, 99, now.Add(-48*time.Hour), now.Add(-48*time.Hour), &expiry)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrPassExpired, "admin cannot use an expired pass")
}

func TestAdminRegister_ActivityFull(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	existing := []*domain.Reservation{
		mustNewReservation(1, 10, 100, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(2, 10, 101, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(3, 10, 102, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(4, 10, 103, 30, domain.StatusConfirmed, fixedNow),
		mustNewReservation(5, 10, 104, 30, domain.StatusConfirmed, fixedNow),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return existing, nil },
	}
	uc := newAdminRegisterUseCase(activityRepo, nil, nil, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrActivityFull, "admin cannot book a full activity")
}

func TestAdminRegister_DuplicateReservation(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	pass := validPass(30, 99, 5)

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
			return 0, domain.ErrDuplicateReservation
		},
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrDuplicateReservationForDog, "admin cannot create duplicate reservation")
}

// ── Pass consumption ──────────────────────────────────────────────

func TestAdminRegister_PassSessionConsumed(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	pass := validPass(30, 99, 5)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	var updatedPass *domain.Pass
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update: func(_ context.Context, p *domain.Pass) error {
			updatedPass = p
			return nil
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create:         func(context.Context, *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	require.NotNil(t, updatedPass)
	assert.Equal(t, 4, updatedPass.RemainingSessions(), "one session must be consumed: 5 → 4")
}

// ── Dog + Pass consistency (regression: Ana's dog must not consume Juan's pass) ─

func TestAdminRegister_DogAndPassOwnerMismatch(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 1)        // owned by Ana (user 1)
	pass := validPass(30, 99, 5)  // owned by Juan (user 99)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	var passUpdated, reservationCreated bool
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			reservationCreated = true
			return 0, nil
		},
	}
	passRepo.update = func(context.Context, *domain.Pass) error {
		passUpdated = true
		return nil
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrDogPassOwnerMismatch, "admin must not be allowed to combine Ana's dog with Juan's pass")
	assert.False(t, reservationCreated, "no reservation may be created on a mismatch")
	assert.False(t, passUpdated, "the pass session must not be consumed on a mismatch")
	assert.Equal(t, 5, pass.RemainingSessions(), "pass counter untouched on rejection")
}

func TestAdminRegister_DogAndPassSameOwnerPasses(t *testing.T) {
	t.Parallel()
	// Positive symmetric test for the mismatch guard above: when both
	// the dog and the pass belong to the same user (here, Juan = 99),
	// the admin can complete the booking on their behalf.
	const ownerID = 99
	activity := validFutureActivity(10)
	dog := validDog(20, ownerID)
	pass := validPass(30, ownerID, 5)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	var capturedReservation *domain.Reservation
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(_ context.Context, r *domain.Reservation) (int, error) {
			capturedReservation = r
			assert.Equal(t, domain.StatusConfirmed, r.Status())
			return 42, nil
		},
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, 42, output.ID)
	assert.Equal(t, domain.StatusConfirmed, output.Status)
	require.NotNil(t, capturedReservation)
	assert.Equal(t, 4, pass.RemainingSessions(), "session consumed because owners match")
}

// ── Negative paths (missing entities) ──────────────────────────────

func TestAdminRegister_ActivityNotFound(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return nil, domain.ErrNotFound },
	}
	uc := newAdminRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestAdminRegister_ActivityInPast(t *testing.T) {
	t.Parallel()
	past := domain.MustNewActivity(10, "Paseo", "", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-24*time.Hour))
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return past, nil },
	}
	uc := newAdminRegisterUseCase(activityRepo, nil, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrActivityInPast)
}

func TestAdminRegister_DogNotFound(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return nil, domain.ErrNotFound },
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, nil, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidDog)
}

func TestAdminRegister_PassNotFound(t *testing.T) {
	t.Parallel()
	dog := validDog(20, 99)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return nil, domain.ErrNotFound },
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, noListActivity(), nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.ErrorIs(t, err, ErrInvalidPass)
}

// ── Repository error wrapping ──────────────────────────────────────

func TestAdminRegister_ActivityRepoErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := newAdminRegisterUseCase(activityRepo, nil, nil, nil, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get activity")
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestAdminRegister_ListByActivityErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return validFutureActivity(10), nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return validDog(20, 99), nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return nil, errors.New("query timeout")
		},
	}
	pass := validPass(30, 99, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list reservations for activity")
	assert.Contains(t, err.Error(), "query timeout")
}

func TestAdminRegister_PassUpdateErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return validDog(20, 99), nil },
	}
	pass := validPass(30, 99, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
		update: func(context.Context, *domain.Pass) error {
			return errors.New("movement insert failed")
		},
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) { return nil, nil },
		create: func(context.Context, *domain.Reservation) (int, error) {
			t.Fatal("reservation Create must not be called after pass Update fails")
			return 0, nil
		},
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "movement insert failed")
}

// ── Admin-specific behaviour (regression guards) ───────────────────

func TestAdminRegister_GetByIDsNotCalledOnAdminPath(t *testing.T) {
	t.Parallel()
	// Admin path must skip the compatibility check entirely, so the
	// dog repo's GetByIDs (which loads slot-holder dogs for the
	// conflict evaluation) must never be invoked. This is both a
	// coverage guard and a regression guard: if someone later
	// refactors the adminOverride branch and accidentally enters the
	// compatibility block, this test fails immediately.
	activity := validFutureActivity(10)
	dog := validDog(20, 99)
	pass := validPass(30, 99, 5)
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
		getByIDs: func(context.Context, []int) ([]*domain.Dog, error) {
			t.Fatal("GetByIDs must not be called on admin path (compatibility check is bypassed)")
			return nil, nil
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{
				mustNewReservation(1, 10, 21, 30, domain.StatusConfirmed, fixedNow),
			}, nil
		},
		create: func(context.Context, *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.NoError(t, err)
}

func TestAdminRegister_NoConflictStaysConfirmed(t *testing.T) {
	t.Parallel()
	// The candidate carries a trigger, but the dog holding the slot
	// presents no matching trait. Admin path skips the compatibility
	// block entirely, so the result is CONFIRMED.
	candidate := dogWithTrigger(20, 99, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := validDog(21, 1) // no MACHO_ENTERO trait
	activityRepo, dogRepo, passRepo, reservationRepo := adminFlowStubs(t, candidate, other, 99)
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		assert.Equal(t, domain.StatusConfirmed, r.Status())
		return 99, nil
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, output.Status)
}

func TestAdminRegister_BidirectionalConflictBypassed(t *testing.T) {
	t.Parallel()
	// Both directions fire (candidate trigger on MACHO_ENTERO, other
	// trigger on ALTA_ENERGIA; each presents the matching trait). On
	// the user path this would land at StatusPendingToConfirm; on the
	// admin path the entire compatibility block is bypassed, so the
	// result is CONFIRMED.
	candidate := dogWithTrigger(20, 99, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	_, _ = candidate.AddTrait(mustTrait(3, "ALTA_ENERGIA", "Alta energía", domain.IncompatibilityLevelBaja))

	other := dogWithTrait(21, 1, mustTrait(2, "MACHO_ENTERO", "Macho entero (no castrado)", domain.IncompatibilityLevelBaja))
	_, _ = other.AddIncompatibility(mustTrigger(4, "Reactivo a alta energía", domain.IncompatibilityLevelMedia, "ALTA_ENERGIA"))

	activityRepo, dogRepo, passRepo, reservationRepo := adminFlowStubs(t, candidate, other, 99)
	var capturedStatus domain.ReservationStatus
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		capturedStatus = r.Status()
		return 99, nil
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validAdminRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, output.Status)
	assert.Equal(t, domain.StatusConfirmed, capturedStatus)
}

func TestAdminRegister_PendingReservationAlsoHoldsSlot(t *testing.T) {
	t.Parallel()
	// PENDING_TO_CONFIRM reservations occupy their slot for the
	// capacity check on the admin path too (admin only bypasses
	// ownership and compatibility, not capacity). One pending dog +
	// one extra slot → admin book 2nd dog succeeds.
	activity := domain.MustNewActivity(10, "Paseo", "", "Central", domain.TypeRoute, 2, 1, fixedNow.Add(7*24*time.Hour))
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dog := validDog(20, 99)
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	pass := validPass(30, 99, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{
				mustNewReservation(1, 10, 21, 30, domain.StatusPendingToConfirm, fixedNow),
			}, nil
		},
		create: func(_ context.Context, r *domain.Reservation) (int, error) {
			assert.Equal(t, domain.StatusConfirmed, r.Status())
			return 99, nil
		},
	}
	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.NoError(t, err, "capacity 2 with 1 pending slot-holder leaves room for one more booking")
}

// ── Shared helper for admin-path conflict tests ────────────────────

// adminFlowStubs is the admin-path equivalent of registerFlowStubs.
// It differs from the user-path helper in three ways:
//  1. The pass is created for the given ownerID so dog and pass
//     always belong to the same user (the new consistency guard).
//  2. The dog repo's GetByIDs returns a fatal error if invoked — the
//     admin path must not load slot-holder dogs (compatibility is
//     bypassed). Tests that *want* to drive the compatibility code
//     must use the user-path helper instead.
//  3. The listByActivity stub seeds one existing confirmed
//     reservation for the "other" dog so capacity is exercised.
func adminFlowStubs(t *testing.T, candidate, other *domain.Dog, ownerID int) (
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
			return candidate, nil
		},
		getByIDs: func(context.Context, []int) ([]*domain.Dog, error) {
			t.Fatal("GetByIDs must not be called on admin path (compatibility is bypassed)")
			return nil, nil
		},
	}
	pass := validPass(30, ownerID, 5)
	passRepo := &stubPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) { return pass, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivity: func(context.Context, int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{
				mustNewReservation(1, 10, other.ID(), 30, domain.StatusConfirmed, fixedNow),
			}, nil
		},
	}
	return activityRepo, dogRepo, passRepo, reservationRepo
}
