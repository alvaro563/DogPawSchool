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
	"github.com/stretchr/testify/assert"
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
	activity, err := domain.NewActivity(0, "Paseo Integración", "", "Parque Central",
		domain.TypeRoute, capacity, 1, date, nil, nil)
	require.NoError(t, err)
	repo := postgres.NewActivityRepository(testDB)
	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id, 0, true)
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
		integrationNow, integrationNow, nil, false)
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

func newIntegrationConfirmUC() *ConfirmPendingReservationUseCase {
	return NewConfirmPendingReservationUseCase(
		postgres.NewTransactor(testDB),
		postgres.NewReservationRepository(testDB),
	)
}

// TestAdminConfirmSucceeds runs the full lifecycle against the real DB:
// a MEDIA conflict creates a PENDING_TO_CONFIRM reservation, and the admin
// confirm transitions it to CONFIRMED.
func TestAdminConfirmSucceeds(t *testing.T) {
	cleanTables(t, testDB)

	registerUC := newIntegrationRegisterUC()
	confirmUC := newIntegrationConfirmUC()

	now := integrationNow
	user := seedIntegrationUser(t, "confirm-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", []*domain.Incompatibility{trait}, nil)
	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, []*domain.Incompatibility{trigger})
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out, err := registerUC.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out.Status)

	confirmOut, err := confirmUC.Execute(context.Background(), MustNewConfirmPendingReservationInput(out.ID))
	require.NoError(t, err)
	require.NotNil(t, confirmOut.Reservation)
	require.Equal(t, domain.StatusConfirmed, confirmOut.Reservation.Status())

	persisted, err := postgres.NewReservationRepository(testDB).GetByID(context.Background(), out.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusConfirmed, persisted.Status())
}

// TestNoConflictCreatesConfirmed validates that a booking with no
// incompatibility conflicts is persisted as CONFIRMED immediately
// and the pass session is consumed.
func TestNoConflictCreatesConfirmed(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "noconflict-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, nil)
	pass := seedIntegrationPass(t, user.ID(), 10)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), pass.ID(), func() time.Time { return now })
	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusConfirmed, out.Status)

	reservationRepo := postgres.NewReservationRepository(testDB)
	reservations, err := reservationRepo.ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	require.Equal(t, domain.StatusConfirmed, reservations[0].Status())

	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	require.Equal(t, 9, gotPass.RemainingSessions(), "one session consumed for the booking")
}

// TestListActivityRosterUseCase_Integration verifies the full
// backend pipeline against real Postgres: two owners, three
// reservations (CONFIRMED, PENDING, CANCELLED_LATE). The output
// must partition correctly, resolve the owner names in a single
// user query, and drop the cancelled one.
func TestListActivityRosterUseCase_Integration(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	ownerAna := seedIntegrationUser(t, "ana-roster@test.com")
	ownerJuan := seedIntegrationUser(t, "juan-roster@test.com")
	activity := seedIntegrationActivity(t, 10, now.Add(7*24*time.Hour))

	// Two dogs for Ana (one confirmed, one pending), one dog for
	// Juan (confirmed), one cancelled for Ana (must be dropped).
	dogAna1 := seedIntegrationDog(t, ownerAna.ID(), "Luna", nil, nil)
	dogAna2 := seedIntegrationDog(t, ownerAna.ID(), "Maya", nil, nil)
	dogJuan := seedIntegrationDog(t, ownerJuan.ID(), "Toby", nil, nil)
	dogAna3 := seedIntegrationDog(t, ownerAna.ID(), "Coco", nil, nil)

	passAna := seedIntegrationPass(t, ownerAna.ID(), 10)
	passJuan := seedIntegrationPass(t, ownerJuan.ID(), 10)
	passAna2 := seedIntegrationPass(t, ownerAna.ID(), 10)

	// Confirmed entries: Luna (Ana), Toby (Juan).
	seedIntegrationReservation(t, activity.ID(), dogAna1.ID(), passAna.ID(), domain.StatusConfirmed, now.Add(-2*time.Hour))
	seedIntegrationReservation(t, activity.ID(), dogJuan.ID(), passJuan.ID(), domain.StatusConfirmed, now.Add(-1*time.Hour))
	// Pending entry: Maya (Ana).
	seedIntegrationReservation(t, activity.ID(), dogAna2.ID(), passAna2.ID(), domain.StatusPendingToConfirm, now.Add(-30*time.Minute))
	// Dropped entry: Coco (Ana) cancelled late.
	seedIntegrationReservation(t, activity.ID(), dogAna3.ID(), passAna.ID(), domain.StatusCancelledLate, now.Add(-15*time.Minute))

	uc := NewListActivityRosterUseCase(
		postgres.NewActivityRepository(testDB),
		postgres.NewReservationRepository(testDB),
		postgres.NewUserRepository(testDB),
	)
	out, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(activity.ID()))
	require.NoError(t, err)
	require.NotNil(t, out.Activity)
	assert.Equal(t, activity.ID(), out.Activity.ID())

	require.Len(t, out.Confirmed, 2, "two CONFIRMED")
	// Order is created_at ASC: Luna (older) then Toby.
	assert.Equal(t, dogAna1.ID(), out.Confirmed[0].DogID())
	assert.Equal(t, "Luna", out.Confirmed[0].DogName())
	assert.Equal(t, ownerAna.ID(), out.Confirmed[0].OwnerID())
	assert.Equal(t, ownerAna.Name(), out.Confirmed[0].OwnerName())
	assert.Equal(t, dogJuan.ID(), out.Confirmed[1].DogID())
	assert.Equal(t, "Toby", out.Confirmed[1].DogName())
	assert.Equal(t, ownerJuan.ID(), out.Confirmed[1].OwnerID())
	assert.Equal(t, ownerJuan.Name(), out.Confirmed[1].OwnerName())

	require.Len(t, out.Pending, 1, "one PENDING_TO_CONFIRM")
	assert.Equal(t, dogAna2.ID(), out.Pending[0].DogID())
	assert.Equal(t, "Maya", out.Pending[0].DogName())
	assert.Equal(t, ownerAna.ID(), out.Pending[0].OwnerID())
	assert.Equal(t, ownerAna.Name(), out.Pending[0].OwnerName())

	// Negative path: invalid activity id returns ErrInvalidActivity.
	_, err = uc.Execute(context.Background(), MustNewListActivityRosterInput(99999))
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

// ── Sex/neutered and special-condition integration tests ──────────

// seedIntegrationMaleDog creates a male dog owned by userID with the
// given neutered state and (optionally) has_special_condition=true.
// Uses the real Postgres repository so the column round-trip
// (including has_special_condition) is exercised end-to-end.
func seedIntegrationMaleDog(t *testing.T, userID int, name string, neutered, hasSpecialCondition bool) *domain.Dog {
	t.Helper()
	dog, err := domain.NewDog(0, name, "Labrador", "ES-INTM-"+strings.ToUpper(name), 24,
		domain.SexMale, 12.0, userID)
	require.NoError(t, err)
	if neutered {
		dog.SetNeutered(true)
	}
	repo := postgres.NewDogRepository(testDB)
	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	if hasSpecialCondition {
		require.NoError(t, got.ApplyPatch(domain.DogPatch{HasSpecialCondition: boolPtr(true)}))
		require.NoError(t, repo.Update(context.Background(), got))
	}
	// Re-fetch to verify the patch round-trip.
	got, err = repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	return got
}

func boolPtr(v bool) *bool { return &v }

// TestSexNeutered_IntactVsIntactBlocksAndDoesNotConsumePass validates
// end-to-end that two intact males in the same activity block the
// registration, leave the DB without a new reservation row, and do not
// consume one session from the candidate's pass.
func TestSexNeutered_IntactVsIntactBlocksAndDoesNotConsumePass(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "sn-block-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	// Existing slot holder: intact male (Rex). Candidate: intact male (Luna).
	existing := seedIntegrationMaleDog(t, user.ID(), "Rex", false, false)
	candidate := seedIntegrationMaleDog(t, user.ID(), "Luna", false, false)
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), existing.ID(), classPass.ID(), domain.StatusConfirmed, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	_, err := uc.Execute(context.Background(), in)
	require.Error(t, err)
	var sexErr *SexNeuteredConflictError
	require.True(t, errors.As(err, &sexErr))
	require.NotNil(t, sexErr.IncomingDog)
	assert.Equal(t, candidate.ID(), sexErr.IncomingDog.ID())
	require.Len(t, sexErr.BlockingDogs, 1)
	assert.Equal(t, existing.ID(), sexErr.BlockingDogs[0].ID())

	// No second reservation was created.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	assert.Len(t, reservations, 1, "no reservation may be created when two intact males conflict")

	// Pass session must NOT have been consumed.
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	assert.Equal(t, 10, gotPass.RemainingSessions())
}

// TestSexNeutered_IntactVsCastratedCreatesPending validates that a
// male+intact candidate with a castrated male already in the class
// lands at StatusPendingToConfirm with the sex/neutered reason
// surfaced in the output, persists the reservation, and consumes one
// session (the slot is held).
func TestSexNeutered_IntactVsCastratedCreatesPending(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "sn-pending-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	existing := seedIntegrationMaleDog(t, user.ID(), "Max", true, false)   // castrated male
	candidate := seedIntegrationMaleDog(t, user.ID(), "Toby", false, false) // intact male
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), existing.ID(), classPass.ID(), domain.StatusConfirmed, now)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out.Status)
	require.Len(t, out.PendingReasons, 1)
	assert.Equal(t, SexNeuteredReasonPrefix+domain.ReasonIntactVsCastrated, out.PendingReasons[0].Code)
	assert.Equal(t, []int{candidate.ID(), existing.ID()}, out.PendingReasons[0].DogIDs)

	// Reservation persisted in the DB with PENDING_TO_CONFIRM.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	assert.Len(t, reservations, 2)
	var pending *domain.Reservation
	for _, r := range reservations {
		if r.DogID() == candidate.ID() {
			pending = r
		}
	}
	require.NotNil(t, pending)
	assert.Equal(t, domain.StatusPendingToConfirm, pending.Status())

	// One session consumed (slot held).
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	assert.Equal(t, 9, gotPass.RemainingSessions())
}

// TestHasSpecialConditionIntegration_CreatesPendingWithReason
// validates that a dog carrying has_special_condition=true lands at
// StatusPendingToConfirm with the special-condition reason, even
// when there are no other conflicts.
func TestHasSpecialConditionIntegration_CreatesPendingWithReason(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "hsc-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	candidate := seedIntegrationMaleDog(t, user.ID(), "Luna", true, true)
	// No existing slot holder — empty class. Sanity check: the only
	// reason the candidate becomes pending is the special-condition
	// flag itself.
	pass := seedIntegrationPass(t, user.ID(), 10)

	uc := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), pass.ID(), func() time.Time { return now })
	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out.Status)
	require.Len(t, out.PendingReasons, 1)
	assert.Equal(t, domain.ReasonHasSpecialCondition, out.PendingReasons[0].Code)
	assert.Equal(t, []int{candidate.ID()}, out.PendingReasons[0].DogIDs)

	// Reservation persisted in the DB with PENDING_TO_CONFIRM.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	assert.Equal(t, domain.StatusPendingToConfirm, reservations[0].Status())

	// One session consumed (slot held).
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	assert.Equal(t, 9, gotPass.RemainingSessions())
}

// ── Re-registration after cancel/reject (regression for partial
//    unique index 000016) ───────────────────────────────────────────

// newIntegrationCancelUC wires CancelReservationUseCase with the real
// Postgres repositories and transactor.
func newIntegrationCancelUC() *CancelReservationUseCase {
	return NewCancelReservationUseCase(
		postgres.NewTransactor(testDB),
		postgres.NewActivityRepository(testDB),
		postgres.NewDogRepository(testDB),
		postgres.NewPassRepository(testDB),
		postgres.NewReservationRepository(testDB),
	)
}

// TestCancelInTimeAllowsReregistration verifies that after a CONFIRMED
// reservation is cancelled in time, the same dog can be registered to
// the same activity again. The partial unique index
// uniq_reservation_dog_active excludes CANCELLED_IN_TIME, so the new
// INSERT succeeds. The pass session is refunded on cancel (10 → 9 →
// 8) and re-consumed on the new booking (8 → 7).
func TestCancelInTimeAllowsReregistration(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "reregcancel-intime@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))
	dog := seedIntegrationDog(t, user.ID(), "Luna", nil, nil)
	pass := seedIntegrationPass(t, user.ID(), 10)

	registerUC := newIntegrationRegisterUC()
	cancelUC := newIntegrationCancelUC()

	// First booking — CONFIRMED.
	in1 := MustNewRegisterReservationInput(user.ID(), activity.ID(), dog.ID(), pass.ID(), func() time.Time { return now })
	out1, err := registerUC.Execute(context.Background(), in1)
	require.NoError(t, err)
	require.Equal(t, domain.StatusConfirmed, out1.Status)

	// Cancel in time (admin path so we don't need ownership wiring).
	cancelIn := MustNewCancelReservationAdminInput(out1.ID, func() time.Time { return now })
	_, err = cancelUC.Execute(context.Background(), cancelIn)
	require.NoError(t, err)

	// Pass refunded to 10.
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	require.Equal(t, 10, gotPass.RemainingSessions(), "cancel in time must refund the session")

	// The bug case: re-register the same dog. Pre-fix this would fail
	// with ErrDuplicateReservationForDog; post-fix it succeeds.
	in2 := MustNewRegisterReservationInput(user.ID(), activity.ID(), dog.ID(), pass.ID(), func() time.Time { return now })
	out2, err := registerUC.Execute(context.Background(), in2)
	require.NoError(t, err, "re-registration after cancel-in-time must succeed")
	require.Equal(t, domain.StatusConfirmed, out2.Status)
	require.NotEqual(t, out1.ID, out2.ID, "the new reservation must have a fresh id")

	// DB: 2 rows for (activity, dog) — 1 CANCELLED_IN_TIME + 1 CONFIRMED.
	reservations, err := postgres.NewReservationRepository(testDB).ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 2)
	statuses := []domain.ReservationStatus{reservations[0].Status(), reservations[1].Status()}
	assert.Contains(t, statuses, domain.StatusCancelledInTime)
	assert.Contains(t, statuses, domain.StatusConfirmed)

	// Only the active one holds the slot (capacity semantics unchanged).
	assert.True(t, reservations[0].HoldsSlot() != reservations[1].HoldsSlot(),
		"exactly one of the two rows must hold the slot")

	// Pass session re-consumed by the new booking: 10 → 9.
	gotPass, err = postgres.NewPassRepository(testDB).GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	assert.Equal(t, 9, gotPass.RemainingSessions())
}

// TestCancelLateAllowsReregistration verifies that a CANCELLED_LATE
// reservation — the terminal status produced when admin cancels a
// past booking without refunding the pass — still frees the slot for
// re-registration. The partial unique index excludes CANCELLED_LATE,
// so a fresh INSERT for the same (activity_id, dog_id) succeeds.
// We bypass the cancel use case (which also blocks past activities)
// and insert CANCELLED_LATE directly via the repository: the test is
// about the DB invariant, not the cancel policy.
func TestCancelLateAllowsReregistration(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "reregcancel-late@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))
	dog := seedIntegrationDog(t, user.ID(), "Toby", nil, nil)
	pass := seedIntegrationPass(t, user.ID(), 10)

	repo := postgres.NewReservationRepository(testDB)
	// Plant a CANCELLED_LATE row directly: the use case would refuse
	// it (no past activity), but the partial-index regression test is
	// specifically about that status not blocking re-registration.
	cancelled, err := domain.NewReservationWithStatus(0, activity.ID(), dog.ID(), pass.ID(), domain.StatusCancelledLate, now)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), cancelled)
	require.NoError(t, err)

	// Re-registering is allowed: the partial index doesn't see CANCELLED_LATE.
	registerUC := newIntegrationRegisterUC()
	in := MustNewRegisterReservationInput(user.ID(), activity.ID(), dog.ID(), pass.ID(), func() time.Time { return now })
	out, err := registerUC.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, domain.StatusConfirmed, out.Status)

	// DB now has 2 rows for (activity, dog) — 1 CANCELLED_LATE + 1 CONFIRMED.
	reservations, err := repo.ListByActivity(context.Background(), activity.ID())
	require.NoError(t, err)
	require.Len(t, reservations, 2)
	statuses := []domain.ReservationStatus{reservations[0].Status(), reservations[1].Status()}
	assert.Contains(t, statuses, domain.StatusCancelledLate)
	assert.Contains(t, statuses, domain.StatusConfirmed)
}

// TestRejectPendingAllowsReregistration validates the full reject +
// re-register loop. After admin rejects a PENDING_TO_CONFIRM
// reservation (refunding the session), the same dog can be registered
// to the same activity again.
func TestRejectPendingAllowsReregistration(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "reregreject@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))

	trait := seedIntegrationTrait(t, "MACHO_ENTERO", "Macho entero (no castrado)")
	trigger := seedIntegrationTrigger(t, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO")

	classDog := seedIntegrationDog(t, user.ID(), "Rex", []*domain.Incompatibility{trait}, nil)
	candidate := seedIntegrationDog(t, user.ID(), "Luna", nil, []*domain.Incompatibility{trigger})
	classPass := seedIntegrationPass(t, user.ID(), 10)
	candidatePass := seedIntegrationPass(t, user.ID(), 10)

	seedIntegrationReservation(t, activity.ID(), classDog.ID(), classPass.ID(), domain.StatusConfirmed, now)

	registerUC := newIntegrationRegisterUC()
	rejectUC := newIntegrationRejectUC()

	// First attempt: MEDIA conflict → PENDING_TO_CONFIRM.
	in1 := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out1, err := registerUC.Execute(context.Background(), in1)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPendingToConfirm, out1.Status)

	// Admin rejects → CANCELLED_IN_TIME, session refunded.
	_, err = rejectUC.Execute(context.Background(), MustNewRejectPendingReservationInput(out1.ID, func() time.Time { return now }))
	require.NoError(t, err)
	gotPass, err := postgres.NewPassRepository(testDB).GetByID(context.Background(), candidatePass.ID())
	require.NoError(t, err)
	require.Equal(t, 10, gotPass.RemainingSessions())

	// Remove the trigger so the next register succeeds CONFIRMED.
	candidate2, err := postgres.NewDogRepository(testDB).GetByID(context.Background(), candidate.ID())
	require.NoError(t, err)
	for _, incomp := range candidate2.Incompatibilities() {
		_, _ = candidate2.RemoveIncompatibility(incomp.ID())
	}
	require.NoError(t, postgres.NewDogRepository(testDB).Update(context.Background(), candidate2))

	// Re-register — succeeds because the partial index excludes
	// CANCELLED_IN_TIME.
	in2 := MustNewRegisterReservationInput(user.ID(), activity.ID(), candidate.ID(), candidatePass.ID(), func() time.Time { return now })
	out2, err := registerUC.Execute(context.Background(), in2)
	require.NoError(t, err, "re-register after admin reject must succeed")
	require.Equal(t, domain.StatusConfirmed, out2.Status)
	require.NotEqual(t, out1.ID, out2.ID)
}

// TestPartialUniqueIndexBlocksTrueDuplicate confirms that the
// partial unique index still does its job: two CONFIRMED reservations
// for the same (activity_id, dog_id) cannot coexist. We bypass the use
// case (which pre-checks via ListByActivity) and insert a duplicate
// directly via the repository to simulate a concurrent race that
// slipped through the pre-check.
func TestPartialUniqueIndexBlocksTrueDuplicate(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "reregblock-true-dup@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))
	dog := seedIntegrationDog(t, user.ID(), "Luna", nil, nil)
	pass := seedIntegrationPass(t, user.ID(), 10)

	repo := postgres.NewReservationRepository(testDB)
	first, err := domain.NewReservationWithStatus(0, activity.ID(), dog.ID(), pass.ID(), domain.StatusConfirmed, now)
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), first)
	require.NoError(t, err)
	require.Positive(t, id)

	// Attempt a second CONFIRMED for the same (activity, dog): the
	// partial unique index must reject it.
	second, err := domain.NewReservationWithStatus(0, activity.ID(), dog.ID(), pass.ID(), domain.StatusConfirmed, now)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), second)
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrDuplicateReservation)
}

// ── Forgive end-to-end (regression for the admin "Canceladas tarde"
//    section) ─────────────────────────────────────────────────────

func newIntegrationForgiveUC() *ForgiveReservationUseCase {
	return NewForgiveReservationUseCase(
		postgres.NewTransactor(testDB),
		postgres.NewPassRepository(testDB),
		postgres.NewReservationRepository(testDB),
	)
}

// TestForgiveIntegration_SuccessRefundsPass runs the full late-cancel
// + admin forgive cycle against Postgres: a CANCELLED_LATE
// reservation is forgiven; the pass session is refunded and the
// status moves to FORGIVEN. Plant the CANCELLED_LATE row directly to
// bypass the cancel use case, which would also block past activities.
func TestForgiveIntegration_SuccessRefundsPass(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "forgive-owner@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))
	dog := seedIntegrationDog(t, user.ID(), "Luna", nil, nil)

	// Set up the realistic state: one session already consumed by
	// the original (now late-cancelled) booking.
	pass := seedIntegrationPass(t, user.ID(), 10)
	passRepo := postgres.NewPassRepository(testDB)
	loaded, err := passRepo.GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	_, err = loaded.ConsumeSession("test-consume", now)
	require.NoError(t, err)
	require.NoError(t, passRepo.Update(context.Background(), loaded))
	gotPass, err := passRepo.GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	require.Equal(t, 9, gotPass.RemainingSessions(), "precondition: 1 session consumed by the original booking")

	// Plant the CANCELLED_LATE row directly (the cancel use case
	// would also block past activities).
	reservationRepo := postgres.NewReservationRepository(testDB)
	cancelled, err := domain.NewReservationWithStatus(0, activity.ID(), dog.ID(), pass.ID(), domain.StatusCancelledLate, now)
	require.NoError(t, err)
	cancelledID, err := reservationRepo.Create(context.Background(), cancelled)
	require.NoError(t, err)
	require.Positive(t, cancelledID)

	// Admin forgives.
	forgiveUC := newIntegrationForgiveUC()
	out, err := forgiveUC.Execute(context.Background(),
		MustNewForgiveReservationInput(cancelledID, func() time.Time { return now }))
	require.NoError(t, err)
	assert.True(t, out.PassSessionRefunded, "pass has consumed sessions → refundable")
	assert.Equal(t, domain.StatusForgiven, out.Reservation.Status())

	// Persisted reservation has transitioned to FORGIVEN.
	persisted, err := reservationRepo.GetByID(context.Background(), cancelledID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusForgiven, persisted.Status())

	// Pass session refunded: 9 → 10.
	gotPass, err = passRepo.GetByID(context.Background(), pass.ID())
	require.NoError(t, err)
	assert.Equal(t, 10, gotPass.RemainingSessions())
}

// TestForgiveIntegration_NotLateCancelledReturnsErrNotLateCancelled
// verifies that forgiving a CONFIRMED reservation (or any other
// non-CANCELLED_LATE status) is rejected with the sentinel that maps
// to 409 not_late_cancelled on the wire.
func TestForgiveIntegration_NotLateCancelledReturnsErrNotLateCancelled(t *testing.T) {
	cleanTables(t, testDB)

	now := integrationNow
	user := seedIntegrationUser(t, "forgive-notlate@test.com")
	activity := seedIntegrationActivity(t, 5, now.Add(7*24*time.Hour))
	dog := seedIntegrationDog(t, user.ID(), "Toby", nil, nil)
	pass := seedIntegrationPass(t, user.ID(), 10)

	confirmed := mustNewReservation(0, activity.ID(), dog.ID(), pass.ID(), domain.StatusConfirmed, now)
	reservationRepo := postgres.NewReservationRepository(testDB)
	id, err := reservationRepo.Create(context.Background(), confirmed)
	require.NoError(t, err)

	forgiveUC := newIntegrationForgiveUC()
	_, err = forgiveUC.Execute(context.Background(),
		MustNewForgiveReservationInput(id, func() time.Time { return now }))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotLateCancelled),
		"expected ErrNotLateCancelled, got %T: %v", err, err)
}
