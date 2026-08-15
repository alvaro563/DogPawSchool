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
	pass := validPass(30, 88, 5) // owned by user 88

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
		create:         func(context.Context, *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := newAdminRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validAdminRegisterInput())
	assert.NoError(t, err, "admin must be allowed to book any dog regardless of ownership")
}

func TestAdminRegister_PassOwnedByAnotherUser(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	dog := validDog(20, 1)
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
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
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
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
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
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
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
	pass := validPass(30, 88, 0) // exhausted

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
	pass := domain.MustNewPass(30, 5, 5, 1000, domain.PassGeneric, 88, now.Add(-48*time.Hour), now.Add(-48*time.Hour), &expiry)

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
	pass := validPass(30, 88, 5)

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
	pass := validPass(30, 88, 5)

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
