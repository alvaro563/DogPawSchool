package reservation

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

// mockReservationRepository is a hand-rolled mock used across the
// reservation use case tests. Each field is a function that the test
// sets; nil fields fall back to a sensible no-op so a test only needs
// to stub the methods it cares about.
type mockReservationRepository struct {
	create         func(ctx context.Context, reservation *domain.Reservation) (int, error)
	getByID        func(ctx context.Context, id int) (*domain.Reservation, error)
	update         func(ctx context.Context, reservation *domain.Reservation) error
	listByActivity func(ctx context.Context, activityID int) ([]*domain.Reservation, error)
	listByDog      func(ctx context.Context, dogID int) ([]*domain.Reservation, error)
	listByPass     func(ctx context.Context, passID int) ([]*domain.Reservation, error)

	// View methods (read paths).
	getView            func(ctx context.Context, id int) (*domain.ReservationView, error)
	listByUserView     func(ctx context.Context, userID int, status *domain.ReservationStatus, from, to *time.Time, limit, offset int) ([]*domain.ReservationView, error)
	listByUserUpcoming func(ctx context.Context, userID, limit, offset int) ([]*domain.ReservationView, error)
	listByDogView      func(ctx context.Context, dogID, limit, offset int) ([]*domain.ReservationView, error)
	listByPassView     func(ctx context.Context, passID, limit, offset int) ([]*domain.ReservationView, error)
	listByActivityView func(ctx context.Context, activityID, limit, offset int) ([]*domain.ReservationView, error)
}

func (m *mockReservationRepository) Create(ctx context.Context, reservation *domain.Reservation) (int, error) {
	if m.create != nil {
		return m.create(ctx, reservation)
	}
	return 0, nil
}

func (m *mockReservationRepository) GetByID(ctx context.Context, id int) (*domain.Reservation, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockReservationRepository) Update(ctx context.Context, reservation *domain.Reservation) error {
	if m.update != nil {
		return m.update(ctx, reservation)
	}
	return nil
}

func (m *mockReservationRepository) ListByActivity(ctx context.Context, activityID int) ([]*domain.Reservation, error) {
	if m.listByActivity != nil {
		return m.listByActivity(ctx, activityID)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByDog(ctx context.Context, dogID int) ([]*domain.Reservation, error) {
	if m.listByDog != nil {
		return m.listByDog(ctx, dogID)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByPass(ctx context.Context, passID int) ([]*domain.Reservation, error) {
	if m.listByPass != nil {
		return m.listByPass(ctx, passID)
	}
	return nil, nil
}

func (m *mockReservationRepository) GetView(ctx context.Context, id int) (*domain.ReservationView, error) {
	if m.getView != nil {
		return m.getView(ctx, id)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByUserView(
	ctx context.Context,
	userID int,
	status *domain.ReservationStatus,
	from, to *time.Time,
	limit, offset int,
) ([]*domain.ReservationView, error) {
	if m.listByUserView != nil {
		return m.listByUserView(ctx, userID, status, from, to, limit, offset)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByUserUpcomingView(ctx context.Context, userID, limit, offset int) ([]*domain.ReservationView, error) {
	if m.listByUserUpcoming != nil {
		return m.listByUserUpcoming(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByDogView(ctx context.Context, dogID, limit, offset int) ([]*domain.ReservationView, error) {
	if m.listByDogView != nil {
		return m.listByDogView(ctx, dogID, limit, offset)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByPassView(ctx context.Context, passID, limit, offset int) ([]*domain.ReservationView, error) {
	if m.listByPassView != nil {
		return m.listByPassView(ctx, passID, limit, offset)
	}
	return nil, nil
}

func (m *mockReservationRepository) ListByActivityView(ctx context.Context, activityID, limit, offset int) ([]*domain.ReservationView, error) {
	if m.listByActivityView != nil {
		return m.listByActivityView(ctx, activityID, limit, offset)
	}
	return nil, nil
}

// stubActivityRepository is the local mock for the activity repo used
// by the RegisterReservationUseCase. It mirrors the activity use
// case mock interface but is defined here so the reservation tests
// do not need to import the activity test package.
type stubActivityRepository struct {
	getByID func(ctx context.Context, id int) (*domain.Activity, error)
}

func (s *stubActivityRepository) GetByID(ctx context.Context, id int) (*domain.Activity, error) {
	if s.getByID != nil {
		return s.getByID(ctx, id)
	}
	return nil, nil
}

func (s *stubActivityRepository) Create(ctx context.Context, activity *domain.Activity) (int, error) {
	return 0, nil
}
func (s *stubActivityRepository) Update(ctx context.Context, activity *domain.Activity) error {
	return nil
}
func (s *stubActivityRepository) Delete(ctx context.Context, id int) error { return nil }
func (s *stubActivityRepository) List(ctx context.Context, _, _ int) ([]*domain.Activity, error) {
	return nil, nil
}
func (s *stubActivityRepository) ListUpcoming(ctx context.Context, _, _ int) ([]*domain.Activity, error) {
	return nil, nil
}

// stubDogRepository is the local mock for the dog repo. The use
// case only calls GetByID, so other methods are zero-value fallbacks.
type stubDogRepository struct {
	getByID func(ctx context.Context, id int) (*domain.Dog, error)
}

func (s *stubDogRepository) GetByID(ctx context.Context, id int) (*domain.Dog, error) {
	if s.getByID != nil {
		return s.getByID(ctx, id)
	}
	return nil, nil
}

func (s *stubDogRepository) Create(ctx context.Context, dog *domain.Dog) (int, error) {
	return 0, nil
}
func (s *stubDogRepository) Update(ctx context.Context, dog *domain.Dog) error { return nil }
func (s *stubDogRepository) ListByOwner(ctx context.Context, _, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListAll(ctx context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByIncompatibility(ctx context.Context, _, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByBreed(ctx context.Context, _ string, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListBySex(ctx context.Context, _ domain.Sex, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByNeutered(ctx context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByHeat(ctx context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByIsActive(ctx context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListByAgeBracket(ctx context.Context, _ domain.AgeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) ListBySizeBracket(ctx context.Context, _ domain.SizeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepository) SetDogNeutered(ctx context.Context, _ int, _ bool) error { return nil }
func (s *stubDogRepository) SetDogHeat(ctx context.Context, _ int, _ bool) error     { return nil }
func (s *stubDogRepository) Delete(ctx context.Context, _ int) error                 { return nil }

// stubPassRepository is the local mock for the pass repo. The use
// case calls GetByID, Update, and AddMovement.
type stubPassRepository struct {
	getByID     func(ctx context.Context, id int) (*domain.Pass, error)
	update      func(ctx context.Context, pass *domain.Pass) error
	addMovement func(ctx context.Context, movement *domain.PassMovement) error
}

func (s *stubPassRepository) GetByID(ctx context.Context, id int) (*domain.Pass, error) {
	if s.getByID != nil {
		return s.getByID(ctx, id)
	}
	return nil, nil
}
func (s *stubPassRepository) Update(ctx context.Context, pass *domain.Pass) error {
	if s.update != nil {
		return s.update(ctx, pass)
	}
	return nil
}
func (s *stubPassRepository) AddMovement(ctx context.Context, movement *domain.PassMovement) error {
	if s.addMovement != nil {
		return s.addMovement(ctx, movement)
	}
	return nil
}
func (s *stubPassRepository) Create(ctx context.Context, pass *domain.Pass) (int, error) {
	return 0, nil
}
func (s *stubPassRepository) ListAll(ctx context.Context, _, _ int) ([]*domain.Pass, error) {
	return nil, nil
}
func (s *stubPassRepository) ListByOwner(ctx context.Context, _, _, _ int) ([]*domain.Pass, error) {
	return nil, nil
}

// stubTransactor is a fake Transactor for use case tests. By default
// it invokes the closure synchronously without a real DB transaction;
// tests can swap fn to inject behaviour (e.g., a rollback path).
type stubTransactor struct {
	fn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (s *stubTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.fn != nil {
		return s.fn(ctx, fn)
	}
	return fn(ctx)
}

// mustNewReservation is a test helper that panics on construction
// error. Use it inside tests where the input is known to be valid.
func mustNewReservation(id, activityID, dogID, passID int, status domain.ReservationStatus, createdAt time.Time) *domain.Reservation {
	reservation, err := domain.NewReservationWithStatus(id, activityID, dogID, passID, status, createdAt)
	if err != nil {
		panic(err)
	}
	return reservation
}

// mustNewReservationView builds a domain.ReservationView from its
// four constituent aggregates. The dog uses id==userID for the
// userID and zero age/weight/sex (the read model only cares about
// the ids and the name; NewDog accepts these as long as id > 0 and
// userID > 0).
func mustNewReservationView(
	id, activityID, dogID, dogUserID, passID, passUserID int,
	status domain.ReservationStatus,
	createdAt time.Time,
	activityName, activityLocation string,
	activityDate time.Time,
	dogName string,
	passRemaining int,
) *domain.ReservationView {
	reservation := mustNewReservation(id, activityID, dogID, passID, status, createdAt)
	activity := domain.MustNewActivity(activityID, activityName, activityLocation,
		domain.TypeRoute, 5, 1, activityDate)
	dog, err := domain.NewDog(dogID, dogName, "TestBreed", "ES-TEST-"+strconv.Itoa(dogID),
		24, domain.SexMale, 10, dogUserID)
	if err != nil {
		panic(err)
	}
	pass := domain.MustNewPass(passID, 10, passRemaining, 1000, domain.PassGeneric,
		passUserID, createdAt, createdAt, nil)
	view, err := domain.NewReservationView(reservation, activity, dog, pass)
	if err != nil {
		panic(err)
	}
	return view
}

// sentinelErr is a small, import-free error used in tests across this
// package to verify that repository errors are wrapped correctly. It
// lives in a _test.go file so it is not part of the compiled binary.
var sentinelErr = errors.New("repo failure")

// assertValidationError is shared by every use case test in this
// package. It asserts err is a *ValidationError with the expected
// field name.
func assertValidationError(t *testing.T, err error, wantField string) {
	t.Helper()
	var validationErr *ValidationError
	if assert.True(t, errors.As(err, &validationErr), "expected ValidationError, got %T (%v)", err, err) {
		assert.Equal(t, wantField, validationErr.Field)
	}
}
