package reservation

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	"dogpaw/migrations"
)

// integrationNow anchors every time-dependent value (activity dates, pass
// timestamps, and the frozen clock injected into the use cases) so the
// suite is deterministic regardless of the wall clock.
var integrationNow = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

// testDB is the shared connection to the Testcontainers Postgres. It is
// set by TestMain after the real migrations have run.
var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c, connStr, err := startPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("reservation integration: start container: %v", err)
	}
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("reservation integration: terminate container: %v", err)
		}
	}()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("reservation integration: sql.Open: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("reservation integration: db.Ping: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("reservation integration: migrate: %v", err)
	}

	testDB = db
	os.Exit(m.Run())
}

func startPostgresContainer(ctx context.Context) (*tcpostgres.PostgresContainer, string, error) {
	c, err := tcpostgres.Run(ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("dogpaw_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategyAndDeadline(
			60*time.Second,
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, "", err
	}
	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}
	return c, connStr, nil
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	defer src.Close()
	driver, err := migratepg.WithInstance(db, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// cleanTables wipes every data table between tests. incompatibilities is
// included so each test is fully self-contained (the migration seeds are
// re-created on demand by the seed helpers below).
func cleanTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"pass_movements",
		"reservations",
		"invitations",
		"dog_incompatibilities",
		"passes",
		"dogs",
		"activities",
		"incompatibilities",
		"users",
	}
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE "+strings.Join(tables, ",")+" RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// newIntegrationRegisterUC wires RegisterReservationUseCase with the real
// Postgres repositories and transactor.
func newIntegrationRegisterUC() *RegisterReservationUseCase {
	return NewRegisterReservationUseCase(
		postgres.NewTransactor(testDB),
		postgres.NewActivityRepository(testDB),
		postgres.NewDogRepository(testDB),
		postgres.NewPassRepository(testDB),
		postgres.NewReservationRepository(testDB),
	)
}

// newIntegrationRejectUC wires RejectPendingReservationUseCase with the
// real Postgres repositories and transactor.
func newIntegrationRejectUC() *RejectPendingReservationUseCase {
	return NewRejectPendingReservationUseCase(
		postgres.NewTransactor(testDB),
		postgres.NewPassRepository(testDB),
		postgres.NewReservationRepository(testDB),
	)
}

func seedIntegrationUser(t *testing.T, email string) *domain.User {
	t.Helper()
	user, err := domain.NewUser(0, "Integration Owner", email, strings.Repeat("a", 60), domain.RoleRegular)
	require.NoError(t, err)
	repo := postgres.NewUserRepository(testDB)
	_, err = repo.Create(context.Background(), user)
	require.NoError(t, err)
	got, err := repo.GetByEmail(context.Background(), email)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func seedIntegrationActivity(t *testing.T, capacity int, date time.Time) *domain.Activity {
	t.Helper()
	activity, err := domain.NewActivity(0, "Paseo Integración", "Parque Central",
		domain.TypeRoute, capacity, 1, date)
	require.NoError(t, err)
	repo := postgres.NewActivityRepository(testDB)
	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func seedIntegrationTrait(t *testing.T, code, name string) *domain.Incompatibility {
	t.Helper()
	trait, err := domain.NewTraitIncompatibility(0, code, name, domain.IncompatibilityLevelBaja)
	require.NoError(t, err)
	return createIntegrationIncompatibility(t, trait)
}

func seedIntegrationTrigger(t *testing.T, name string, level domain.IncompatibilityLevel, targetCode string) *domain.Incompatibility {
	t.Helper()
	trigger, err := domain.NewTriggerIncompatibility(0, name, level, targetCode)
	require.NoError(t, err)
	return createIntegrationIncompatibility(t, trigger)
}

func createIntegrationIncompatibility(t *testing.T, incomp *domain.Incompatibility) *domain.Incompatibility {
	t.Helper()
	repo := postgres.NewIncompatibilityRepository(testDB)
	id, err := repo.Create(context.Background(), incomp)
	require.NoError(t, err)
	got, err := repo.GetIncompatibilityByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func seedIntegrationDog(t *testing.T, userID int, name string, traits, triggers []*domain.Incompatibility) *domain.Dog {
	t.Helper()
	dog, err := domain.NewDog(0, name, "Labrador", "ES-INT-"+strings.ToUpper(name), 24,
		domain.SexFemale, 22.5, userID)
	require.NoError(t, err)
	repo := postgres.NewDogRepository(testDB)
	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)

	for _, trait := range traits {
		_, err := got.AddTrait(trait)
		require.NoError(t, err)
	}
	for _, trigger := range triggers {
		_, err := got.AddIncompatibility(trigger)
		require.NoError(t, err)
	}
	require.NoError(t, repo.Update(context.Background(), got))
	return got
}

func seedIntegrationPass(t *testing.T, userID, sessions int) *domain.Pass {
	t.Helper()
	pass, err := domain.NewPass(0, sessions, sessions, 5000, domain.PassGeneric, userID,
		integrationNow, integrationNow, nil)
	require.NoError(t, err)
	repo := postgres.NewPassRepository(testDB)
	id, err := repo.Create(context.Background(), pass)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func seedIntegrationReservation(t *testing.T, activityID, dogID, passID int, status domain.ReservationStatus, createdAt time.Time) *domain.Reservation {
	t.Helper()
	res, err := domain.NewReservationWithStatus(0, activityID, dogID, passID, status, createdAt)
	require.NoError(t, err)
	repo := postgres.NewReservationRepository(testDB)
	id, err := repo.Create(context.Background(), res)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

// TestConflictCandidateTrigger validates the candidate->class direction:
// Dog A (already in class) presents the trait MACHO_ENTERO and the
// candidate carries an ABSOLUTA trigger on it. The booking must fail with
// IncompatibleDogsError and the pass must be untouched.
func TestConflictCandidateTrigger(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "candidate-trigger@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", []*domain.Incompatibility{trait}, nil)
	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, []*domain.Incompatibility{trigger})
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	// Dog A already holds a slot in the activity.
	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	_, err := uc.Execute(context.Background(), in)

	require.Error(t, err)
	var incompatErr *IncompatibleDogsError
	require.True(t, errors.As(err, &incompatErr), "expected IncompatibleDogsError, got %v", err)
	require.Len(t, incompatErr.Conflicts, 1)
	require.Equal(t, domain.IncompatibilityLevelAbsoluta, incompatErr.Conflicts[0].TriggerLevel)

	// The candidate's pass must not be touched: no session consumed.
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 10, gotPass.RemainingSessions(), "pass must not be touched on an ABSOLUTA conflict")

	// And no reservation may have been created for the candidate.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 1, "only Dog A's reservation must exist")
}

// TestConflictExistingTrigger validates the class->candidate direction:
// Dog A (already in class) carries an ABSOLUTA trigger on MACHO_ENTERO and
// the candidate presents that trait. The booking must fail with
// IncompatibleDogsError and the pass must be untouched.
func TestConflictExistingTrigger(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "existing-trigger@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", nil, []*domain.Incompatibility{trigger})
	candidate := seedIntegrationDog(t, user.ID(), "Luna", []*domain.Incompatibility{trait}, nil)
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	_, err := uc.Execute(context.Background(), in)

	require.Error(t, err)
	var incompatErr *IncompatibleDogsError
	require.True(t, errors.As(err, &incompatErr), "expected IncompatibleDogsError, got %v", err)
	require.Len(t, incompatErr.Conflicts, 1)
	require.Equal(t, domain.IncompatibilityLevelAbsoluta, incompatErr.Conflicts[0].TriggerLevel)

	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 10, gotPass.RemainingSessions(), "pass must not be touched on an ABSOLUTA conflict")

	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 1, "only Dog A's reservation must exist")
}

// TestMediumSeverityCreatesPending validates that a MEDIA conflict does
// not block the booking: the reservation is created in
// PENDING_TO_CONFIRM (slot held) and exactly one pass session is
// consumed.
func TestMediumSeverityCreatesPending(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "medium-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", []*domain.Incompatibility{trait}, nil)
	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, []*domain.Incompatibility{trigger})
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out.Status)

	// The PENDING reservation is persisted in the DB.
	reservationRepo := postgres.NewReservationRepository(testDB)
	reservations, err := reservationRepo.ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 2)
	var pending *domain.Reservation
	for _, r := range reservations {
		if r.DogID() == candidate.ID() {
			pending = r
		}
	}
	require.NotNil(t, pending, "candidate reservation must exist")
	require.Equal(t, domain.StatusPendingToConfirm, pending.Status())

	// Exactly one session consumed from the candidate's pass.
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 9, gotPass.RemainingSessions(), "MEDIA conflict consumes one session (the slot is held)")
}

// TestHoldsSlotCapacity validates that a PENDING_TO_CONFIRM reservation
// occupies its slot: with a capacity-2 activity holding 1 CONFIRMED and 1
// PENDING booking, a third dog cannot register.
func TestHoldsSlotCapacity(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "capacity-owner@test.com")
	activity := seedIntegrationActivity(t, 2, now.Add(7*24*time.Hour))

	dogA := seedIntegrationDog(t, user.ID(), "Rex", nil, nil)
	dogB := seedIntegrationDog(t, user.ID(), "Bolt", nil, nil)
	dogC := seedIntegrationDog(t, user.ID(), "Luna", nil, nil)

	passA := seedIntegrationPass(t, user.ID(), 10)
	passB := seedIntegrationPass(t, user.ID(), 10)
	passC := seedIntegrationPass(t, user.ID(), 10)

	// 1 CONFIRMED + 1 PENDING_TO_CONFIRM: both hold their slot.
	seedIntegrationReservation(t, activity.ID(), dogA.ID(), passA.ID(), domain.StatusConfirmed, now)
	seedIntegrationReservation(t, activity.ID(), dogB.ID(), passB.ID(), domain.StatusPendingToConfirm, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), dogC.ID(), passC.ID(), func() time.Time { return now })
	_, err := uc.Execute(context.Background(), in)
	require.ErrorIs(t, err, ErrActivityFull)

	// No third reservation was created and dogC's pass is untouched.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 2)
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), passC.ID())
	require.NoError(t, err)
	require.Equal(t, 10, gotPass.RemainingSessions())
}

// TestAdminRejectRefundsPass runs the full lifecycle against the real DB:
// a MEDIA conflict creates a PENDING_TO_CONFIRM reservation (one session
// consumed), and the admin reject transitions it to CANCELLED_IN_TIME and
// refunds the consumed session (+1 to the pass balance).
func TestAdminRejectRefundsPass(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "reject-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", []*domain.Incompatibility{trait}, nil)
	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, []*domain.Incompatibility{trigger})
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	// End-to-end: the MEDIA conflict creates the PENDING reservation and
	// consumes one session (10 -> 9).
	registerUC := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out, err := registerUC.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out.Status)

	consumedPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 9, consumedPass.RemainingSessions())

	// The admin rejects the pending reservation.
	rejectUC := newIntegrationRejectUC()
	rejectOut, err := rejectUC.Execute(context.Background(),
		MustNewRejectPendingReservationInput(out.ID, func() time.Time { return now }))
	require.NoError(t, err)
	require.NotNil(t, rejectOut.Reservation)
	require.Equal(t, domain.StatusCancelledInTime, rejectOut.Reservation.Status())

	// DB: the reservation is CANCELLED_IN_TIME...
	persisted, err := postgres.NewReservationRepository(testDB).GetByID(context.Background(), out.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusCancelledInTime, persisted.Status())

	// ...and the pass balance is back to 10 (+1 refund).
	refundedPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 10, refundedPass.RemainingSessions(), "reject must refund the consumed session (+1)")
}
