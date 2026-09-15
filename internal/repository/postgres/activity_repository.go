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

// ErrActivityNotFound aliases the domain not-found sentinel so callers
// match it via errors.Is(err, domain.ErrNotFound) without importing this
// package.
var (
	ErrActivityNotFound = domain.ErrNotFound
)

// activitySelectClause is the projection reused by every read method,
// sharing the LEFT JOIN to dogs so the visibility filter can reference
// d.user_id uniformly across List / ListByDateRange / ListByClosed /
// ListUpcoming / GetByID. The 10-column SELECT reads only activity
// fields; d.user_id is referenced exclusively in the WHERE.
// Keep the column order in lockstep with scanActivity.
//
// Aliases 'a' (activities) and 'd' (dogs) are required so that the
// visibility predicate (a.dog_id IS NULL OR d.user_id = $N) and any
// downstream predicates can refer unambiguously to both tables.
const activitySelectClause = `SELECT a.id, a.name, a.description, a.activity_type, a.max_capacity,
           a.location, a.duration_in_hours, a.date, a.closed, a.dog_id
    FROM activities a
    LEFT JOIN dogs d ON a.dog_id = d.id`

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// visibilityPredicate returns the SQL fragment that gates activities
// from the perspective of (viewerUserID, viewerIsAdmin). Admin
// viewers see every row (empty fragment → no predicate). Non-admin
// users see group activities (a.dog_id IS NULL — user_id on the LEFT
// JOIN is NULL too) and individual activities for dogs they own (d.user_id
// = viewerUserID). The fragment is intentionally written with literals
// (no params) to avoid renumbering $N in the calling SQL strings —
// passing the values via fmt.Sprintf is safe because both inputs are
// int coming from session auth context, not user-controlled strings.
//
// Returns "" (empty string) when the caller is admin so the calling
// query doesn't append anything between WHERE clauses.
func visibilityPredicate(viewerUserID int, viewerIsAdmin bool) string {
	if viewerIsAdmin {
		return ""
	}
	return fmt.Sprintf(" AND (a.dog_id IS NULL OR d.user_id = %d)", viewerUserID)
}

// Create inserts a new activity and returns the assigned id. dog_id is
// nullable; for INDIVIDUAL_CLASS it MUST be set (the domain layer
// enforces that, and the row-level CHECK constraint in migration
// 000013 is the last line of defence).
func (repo *ActivityRepository) Create(ctx context.Context, activity *domain.Activity) (int, error) {
	const query = `
		INSERT INTO activities (
			name, description, activity_type, max_capacity,
			location, duration_in_hours, date, dog_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var newActivityID int64
	var dogIDArg any
	if d := activity.DogID(); d != nil {
		dogIDArg = *d
	}
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		activity.Name(), activity.Description(), string(activity.Type()), activity.MaxCapacity(),
		activity.Location(), activity.DurationInHours(), activity.Date(),
		dogIDArg,
	).Scan(&newActivityID)
	if err != nil {
		return 0, mapActivityCreateError(err)
	}
	return int(newActivityID), nil
}

// GetByID fetches a single activity by id. Applies the visibility
// filter so non-admin viewers cannot probe other users' individual
// classes via direct id lookup — they receive ErrActivityNotFound
// instead (no existence leak).
func (repo *ActivityRepository) GetByID(ctx context.Context, id int, viewerUserID int, viewerIsAdmin bool) (*domain.Activity, error) {
	query := activitySelectClause + ` WHERE a.id = $1` + visibilityPredicate(viewerUserID, viewerIsAdmin)
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	activity, err := scanActivity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	return activity, nil
}

// GetByIDForUpdate fetches a single activity by id and locks the row
// with FOR UPDATE OF a until the transaction commits. Returns
// ErrActivityNotFound when no row matches. This is an admin-only
// path (used inside the closure / bulk-complete use cases which
// run with the admin's authority), but for defence in depth it also
// receives the visibility params and gets the same filtering.
//
// FOR UPDATE OF a is required: with the LEFT JOIN to dogs, plain
// FOR UPDATE would try to lock rows on the joined side too, which
// Postgres does not allow in some versions (SQLSTATE 0A000 — lock
// of FOR UPDATE/BY SHARE on outer-join nullable side is rejected).
// Qualifying by the activities alias (`a`) locks only the activity
// row, which is what the test and use cases expect.
func (repo *ActivityRepository) GetByIDForUpdate(ctx context.Context, id int, viewerUserID int, viewerIsAdmin bool) (*domain.Activity, error) {
	query := activitySelectClause + ` WHERE a.id = $1 FOR UPDATE OF a` + visibilityPredicate(viewerUserID, viewerIsAdmin)
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	activity, err := scanActivity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	return activity, nil
}

// Update writes all mutable fields of the activity, including
// dog_id. Returns ErrActivityNotFound if no row matches the id.
func (repo *ActivityRepository) Update(ctx context.Context, activity *domain.Activity) error {
	const query = `
		UPDATE activities
		SET name = $1, description = $2, activity_type = $3, max_capacity = $4,
		    location = $5, duration_in_hours = $6, date = $7, closed = $8, dog_id = $9
		WHERE id = $10
	`
	var dogIDArg any
	if d := activity.DogID(); d != nil {
		dogIDArg = *d
	}
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		activity.Name(), activity.Description(), string(activity.Type()), activity.MaxCapacity(),
		activity.Location(), activity.DurationInHours(), activity.Date(),
		activity.IsClosed(),
		dogIDArg,
		activity.ID(),
	)
	if err != nil {
		return mapActivityUpdateError(err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update activity: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// Delete removes an activity by id. Currently no use case invokes it
// (DeleteActivityUseCase is deferred until cross-aggregate cancellation
// and refund logic is designed). It is implemented here so the
// interface assertion compiles and the method is ready for use.
func (repo *ActivityRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM activities WHERE id = $1`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete activity: %w", err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete activity: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// List returns a paginated list of activities (with the visibility
// filter applied when the caller is not an admin). Ordered by date ASC.
func (repo *ActivityRepository) List(ctx context.Context, viewerUserID int, viewerIsAdmin bool, limit, offset int) ([]*domain.Activity, error) {
	query := activitySelectClause + visibilityPredicate(viewerUserID, viewerIsAdmin) + `
		ORDER BY a.date ASC
		LIMIT $1 OFFSET $2`
	return repo.queryActivities(ctx, query, limit, offset)
}

// ListByDateRange returns a paginated list of activities whose date
// falls within [from, to), with the visibility filter applied.
// Ordered by date ASC.
func (repo *ActivityRepository) ListByDateRange(ctx context.Context, viewerUserID int, viewerIsAdmin bool, from, to time.Time, limit, offset int) ([]*domain.Activity, error) {
	query := activitySelectClause + `
		WHERE a.date >= $1 AND a.date < $2` + visibilityPredicate(viewerUserID, viewerIsAdmin) + `
		ORDER BY a.date ASC
		LIMIT $3 OFFSET $4`
	return repo.queryActivities(ctx, query, from, to, limit, offset)
}

// ListByClosed returns activities whose closed flag equals
// `closed`, optionally scoped to a date range, with the visibility
// filter applied. from/to are POINTERS: passing nil for either bound
// disables that side of the range. The $N::timestamptz IS NULL
// predicate handles a nil bound correctly — passing a zero time.Time
// instead would serialise to `0001-01-01 00:00:00 UTC` and
// silently exclude every row. Inclusive on both ends. Ordered by
// date ASC (closest first).
func (repo *ActivityRepository) ListByClosed(
	ctx context.Context,
	viewerUserID int,
	viewerIsAdmin bool,
	closed bool,
	from, to *time.Time,
	limit, offset int,
) ([]*domain.Activity, error) {
	query := activitySelectClause + `
		WHERE a.closed = $1
		  AND ($2::timestamptz IS NULL OR a.date >= $2)
		  AND ($3::timestamptz IS NULL OR a.date <= $3)` + visibilityPredicate(viewerUserID, viewerIsAdmin) + `
		ORDER BY a.date ASC
		LIMIT $4 OFFSET $5`
	return repo.queryActivities(ctx, query, closed, nullableTime(from), nullableTime(to), limit, offset)
}

// ListUpcoming returns a paginated list of activities scheduled at or
// after the current time (with the visibility filter applied), soonest
// first.
func (repo *ActivityRepository) ListUpcoming(ctx context.Context, viewerUserID int, viewerIsAdmin bool, limit, offset int) ([]*domain.Activity, error) {
	query := activitySelectClause + `
		WHERE a.date >= NOW()` + visibilityPredicate(viewerUserID, viewerIsAdmin) + `
		ORDER BY a.date ASC
		LIMIT $1 OFFSET $2`
	return repo.queryActivities(ctx, query, limit, offset)
}

// queryActivities is the shared row-iteration loop. Returns a
// non-nil empty slice on no rows.
func (repo *ActivityRepository) queryActivities(ctx context.Context, query string, args ...any) ([]*domain.Activity, error) {
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query activities: %w", err)
	}
	defer rows.Close()

	activities := make([]*domain.Activity, 0)
	for rows.Next() {
		activity, err := scanActivity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return activities, nil
}

// scanner is the minimal interface satisfied by both *sql.Row and
// *sql.Rows. Used by scanActivity to share the column ordering.
type scanner interface {
	Scan(dest ...any) error
}

// scanActivity reads one activity row. The column order MUST match
// activitySelectClause (10 activity columns; the d.user_id is NOT
// read from this scanner because the visibility filter has already
// narrowed the row set by the time we get here).
func scanActivity(row scanner) (*domain.Activity, error) {
	var (
		activityID      int
		activityName    string
		description     string
		activityType    string
		maxCapacity     int
		location        string
		durationInHours int
		activityDate    time.Time
		closed          bool
		dogID           sql.NullInt64
	)
	if err := row.Scan(
		&activityID, &activityName, &description, &activityType, &maxCapacity,
		&location, &durationInHours, &activityDate, &closed, &dogID,
	); err != nil {
		return nil, err
	}
	var dogPtr *int
	if dogID.Valid {
		dogPtr = new(int)
		*dogPtr = int(dogID.Int64)
	}
	return domain.ReconstituteActivity(
		activityID, activityName, description, location,
		domain.ActivityType(activityType), maxCapacity, durationInHours, activityDate, closed,
		dogPtr)
}

func mapActivityCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrForeignKeyViolation, pgErrCheckViolation:
			return fmt.Errorf("create activity: %w", err)
		}
	}
	return fmt.Errorf("create activity: %w", err)
}

func mapActivityUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrCheckViolation:
			return fmt.Errorf("update activity: %w", err)
		}
	}
	return fmt.Errorf("update activity: %w", err)
}

// pgErrCheckViolation is the SQLSTATE for CHECK constraint violations.
// Mirrors pgErrForeignKeyViolation / pgErrUniqueViolation defined in
// dog_repository.go.
const pgErrCheckViolation = "23514"

var _ domain.ActivityRepository = (*ActivityRepository)(nil)
