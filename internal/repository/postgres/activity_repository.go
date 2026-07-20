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
	ErrActivityNotFound = errors.New("postgres: activity not found")
)

// activitySelectClause is the 9-column projection reused by every read
// method. Keep the column order in lockstep with scanActivity.
const activitySelectClause = `SELECT id, name, activity_type, max_capacity,
	       location, duration_in_hours, date
	FROM activities`

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Create inserts a new activity and returns the assigned id. A
// foreign-key violation (code 23503) is wrapped as a generic error;
// activity creation never references other tables so 23503 is not
// expected at runtime but is mapped for safety.
func (repo *ActivityRepository) Create(ctx context.Context, activity *domain.Activity) (int, error) {
	const query = `
		INSERT INTO activities (
			name, activity_type, max_capacity,
			location, duration_in_hours, date
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var newActivityID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		activity.Name(), string(activity.Type()), activity.MaxCapacity(),
		activity.Location(), activity.DurationInHours(), activity.Date(),
	).Scan(&newActivityID)
	if err != nil {
		return 0, mapActivityCreateError(err)
	}
	return int(newActivityID), nil
}

// GetByID fetches a single activity by id. Returns ErrActivityNotFound
// when no row matches.
func (repo *ActivityRepository) GetByID(ctx context.Context, id int) (*domain.Activity, error) {
	query := activitySelectClause + ` WHERE id = $1`
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

// Update writes all mutable fields of the activity. Returns
// ErrActivityNotFound if no row matches the id.
func (repo *ActivityRepository) Update(ctx context.Context, activity *domain.Activity) error {
	const query = `
		UPDATE activities
		SET name = $1, activity_type = $2, max_capacity = $3,
		    location = $4, duration_in_hours = $5, date = $6
		WHERE id = $7
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		activity.Name(), string(activity.Type()), activity.MaxCapacity(),
		activity.Location(), activity.DurationInHours(), activity.Date(),
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

// List returns a paginated list of all activities, most recent first.
func (repo *ActivityRepository) List(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
	query := activitySelectClause + `
		ORDER BY date DESC
		LIMIT $1 OFFSET $2`
	return repo.queryActivities(ctx, query, limit, offset)
}

// ListUpcoming returns a paginated list of activities scheduled at or
// after the current time, soonest first.
func (repo *ActivityRepository) ListUpcoming(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
	query := activitySelectClause + `
		WHERE date >= NOW()
		ORDER BY date ASC
		LIMIT $1 OFFSET $2`
	return repo.queryActivities(ctx, query, limit, offset)
}

// queryActivities is the shared row-iteration loop for List and
// ListUpcoming. Returns a non-nil empty slice on no rows.
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
// activitySelectClause.
func scanActivity(row scanner) (*domain.Activity, error) {
	var (
		activityID      int
		activityName    string
		activityType    string
		maxCapacity     int
		location        string
		durationInHours int
		activityDate    time.Time
	)
	if err := row.Scan(
		&activityID, &activityName, &activityType, &maxCapacity,
		&location, &durationInHours, &activityDate,
	); err != nil {
		return nil, err
	}
	return domain.NewActivity(
		activityID, activityName, location,
		domain.ActivityType(activityType), maxCapacity, durationInHours, activityDate,
	)
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
