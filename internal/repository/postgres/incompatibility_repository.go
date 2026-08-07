package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
)

// ErrIncompatibilityInUse aliases the domain sentinel returned by Delete
// when the incompatibility is still attached to at least one dog (FK
// 23503 from dog_incompatibilities) or, for a trait, still referenced as
// the target of at least one trigger.
var ErrIncompatibilityInUse = domain.ErrIncompatibilityInUse

type IncompatibilityRepository struct {
	db *sql.DB
}

func NewIncompatibilityRepository(db *sql.DB) *IncompatibilityRepository {
	return &IncompatibilityRepository{db: db}
}

const incompatibilitySelectClause = `SELECT id, name, level_type, code, target_trait_code FROM incompatibilities`

type incompatRowScanner interface {
	Scan(dest ...any) error
}

func scanIncompatibility(row incompatRowScanner) (*domain.Incompatibility, error) {
	var (
		incompID     int
		incompatName string
		levelType    string
		code         sql.NullString
		target       sql.NullString
	)
	if err := row.Scan(&incompID, &incompatName, &levelType, &code, &target); err != nil {
		return nil, err
	}
	if code.Valid {
		return domain.NewTraitIncompatibility(incompID, code.String, incompatName, domain.IncompatibilityLevel(levelType))
	}
	return domain.NewTriggerIncompatibility(incompID, incompatName, domain.IncompatibilityLevel(levelType), target.String)
}

func (repo *IncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	const query = incompatibilitySelectClause + ` WHERE id = $1`
	incompat, err := scanIncompatibility(runner(ctx, repo.db).QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get incompatibility %d: %w", id, err)
	}
	return incompat, nil
}

func (repo *IncompatibilityRepository) GetByCode(ctx context.Context, code string) (*domain.Incompatibility, error) {
	const query = incompatibilitySelectClause + ` WHERE code = $1`
	incompat, err := scanIncompatibility(runner(ctx, repo.db).QueryRowContext(ctx, query, code))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get incompatibility by code %q: %w", code, err)
	}
	return incompat, nil
}

func (repo *IncompatibilityRepository) Create(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
	const query = `
		INSERT INTO incompatibilities (code, name, level_type, target_trait_code)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var newIncompatID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		nullString(incomp.Code()),
		incomp.Name(),
		string(incomp.Type()),
		nullString(incomp.TargetTraitCode()),
	).Scan(&newIncompatID)
	if err != nil {
		return 0, mapIncompatibilityCreateError(err)
	}
	return int(newIncompatID), nil
}

func (repo *IncompatibilityRepository) List(ctx context.Context, level *domain.IncompatibilityLevel) ([]*domain.Incompatibility, error) {
	query := incompatibilitySelectClause
	args := make([]any, 0, 1)
	if level != nil {
		args = append(args, string(*level))
		query += ` WHERE level_type = $1`
	}
	query += ` ORDER BY name`

	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list incompatibilities: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Incompatibility, 0)
	for rows.Next() {
		incompat, err := scanIncompatibility(rows)
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
	const query = `
		UPDATE incompatibilities
		SET name = $1, level_type = $2, code = $3, target_trait_code = $4
		WHERE id = $5
	`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query,
		incomp.Name(), string(incomp.Type()),
		nullString(incomp.Code()), nullString(incomp.TargetTraitCode()), incomp.ID(),
	)
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
	incompat, err := repo.GetIncompatibilityByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete incompatibility %d: %w", id, err)
	}
	if incompat.Code() != "" {
		const checkQuery = `SELECT COUNT(*) FROM incompatibilities WHERE target_trait_code = $1`
		var count int
		if err := runner(ctx, repo.db).QueryRowContext(ctx, checkQuery, incompat.Code()).Scan(&count); err != nil {
			return fmt.Errorf("check triggers targeting trait %q: %w", incompat.Code(), err)
		}
		if count > 0 {
			return ErrIncompatibilityInUse
		}
	}

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
		switch pgErr.ConstraintName {
		case "idx_incompatibilities_name":
			return domain.ErrDuplicateIncompatibilityName
		case "idx_incompatibilities_code":
			return domain.ErrDuplicateIncompatibilityCode
		}
	}
	return fmt.Errorf("create incompatibility: %w", err)
}

func mapIncompatibilityUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "idx_incompatibilities_name":
			return domain.ErrDuplicateIncompatibilityName
		case "idx_incompatibilities_code":
			return domain.ErrDuplicateIncompatibilityCode
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
