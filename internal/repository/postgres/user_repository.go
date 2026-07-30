package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
)

// userSelectClause is the 6-column projection reused by every read
// method. Keep the column order in lockstep with scanUser. The
// password column is included so we can reconstruct the full
// *domain.User aggregate; the handler is responsible for stripping it
// from the wire response.
const userSelectClause = `SELECT id, name, email, password, role, is_active
	FROM users`

// UserRepository is the postgres implementation of domain.UserRepository.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user. The database assigns the id via
// GENERATED ALWAYS AS IDENTITY. A unique violation on the email column
// is mapped to domain.ErrDuplicateEmail; any other postgres error is
// wrapped with context.
func (repo *UserRepository) Create(ctx context.Context, user *domain.User) (int, error) {
	const query = `
		INSERT INTO users (name, email, password, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id int
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		user.Name(), user.Email(), user.Password(), string(user.Role()), user.IsActive(),
	).Scan(&id)
	if err != nil {
		return 0, mapUserUniqueError(err, "create user")
	}
	return id, nil
}

// GetByID fetches a single user by id. Returns domain.ErrNotFound when
// no row matches.
func (repo *UserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := userSelectClause + ` WHERE id = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	return user, nil
}

// GetByEmail fetches a single user by email (case-sensitive — the DB
// has no functional index on email). Returns domain.ErrNotFound when no
// row matches.
func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := userSelectClause + ` WHERE email = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

// Update persists the mutable fields (name, email, password, is_active)
// of the user. Returns domain.ErrNotFound if no row matches the id;
// domain.ErrDuplicateEmail if the new email is already in use by
// another user.
func (repo *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
		UPDATE users
		SET name = $1, email = $2, password = $3, is_active = $4
		WHERE id = $5
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		user.Name(), user.Email(), user.Password(), user.IsActive(), user.ID(),
	)
	if err != nil {
		return mapUserUniqueError(err, "update user")
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update user: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAll returns every user in the table, ordered by id ascending. No
// pagination — callers that need bounded result sets must use
// ListAllPaged. Intended for batch jobs and exports; not safe for
// unbounded user growth.
func (repo *UserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	query := userSelectClause + ` ORDER BY id ASC`
	return repo.queryUsers(ctx, query)
}

// ListAllPaged returns a paginated slice of users, ordered by id
// ascending. Pagination normalization is the caller's responsibility
// (use case input factories already clamp it).
func (repo *UserRepository) ListAllPaged(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := userSelectClause + ` ORDER BY id ASC LIMIT $1 OFFSET $2`
	return repo.queryUsers(ctx, query, limit, offset)
}

// queryUsers is the shared row-iteration loop for ListAll and
// ListAllPaged. Returns a non-nil empty slice on no rows.
func (repo *UserRepository) queryUsers(ctx context.Context, query string, args ...any) ([]*domain.User, error) {
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return users, nil
}

// Delete removes a user by id (hard delete). ON DELETE CASCADE on
// dogs.user_id will remove the user's dogs. No use case currently
// invokes this; it exists for interface completeness.
func (repo *UserRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM users WHERE id = $1`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// scanUser reads one user row. The column order MUST match
// userSelectClause. domain.NewUser always constructs a user with
// is_active=true, so when the row reports is_active=false we flip the
// flag via user.Deactivate() right after construction. This mirrors
// the dog repository's ApplyPatch reconstruction pattern.
func scanUser(row rowScanner) (*domain.User, error) {
	var (
		id                    int
		name, email, password string
		role                  string
		isActive              bool
	)
	if err := row.Scan(&id, &name, &email, &password, &role, &isActive); err != nil {
		return nil, err
	}
	user, err := domain.NewUser(id, name, email, password, domain.UserRole(role))
	if err != nil {
		return nil, fmt.Errorf("reconstruct user: %w", err)
	}
	if !isActive {
		user.Deactivate()
	}
	return user, nil
}

// mapUserUniqueError maps a postgres unique-violation (23505) on the
// users.email column to domain.ErrDuplicateEmail. Any other error is
// wrapped with the supplied operation label.
func mapUserUniqueError(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
		return domain.ErrDuplicateEmail
	}
	return fmt.Errorf("%s: %w", op, err)
}

// Compile-time assertion that *UserRepository satisfies the domain contract.
// If a method signature drifts, the build fails here instead of at runtime.
var _ domain.UserRepository = (*UserRepository)(nil)
