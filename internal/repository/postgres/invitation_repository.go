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

// invitationSelectClause is the 9-column projection reused by every
// read method. Keep the column order in lockstep with scanInvitation.
const invitationSelectClause = `SELECT id, email, token, role, status,
	created_by, expires_at, created_at, updated_at
	FROM invitations`

// InvitationRepository is the postgres implementation of
// domain.InvitationRepository.
type InvitationRepository struct {
	db *sql.DB
}

// NewInvitationRepository creates a new InvitationRepository backed by
// the given *sql.DB.
func NewInvitationRepository(db *sql.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create inserts a new invitation. The database assigns the id via
// GENERATED ALWAYS AS IDENTITY. A unique violation on the token column
// is mapped to domain.ErrDuplicateToken.
func (repo *InvitationRepository) Create(ctx context.Context, inv *domain.Invitation) (int, error) {
	const query = `
		INSERT INTO invitations (email, token, role, status, created_by, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		inv.Email(), inv.Token(), string(inv.Role()), string(inv.Status()),
		inv.CreatedBy(), inv.ExpiresAt(), inv.CreatedAt(), inv.UpdatedAt(),
	).Scan(&id)
	if err != nil {
		return 0, mapInvitationUniqueError(err, "create invitation")
	}
	return id, nil
}

// GetByID fetches a single invitation by id. Returns domain.ErrNotFound
// when no row matches.
func (repo *InvitationRepository) GetByID(ctx context.Context, id int) (*domain.Invitation, error) {
	query := invitationSelectClause + ` WHERE id = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	inv, err := scanInvitation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get invitation %d: %w", id, err)
	}
	return inv, nil
}

// GetByToken fetches a single invitation by its unique token. Returns
// domain.ErrNotFound when no row matches.
func (repo *InvitationRepository) GetByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	query := invitationSelectClause + ` WHERE token = $1`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, token)
	inv, err := scanInvitation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get invitation by token: %w", err)
	}
	return inv, nil
}

// GetByTokenForUpdate is like GetByToken but locks the row with
// SELECT ... FOR UPDATE so that a concurrent transaction cannot
// modify or read this invitation until the current tx completes.
func (repo *InvitationRepository) GetByTokenForUpdate(ctx context.Context, token string) (*domain.Invitation, error) {
	query := invitationSelectClause + ` WHERE token = $1 FOR UPDATE`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, token)
	inv, err := scanInvitation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get invitation by token for update: %w", err)
	}
	return inv, nil
}

// Update persists the mutable fields (email, token, role, status,
// expires_at) of the invitation. Returns domain.ErrNotFound if no row
// matches the id; domain.ErrDuplicateToken if the new token is already
// in use by another invitation.
func (repo *InvitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	const query = `
		UPDATE invitations
		SET email = $1, token = $2, role = $3, status = $4, expires_at = $5
		WHERE id = $6
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		inv.Email(), inv.Token(), string(inv.Role()), string(inv.Status()),
		inv.ExpiresAt(), inv.ID(),
	)
	if err != nil {
		return mapInvitationUniqueError(err, "update invitation")
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update invitation: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListPending returns all invitations with PENDING status, ordered by
// created_at ascending. Returns a non-nil empty slice on no rows.
func (repo *InvitationRepository) ListPending(ctx context.Context, limit, offset int) ([]*domain.Invitation, error) {
	query := invitationSelectClause + ` WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT $1 OFFSET $2`
	return repo.queryInvitations(ctx, query, limit, offset)
}

// ListByEmail returns all invitations for the given email, ordered by
// created_at ascending. Returns a non-nil empty slice on no rows.
func (repo *InvitationRepository) ListByEmail(ctx context.Context, email string) ([]*domain.Invitation, error) {
	query := invitationSelectClause + ` WHERE email = $1 ORDER BY created_at ASC`
	return repo.queryInvitations(ctx, query, email)
}

// queryInvitations is the shared row-iteration loop for ListPending
// and ListByEmail. Returns a non-nil empty slice on no rows.
func (repo *InvitationRepository) queryInvitations(ctx context.Context, query string, args ...any) ([]*domain.Invitation, error) {
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query invitations: %w", err)
	}
	defer rows.Close()

	invitations := make([]*domain.Invitation, 0)
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return invitations, nil
}

// scanInvitation reads one invitation row. The column order MUST match
// invitationSelectClause. Uses domain.NewInvitation which accepts all
// fields including status, so no post-construction fix is needed.
func scanInvitation(row rowScanner) (*domain.Invitation, error) {
	var (
		id, createdBy int
		email, token  string
		role, status  string
		expiresAt     sql.NullTime
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)
	if err := row.Scan(&id, &email, &token, &role, &status,
		&createdBy, &expiresAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	var expiresAtTime, createdAtTime, updatedAtTime time.Time
	if expiresAt.Valid {
		expiresAtTime = expiresAt.Time
	}
	if createdAt.Valid {
		createdAtTime = createdAt.Time
	}
	if updatedAt.Valid {
		updatedAtTime = updatedAt.Time
	}

	inv, err := domain.NewInvitation(
		id, createdBy, email, token,
		domain.UserRole(role),
		domain.InvitationStatus(status),
		expiresAtTime, createdAtTime, updatedAtTime,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct invitation: %w", err)
	}
	return inv, nil
}

// mapInvitationUniqueError maps a postgres unique-violation (23505) to
// domain.ErrDuplicateToken. Any other error is wrapped with the
// supplied operation label.
func mapInvitationUniqueError(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
		return domain.ErrDuplicateToken
	}
	return fmt.Errorf("%s: %w", op, err)
}

// Compile-time assertion that *InvitationRepository satisfies the
// domain contract. If a method signature drifts, the build fails here
// instead of at runtime.
var _ domain.InvitationRepository = (*InvitationRepository)(nil)
