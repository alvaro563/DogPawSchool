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

// ErrIncompatibilityInUse is returned by Delete when the incompatibility
// is still attached to at least one dog (FK 23503 from dog_incompatibilities).
var ErrIncompatibilityInUse = errors.New("incompatibility in use")

type IncompatibilityRepository struct {
	db *sql.DB
}

func NewIncompatibilityRepository(db *sql.DB) *IncompatibilityRepository {
	return &IncompatibilityRepository{db: db}
}

func (repo *IncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	const query = `SELECT id, name, level_type FROM incompatibilities WHERE id = $1`
	var (
		incompID     int
		incompatName string
		levelType    string
	)
	err := runner(ctx, repo.db).QueryRowContext(ctx, query, id).Scan(&incompID, &incompatName, &levelType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get incompatibility %d: %w", id, err)
	}
	return domain.NewIncompatibility(incompID, incompatName, domain.IncompatibilityLevel(levelType))
}

func (repo *IncompatibilityRepository) Create(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
	const query = `INSERT INTO incompatibilities (name, level_type) VALUES ($1, $2) RETURNING id`
	var newIncompatID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query, incomp.Name(), string(incomp.Type())).Scan(&newIncompatID)
	if err != nil {
		return 0, mapIncompatibilityCreateError(err)
	}
	return int(newIncompatID), nil
}

func (repo *IncompatibilityRepository) List(ctx context.Context, level *domain.IncompatibilityLevel) ([]*domain.Incompatibility, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if level == nil {
		const query = `SELECT id, name, level_type FROM incompatibilities ORDER BY name`
		rows, err = runner(ctx, repo.db).QueryContext(ctx, query)
	} else {
		const query = `SELECT id, name, level_type FROM incompatibilities WHERE level_type = $1 ORDER BY name`
		rows, err = runner(ctx, repo.db).QueryContext(ctx, query, string(*level))
	}
	if err != nil {
		return nil, fmt.Errorf("list incompatibilities: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Incompatibility, 0)
	for rows.Next() {
		var (
			incompID     int
			incompatName string
			levelType    string
		)
		if err := rows.Scan(&incompID, &incompatName, &levelType); err != nil {
			return nil, fmt.Errorf("scan incompatibility: %w", err)
		}
		incompat, err := domain.NewIncompatibility(incompID, incompatName, domain.IncompatibilityLevel(levelType))
		if err != nil {
			return nil, fmt.Errorf("reconstruct incompatibility: %w", err)
		}
		out = append(out, incompat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}

func (repo *IncompatibilityRepository) Update(ctx context.Context, incomp *domain.Incompatibility) error {
	const query = `UPDATE incompatibilities SET name = $1, level_type = $2 WHERE id = $3`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, incomp.Name(), string(incomp.Type()), incomp.ID())
	if err != nil {
		return mapIncompatibilityUpdateError(err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("update incompatibility: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *IncompatibilityRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM incompatibilities WHERE id = $1`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, id)
	if err != nil {
		return mapIncompatibilityDeleteError(err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete incompatibility: rows affected: %w", err)
	}
	if rowsAffected == 0 {
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
