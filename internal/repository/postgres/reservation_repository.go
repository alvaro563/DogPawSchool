package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
)

var (
	// ErrReservationNotFound is returned by GetByID/Update when no row
	// matches the id. Mirrors the dog / activity / pass sentinels.
	ErrReservationNotFound = errors.New("postgres: reservation not found")

	// ErrInvalidReservationActivity is returned by Create when the
	// activity_id foreign key does not resolve to an existing activity.
	ErrInvalidReservationActivity = errors.New("postgres: reservation activity_id does not exist")

	// ErrInvalidReservationDog is returned by Create when the dog_id
	// foreign key does not resolve to an existing dog.
	ErrInvalidReservationDog = errors.New("postgres: reservation dog_id does not exist")

	// ErrInvalidReservationPass is returned by Create when the pass_id
	// foreign key does not resolve to an existing pass.
	ErrInvalidReservationPass = errors.New("postgres: reservation pass_id does not exist")

	// ErrDuplicateReservation is returned by Create when the
	// UNIQUE (activity_id, dog_id) constraint is violated. This
	// enforces the "one dog per activity" business rule at the DB
	// level: the same dog cannot be booked twice into the same
	// activity, even if the first booking was later cancelled.
	ErrDuplicateReservation = errors.New("postgres: dog already booked for this activity")
)

// reservationSelectClause is the 6-column projection reused by every
// read method. The updated_at column is intentionally omitted because
// the domain entity does not expose it yet. Keep the column order in
// lockstep with scanReservation.
const reservationSelectClause = `SELECT id, activity_id, dog_id, pass_id, status, created_at
	FROM reservations`

type ReservationRepository struct {
	db *sql.DB
}

func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

// Create inserts a new reservation in StatusConfirmed (the domain
// constructor forces that initial state; this method never lets a
// different status through). Returns the assigned id. Foreign-key
// violations on activity_id, dog_id, and pass_id are mapped to
// dedicated sentinels; the UNIQUE (activity_id, dog_id) constraint
// is mapped to ErrDuplicateReservation so the use case can surface a
// 409.
func (repo *ReservationRepository) Create(ctx context.Context, reservation *domain.Reservation) (int, error) {
	const query = `
		INSERT INTO reservations (activity_id, dog_id, pass_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var newReservationID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		reservation.ActivityID(), reservation.DogID(), reservation.PassID(),
		string(reservation.Status()), reservation.CreatedAt(),
	).Scan(&newReservationID)
	if err != nil {
		return 0, mapReservationCreateError(err)
	}
	return int(newReservationID), nil
}

// GetByID fetches a single reservation by id. Returns
// ErrReservationNotFound when no row matches.
func (repo *ReservationRepository) GetByID(ctx context.Context, id int) (*domain.Reservation, error) {
	query := reservationSelectClause + ` WHERE id = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	reservation, err := scanReservation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}
	return reservation, nil
}

// Update writes the mutable fields of a reservation. The only field
// that changes over a reservation's lifetime is its status (e.g.,
// Confirmed → CancelledLate → Forgiven), so this update is narrow on
// purpose. Returns ErrReservationNotFound if no row matches.
func (repo *ReservationRepository) Update(ctx context.Context, reservation *domain.Reservation) error {
	const query = `
		UPDATE reservations
		SET status = $1
		WHERE id = $2
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, string(reservation.Status()), reservation.ID())
	if err != nil {
		return fmt.Errorf("update reservation: %w", err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update reservation: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrReservationNotFound
	}
	return nil
}

// ListByActivity returns every reservation for the given activity,
// regardless of status. The RegisterReservationUseCase filters for
// CONFIRMED in Go when computing the remaining capacity; this keeps
// the query simple and lets the same method serve future reporting
// use cases (e.g., "all bookings for this activity, including
// cancellations").
func (repo *ReservationRepository) ListByActivity(ctx context.Context, activityID int) ([]*domain.Reservation, error) {
	query := reservationSelectClause + `
		WHERE activity_id = $1
		ORDER BY created_at ASC`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, activityID)
	if err != nil {
		return nil, fmt.Errorf("query reservations by activity: %w", err)
	}
	defer rows.Close()

	reservations := make([]*domain.Reservation, 0)
	for rows.Next() {
		reservation, err := scanReservation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return reservations, nil
}

// ListByDog returns every reservation for the given dog, ordered
// most recent first. Used for "my bookings" views.
func (repo *ReservationRepository) ListByDog(ctx context.Context, dogID int) ([]*domain.Reservation, error) {
	query := reservationSelectClause + `
		WHERE dog_id = $1
		ORDER BY created_at DESC`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, dogID)
	if err != nil {
		return nil, fmt.Errorf("query reservations by dog: %w", err)
	}
	defer rows.Close()

	reservations := make([]*domain.Reservation, 0)
	for rows.Next() {
		reservation, err := scanReservation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return reservations, nil
}

// ListByPass returns every reservation paid from the given pass,
// ordered most recent first. Used for the pass audit view.
func (repo *ReservationRepository) ListByPass(ctx context.Context, passID int) ([]*domain.Reservation, error) {
	query := reservationSelectClause + `
		WHERE pass_id = $1
		ORDER BY created_at DESC`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, passID)
	if err != nil {
		return nil, fmt.Errorf("query reservations by pass: %w", err)
	}
	defer rows.Close()

	reservations := make([]*domain.Reservation, 0)
	for rows.Next() {
		reservation, err := scanReservation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return reservations, nil
}

// reservationScanner is the minimal interface satisfied by both
// *sql.Row and *sql.Rows. Used by scanReservation to share the
// column ordering across all read methods.
type reservationScanner interface {
	Scan(dest ...any) error
}

// scanReservation reads one reservation row. The column order MUST
// match reservationSelectClause. The reconstruction uses
// NewReservationWithStatus so the resulting domain object reflects
// whatever status the row currently holds (it is not always
// StatusConfirmed; later states such as CANCELLED_LATE and FORGIVEN
// are valid).
func scanReservation(row reservationScanner) (*domain.Reservation, error) {
	var (
		reservationID int
		activityID    int
		dogID         int
		passID        int
		status        string
		createdAt     time.Time
	)
	if err := row.Scan(
		&reservationID, &activityID, &dogID, &passID, &status, &createdAt,
	); err != nil {
		return nil, err
	}
	return domain.NewReservationWithStatus(
		reservationID, activityID, dogID, passID,
		domain.ReservationStatus(status), createdAt,
	)
}

func mapReservationCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrForeignKeyViolation:
			// We can't tell from the SQLSTATE alone which FK was
			// violated, so the use case is responsible for
			// checking activity/dog/pass existence BEFORE the
			// insert. This branch is a safety net for race
			// conditions (e.g., the dog is deleted between the
			// GetByID and the Create). We default to the most
			// likely cause in a Create flow: the activity was
			// deleted. The handler maps any of these three to a
			// 400 invalid_<x>_id; the specific mapping for a
			// race condition is best-effort.
			return mapReservationForeignKeyError(pgErr)
		case pgErrUniqueViolation:
			return ErrDuplicateReservation
		case pgErrCheckViolation:
			return fmt.Errorf("create reservation: %w", err)
		}
	}
	return fmt.Errorf("create reservation: %w", err)
}

// mapReservationForeignKeyError attempts to disambiguate which FK
// triggered the violation. pgErr.ColumnName or pgErr.ConstraintName
// is not reliably populated for FK errors in Postgres, so we fall
// back to the FK order: activity_id, dog_id, pass_id. The
// reservations table has FKs in that order (see migration
// 000001_initial_schema.up.sql), so the first constraint that
// matches wins. If none match, the FK error is returned as-is so
// the use case can still see it.
func mapReservationForeignKeyError(pgErr *pgconn.PgError) error {
	switch pgErr.ConstraintName {
	case "fk_reservations_activity":
		return ErrInvalidReservationActivity
	case "fk_reservations_dog":
		return ErrInvalidReservationDog
	case "fk_reservations_pass":
		return ErrInvalidReservationPass
	}
	return fmt.Errorf("create reservation (FK): %w", pgErr)
}

// ============================================================================
// View methods (read paths) — return denormalized *domain.ReservationView
// ============================================================================
//
// The 6 view methods all use a single 3-table JOIN
// (reservations × activities × dogs × passes) and the same
// scanReservationView function. The 16-column projection is kept
// stable; if you change it you must update both the SELECT clauses
// and the scanReservationView function in lockstep.

// reservationViewSelectClause is the 24-column projection reused by
// every read-view method. Keep the column order in lockstep with
// scanReservationView. The clause includes the full SELECT for
// reservations, activities, dogs, AND passes; the caller is
// responsible for appending the WHERE / ORDER BY / LIMIT / OFFSET
// plus the JOIN ON passes (the FROM/JOIN dogs is already included
// here so callers only need `JOIN passes p ON p.id = r.pass_id
// WHERE ...`).
const reservationViewSelectClause = `SELECT
		r.id, r.status, r.created_at,
		a.id, a.name, a.activity_type, a.max_capacity, a.location, a.duration_in_hours, a.date,
		d.id, d.user_id, d.name, d.breed, d.passport,
		p.id, p.num_of_sessions, p.remaining_sessions, p.price,
		p.pass_type, p.created_at, p.updated_at, p.expires_at, p.user_id
	FROM reservations r
	JOIN activities a ON a.id = r.activity_id
	JOIN dogs d ON d.id = r.dog_id`

// GetView returns the denormalized view for a single reservation id.
// Returns ErrReservationNotFound when no row matches.
func (repo *ReservationRepository) GetView(ctx context.Context, id int) (*domain.ReservationView, error) {
	query := reservationViewSelectClause + " JOIN passes p ON p.id = r.pass_id WHERE r.id = $1"
	return queryOneReservationView(ctx, runner(ctx, repo.db), query, id)
}

// ListByUserView returns the views of every reservation whose dog
// belongs to userID, with optional filters on status and
// created_at. status, from, to are nullable: pass nil to skip that
// filter. limit / offset must be normalized by the caller.
func (repo *ReservationRepository) ListByUserView(
	ctx context.Context,
	userID int,
	status *domain.ReservationStatus,
	from, to *time.Time,
	limit, offset int,
) ([]*domain.ReservationView, error) {
	query := reservationViewSelectClause + `
		JOIN passes p ON p.id = r.pass_id
		WHERE d.user_id = $1
		  AND ($2::reservation_status IS NULL OR r.status = $2)
		  AND ($3::timestamptz IS NULL OR r.created_at >= $3)
		  AND ($4::timestamptz IS NULL OR r.created_at <  $4)
		ORDER BY r.created_at DESC
		LIMIT $5 OFFSET $6`
	return queryReservationViews(ctx, runner(ctx, repo.db), query, userID, nullableStatus(status), nullableTime(from), nullableTime(to), limit, offset)
}

// ListByUserUpcomingView returns the views of every CONFIRMED
// reservation whose activity date is at or after the current time,
// ordered by activity date ASC. limit / offset are normalized by the
// caller.
func (repo *ReservationRepository) ListByUserUpcomingView(ctx context.Context, userID, limit, offset int) ([]*domain.ReservationView, error) {
	query := reservationViewSelectClause + `
		JOIN passes p ON p.id = r.pass_id
		WHERE d.user_id = $1
		  AND r.status = 'CONFIRMED'
		  AND a.date >= NOW()
		ORDER BY a.date ASC
		LIMIT $2 OFFSET $3`
	return queryReservationViews(ctx, runner(ctx, repo.db), query, userID, limit, offset)
}

// ListByDogView returns every reservation for a given dog, most
// recent first.
func (repo *ReservationRepository) ListByDogView(ctx context.Context, dogID, limit, offset int) ([]*domain.ReservationView, error) {
	query := reservationViewSelectClause + `
		JOIN passes p ON p.id = r.pass_id
		WHERE r.dog_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`
	return queryReservationViews(ctx, runner(ctx, repo.db), query, dogID, limit, offset)
}

// ListByPassView returns every reservation paid from a given pass,
// most recent first.
func (repo *ReservationRepository) ListByPassView(ctx context.Context, passID, limit, offset int) ([]*domain.ReservationView, error) {
	query := reservationViewSelectClause + `
		JOIN passes p ON p.id = r.pass_id
		WHERE r.pass_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`
	return queryReservationViews(ctx, runner(ctx, repo.db), query, passID, limit, offset)
}

// ListByActivityView returns every reservation for a given
// activity, ordered by created_at ASC (chronological order is the
// most useful for a class roster).
func (repo *ReservationRepository) ListByActivityView(ctx context.Context, activityID, limit, offset int) ([]*domain.ReservationView, error) {
	query := reservationViewSelectClause + `
		JOIN passes p ON p.id = r.pass_id
		WHERE r.activity_id = $1
		ORDER BY r.created_at ASC
		LIMIT $2 OFFSET $3`
	return queryReservationViews(ctx, runner(ctx, repo.db), query, activityID, limit, offset)
}

// queryOneReservationView runs a single-row query and returns a
// *domain.ReservationView. The query must return all 16 columns of
// reservationViewSelectClause plus 4 pass columns. Used by GetView.
func queryOneReservationView(ctx context.Context, runner dbRunner, query string, args ...any) (*domain.ReservationView, error) {
	rows, err := runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query reservation view: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("rows err: %w", err)
		}
		return nil, ErrReservationNotFound
	}
	return scanReservationView(rows)
}

// queryReservationViews runs a multi-row query and returns the slice
// of *domain.ReservationView. The query must return all columns
// required by scanReservationView.
func queryReservationViews(ctx context.Context, runner dbRunner, query string, args ...any) ([]*domain.ReservationView, error) {
	rows, err := runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query reservation views: %w", err)
	}
	defer rows.Close()

	views := make([]*domain.ReservationView, 0)
	for rows.Next() {
		view, err := scanReservationView(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reservation view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return views, nil
}

// nullableStatus converts a *ReservationStatus to the string form
// expected by the optional-filter SQL pattern ($N::text IS NULL OR
// r.status = $N). nil → nil.
func nullableStatus(status *domain.ReservationStatus) any {
	if status == nil {
		return nil
	}
	return string(*status)
}

// nullableTime converts a *time.Time to the form expected by the
// optional-filter SQL pattern ($N::timestamptz IS NULL OR ...).
// nil → nil.
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// scanReservationView reads one full joined row. The column order
// MUST match reservationViewSelectClause plus the pass columns that
// each calling query appends.
//
// The reconstruction strategy is the most direct one: read into
// local variables, build the bare aggregates (Reservation, Activity,
// Dog, Pass) via their constructors, and then assemble them into a
// ReservationView via NewReservationView. If any constructor
// refuses the values, we return the error so a buggy JOIN is
// surfaced immediately.
func scanReservationView(row reservationScanner) (*domain.ReservationView, error) {
	var (
		// reservation (3 cols)
		reservationID int
		status        string
		createdAt     time.Time

		// activity (7 cols)
		activityID       int
		activityName     string
		activityType     string
		maxCapacity      int
		activityLocation string
		durationInHours  int
		activityDate     time.Time

		// dog (5 cols)
		dogID     int
		dogUserID int
		dogName   string
		dogBreed  string
		passport  string

		// pass (5 cols)
		passID            int
		numOfSessions     int
		remainingSessions int
		price             int
		passType          string
		passCreatedAt     time.Time
		passUpdatedAt     time.Time
		passExpiresAt     sql.NullTime
		passUserID        int
	)
	if err := row.Scan(
		&reservationID, &status, &createdAt,
		&activityID, &activityName, &activityType, &maxCapacity, &activityLocation, &durationInHours, &activityDate,
		&dogID, &dogUserID, &dogName, &dogBreed, &passport,
		&passID, &numOfSessions, &remainingSessions, &price,
		&passType, &passCreatedAt, &passUpdatedAt, &passExpiresAt, &passUserID,
	); err != nil {
		return nil, err
	}

	reservation, err := domain.NewReservationWithStatus(
		reservationID, activityID, dogID, passID,
		domain.ReservationStatus(status), createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct reservation: %w", err)
	}
	activity, err := domain.NewActivity(
		activityID, activityName, activityLocation,
		domain.ActivityType(activityType), maxCapacity, durationInHours, activityDate,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct activity: %w", err)
	}
	dog, err := domain.NewDog(
		dogID, dogName, dogBreed, passport,
		// Use zero age/weight; the read model does not need them
		// and NewDog only requires id > 0 and userID > 0 here.
		0, domain.SexMale, 0, dogUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct dog: %w", err)
	}
	var passExpiresAtPtr *time.Time
	if passExpiresAt.Valid {
		passExpiresAtPtr = &passExpiresAt.Time
	}
	pass, err := domain.NewPass(
		passID, numOfSessions, remainingSessions, price, domain.PassType(passType),
		passUserID, passCreatedAt, passUpdatedAt, passExpiresAtPtr,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct pass: %w", err)
	}
	return domain.NewReservationView(reservation, activity, dog, pass)
}

var _ domain.ReservationRepository = (*ReservationRepository)(nil)
