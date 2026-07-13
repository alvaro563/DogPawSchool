package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
	incompatuc "dogpaw/internal/usecase/incompatibility"
)

var ErrIncompatibilityInUse = errors.New("incompatibility in use")

type IncompatibilityRepository struct {
	db *sql.DB
}

func NewIncompatibilityRepository(db *sql.DB) *IncompatibilityRepository {
	return &IncompatibilityRepository{db: db}
}

func (r *IncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	const q = `SELECT id, name, level_type FROM incompatibilities WHERE id = $1`
	var (
		incompID int
		name     string
		level    string
	)
	err := r.db.QueryRowContext(ctx, q, id).Scan(&incompID, &name, &level)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get incompatibility %d: %w", id, err)
	}
	return domain.NewIncompatibility(incompID, name, domain.IncompatibilityLevel(level))
}

func (r *IncompatibilityRepository) Create(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
	const q = `INSERT INTO incompatibilities (name, level_type) VALUES ($1, $2) RETURNING id`
	var newID int64
	err := r.db.QueryRowContext(ctx, q, incomp.Name(), string(incomp.Type())).Scan(&newID)
	if err != nil {
		return 0, mapIncompatibilityCreateError(err)
	}
	return int(newID), nil
}

func (r *IncompatibilityRepository) List(ctx context.Context, level *domain.IncompatibilityLevel) ([]*domain.Incompatibility, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if level == nil {
		const q = `SELECT id, name, level_type FROM incompatibilities ORDER BY name`
		rows, err = r.db.QueryContext(ctx, q)
	} else {
		const q = `SELECT id, name, level_type FROM incompatibilities WHERE level_type = $1 ORDER BY name`
		rows, err = r.db.QueryContext(ctx, q, string(*level))
	}
	if err != nil {
		return nil, fmt.Errorf("list incompatibilities: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Incompatibility, 0)
	for rows.Next() {
		var (
			id   int
			name string
			lvl  string
		)
		if err := rows.Scan(&id, &name, &lvl); err != nil {
			return nil, fmt.Errorf("scan incompatibility: %w", err)
		}
		incomp, err := domain.NewIncompatibility(id, name, domain.IncompatibilityLevel(lvl))
		if err != nil {
			return nil, fmt.Errorf("reconstruct incompatibility: %w", err)
		}
		out = append(out, incomp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}

func (r *IncompatibilityRepository) Update(ctx context.Context, incomp *domain.Incompatibility) error {
	const q = `UPDATE incompatibilities SET name = $1, level_type = $2 WHERE id = $3`
	res, err := r.db.ExecContext(ctx, q, incomp.Name(), string(incomp.Type()), incomp.ID())
	if err != nil {
		return mapIncompatibilityUpdateError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update incompatibility: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *IncompatibilityRepository) Delete(ctx context.Context, id int) error {
	const q = `DELETE FROM incompatibilities WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return mapIncompatibilityDeleteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete incompatibility: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func mapIncompatibilityCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			return incompatuc.ErrDuplicateName
		}
	}
	return fmt.Errorf("create incompatibility: %w", err)
}

func mapIncompatibilityUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			return incompatuc.ErrDuplicateName
		}
	}
	return fmt.Errorf("update incompatibility: %w", err)
}

func mapIncompatibilityDeleteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrForeignKeyViolation:
			return ErrIncompatibilityInUse
		}
	}
	return fmt.Errorf("delete incompatibility: %w", err)
}
