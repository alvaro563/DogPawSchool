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
	// ErrInvalidPassUser is returned by Create when the user_id foreign
	// key does not resolve to an existing user.
	ErrInvalidPassUser = errors.New("postgres: pass user_id does not exist")
	// ErrPassNotFound is returned by GetByID/Update when no row matches.
	ErrPassNotFound = errors.New("postgres: pass not found")
)

// passSelectClause is the 9-column projection reused by every read
// method. Keep the column order in lockstep with scanPass.
const passSelectClause = `SELECT id, num_of_sessions, remaining_sessions, price,
	       pass_type, created_at, updated_at, expires_at, user_id
	FROM passes`

type PassRepository struct {
	db *sql.DB
}

func NewPassRepository(db *sql.DB) *PassRepository {
	return &PassRepository{db: db}
}

// Create inserts a new pass and returns the assigned id. A
// foreign-key violation on user_id (SQLSTATE 23503) is mapped to
// ErrInvalidPassUser so the handler can respond with 400. The
// remaining_sessions column is set to num_of_sessions so the
// CHECK constraint `passes_remaining_le_total` is satisfied.
func (repo *PassRepository) Create(ctx context.Context, pass *domain.Pass) (int, error) {
	const query = `
		INSERT INTO passes (num_of_sessions, remaining_sessions, price, pass_type, user_id, expires_at)
		VALUES ($1, $1, $2, $3, $4, $5)
		RETURNING id
	`
	var newPassID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		pass.NumOfSessions(), pass.Price(), string(pass.Type()),
		pass.UserID(), nullTimePtr(pass.ExpiresAt()),
	).Scan(&newPassID)
	if err != nil {
		return 0, mapPassCreateError(err)
	}
	return int(newPassID), nil
}

// GetByID fetches a single pass by id. Returns ErrPassNotFound when
// no row matches.
func (repo *PassRepository) GetByID(ctx context.Context, id int) (*domain.Pass, error) {
	query := passSelectClause + ` WHERE id = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	pass, err := scanPass(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPassNotFound
		}
		return nil, err
	}
	return pass, nil
}

// Update writes all mutable fields of the pass. Returns
// ErrPassNotFound if no row matches the id.
func (repo *PassRepository) Update(ctx context.Context, pass *domain.Pass) error {
	const query = `
		UPDATE passes
		SET num_of_sessions = $1, remaining_sessions = $2, price = $3,
		    pass_type = $4, expires_at = $5
		WHERE id = $6
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		pass.NumOfSessions(), pass.RemainingSessions(), pass.Price(),
		string(pass.Type()), nullTimePtr(pass.ExpiresAt()), pass.ID(),
	)
	if err != nil {
		return fmt.Errorf("update pass: %w", err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update pass: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrPassNotFound
	}
	return nil
}

// ListAll returns a paginated list of all passes in the system,
// most recent first.
func (repo *PassRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
	query := passSelectClause + `
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	return repo.queryPasses(ctx, query, limit, offset)
}

// ListByOwner returns a paginated list of passes owned by the given
// user, most recent first.
func (repo *PassRepository) ListByOwner(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
	query := passSelectClause + `
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	return repo.queryPasses(ctx, query, userID, limit, offset)
}

// queryPasses is the shared row-iteration loop for ListAll and
// ListByOwner. Returns a non-nil empty slice on no rows.
func (repo *PassRepository) queryPasses(ctx context.Context, query string, args ...any) ([]*domain.Pass, error) {
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query passes: %w", err)
	}
	defer rows.Close()

	passes := make([]*domain.Pass, 0)
	for rows.Next() {
		pass, err := scanPass(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pass: %w", err)
		}
		passes = append(passes, pass)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return passes, nil
}

// AddMovement inserts an append-only pass_movements row. The CHECK
// constraint `pass_movements_amount_nonzero` is enforced by the DB.
func (repo *PassRepository) AddMovement(ctx context.Context, movement *domain.PassMovement) error {
	const query = `
		INSERT INTO pass_movements (pass_id, amount, reason, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := runner(ctx, repo.db).ExecContext(ctx, query,
		movement.PassID(), movement.Amount(), movement.Reason(), movement.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("add pass movement: %w", err)
	}
	return nil
}

func mapPassCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrForeignKeyViolation:
			return ErrInvalidPassUser
		}
	}
	return fmt.Errorf("create pass: %w", err)
}

// scanner is the minimal interface satisfied by both *sql.Row and
// *sql.Rows. Used by scanPass to share the column ordering.
type passScanner interface {
	Scan(dest ...any) error
}

// scanPass reads one pass row. The column order MUST match
// passSelectClause.
func scanPass(row passScanner) (*domain.Pass, error) {
	var (
		passID           int
		numOfSessions    int
		remainingSession int
		price            int
		passType         string
		createdAt        time.Time
		updatedAt        time.Time
		expiresAt        sql.NullTime
		userID           int
	)
	if err := row.Scan(
		&passID, &numOfSessions, &remainingSession, &price,
		&passType, &createdAt, &updatedAt, &expiresAt, &userID,
	); err != nil {
		return nil, err
	}
	var expiresAtPtr *time.Time
	if expiresAt.Valid {
		expiresAtPtr = &expiresAt.Time
	}
	return domain.NewPass(
		passID, numOfSessions, remainingSession, price, domain.PassType(passType),
		userID, createdAt, updatedAt, expiresAtPtr,
	)
}

// nullTimePtr converts an optional *time.Time to a sql.NullTime for
// INSERT/UPDATE. nil and zero-value times both become NULL; non-zero
// times are written as-is. The CHECK constraint
// `passes_expiry_after_creation` is enforced by the DB.
func nullTimePtr(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

var _ domain.PassRepository = (*PassRepository)(nil)
