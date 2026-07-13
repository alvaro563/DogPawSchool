package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
)

var (
	ErrNotFound          = errors.New("postgres: dog not found")
	ErrInvalidUser       = errors.New("postgres: user_id does not exist")
	ErrDuplicatePassport = errors.New("postgres: passport already exists")
)

const (
	pgErrForeignKeyViolation = "23503"
	pgErrUniqueViolation     = "23505"
)

type DogRepository struct {
	db *sql.DB
}

func NewDogRepository(db *sql.DB) *DogRepository {
	return &DogRepository{db: db}
}

func (r *DogRepository) Create(ctx context.Context, d *domain.Dog) (int, error) {
	const q = `
		INSERT INTO dogs (
			user_id, name, breed, age_in_months, sex,
			neutered, heat, weight_kg,
			photo_url, medical_notes, educator_notes,
			passport, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	var newID int64
	err := r.db.QueryRowContext(ctx, q,
		d.UserID(), d.Name(), d.Breed(), d.AgeInMonths(), d.Sex(),
		d.Neutered(), d.Heat(), d.WeightKg(),
		nullString(d.PhotoURL()), nullString(d.MedicalNotes()), nullString(d.EducatorNotes()),
		d.Passport(), d.IsActive(),
	).Scan(&newID)
	if err != nil {
		return 0, mapCreateError(err)
	}
	return int(newID), nil
}

func (r *DogRepository) GetByID(ctx context.Context, id int) (*domain.Dog, error) {
	const q = `
		SELECT id, user_id, name, breed, age_in_months, sex,
		       neutered, heat, weight_kg,
		       photo_url, medical_notes, educator_notes,
		       passport, is_active
		FROM dogs WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	dog, err := scanDog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	incompats, err := r.loadIncompatibilities(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load dog incompatibilities: %w", err)
	}
	for _, in := range incompats {
		if _, err := dog.AddIncompatibility(in); err != nil {
			return nil, fmt.Errorf("attach incompatibility %d: %w", in.ID(), err)
		}
	}
	return dog, nil
}

// loadIncompatibilities returns all incompatibilities currently attached to
// the given dog, in insertion order (oldest first).
func (r *DogRepository) loadIncompatibilities(ctx context.Context, dogID int) ([]*domain.Incompatibility, error) {
	const q = `
		SELECT i.id, i.name, i.level_type
		FROM incompatibilities i
		JOIN dog_incompatibilities di ON di.incompatibility_id = i.id
		WHERE di.dog_id = $1
		ORDER BY di.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, dogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Incompatibility, 0)
	for rows.Next() {
		var (
			id    int
			name  string
			level string
		)
		if err := rows.Scan(&id, &name, &level); err != nil {
			return nil, err
		}
		in, err := domain.NewIncompatibility(id, name, domain.IncompatibilityLevel(level))
		if err != nil {
			return nil, fmt.Errorf("reconstruct incompatibility %d: %w", id, err)
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *DogRepository) ListByOwner(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
	const q = `
		SELECT id, user_id, name, breed, age_in_months, sex,
		       neutered, heat, weight_kg,
		       photo_url, medical_notes, educator_notes,
		       passport, is_active
		FROM dogs
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs: %w", err)
	}
	defer rows.Close()

	dogs := make([]*domain.Dog, 0, limit)
	for rows.Next() {
		dog, err := scanDog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dog: %w", err)
		}
		dogs = append(dogs, dog)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return dogs, nil
}

// Update persists a Dog aggregate: the dog row plus all its incompatibilities.
// Uses a transaction so the aggregate is always consistent:
//  1. UPDATE the dog row
//  2. DELETE all existing dog_incompatibilities rows for this dog
//  3. INSERT the current dog_incompatibilities rows
//
// This is the "aggregate root persistence" pattern: the Dog is the aggregate
// root and its Incompatibility[] slice is part of the aggregate. A single
// Update call must persist the whole aggregate atomically.
func (r *DogRepository) Update(ctx context.Context, d *domain.Dog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Rollback is a no-op after Commit, so this is always safe.
	defer func() { _ = tx.Rollback() }()

	const updateQ = `
		UPDATE dogs SET
			name = $1, breed = $2, age_in_months = $3, sex = $4,
			weight_kg = $5, neutered = $6, heat = $7,
			photo_url = $8, medical_notes = $9, educator_notes = $10,
			passport = $11, is_active = $12
		WHERE id = $13
	`
	res, err := tx.ExecContext(ctx, updateQ,
		d.Name(), d.Breed(), d.AgeInMonths(), d.Sex(),
		d.WeightKg(), d.Neutered(), d.Heat(),
		nullString(d.PhotoURL()), nullString(d.MedicalNotes()), nullString(d.EducatorNotes()),
		d.Passport(), d.IsActive(), d.ID(),
	)
	if err != nil {
		return mapUpdateError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update dog: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM dog_incompatibilities WHERE dog_id = $1`, d.ID()); err != nil {
		return fmt.Errorf("delete dog incompatibilities: %w", err)
	}

	for _, in := range d.Incompatibilities() {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO dog_incompatibilities (dog_id, incompatibility_id) VALUES ($1, $2)`,
			d.ID(), in.ID()); err != nil {
			return fmt.Errorf("insert dog incompatibility %d: %w", in.ID(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *DogRepository) ListAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByIncompatibility(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByBreed(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListBySex(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByNeutered(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByHeat(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByIsActive(ctx context.Context, isActive bool, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListByAgeBracket(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) ListBySizeBracket(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
	return nil, errors.New("postgres: not implemented")
}

func (r *DogRepository) Delete(ctx context.Context, id int) error {
	return errors.New("postgres: not implemented")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDog(s rowScanner) (*domain.Dog, error) {
	var (
		id                          int64
		userID                      int64
		name, breed                 string
		ageInMonths                 int
		sex                         string
		neutered, heat, isActive    bool
		weightKg                    float64
		photoURL, medical, educator sql.NullString
		passport                    string
	)
	if err := s.Scan(&id, &userID, &name, &breed, &ageInMonths, &sex,
		&neutered, &heat, &weightKg, &photoURL, &medical, &educator,
		&passport, &isActive); err != nil {
		return nil, err
	}
	d, err := domain.NewDog(int(id), name, breed, passport, ageInMonths,
		domain.Sex(sex), weightKg, int(userID))
	if err != nil {
		return nil, fmt.Errorf("reconstruct dog: %w", err)
	}
	if err := d.ApplyPatch(domain.DogPatch{
		Neutered:      &neutered,
		Heat:          &heat,
		WeightKg:      &weightKg,
		PhotoURL:      &photoURL.String,
		MedicalNotes:  &medical.String,
		EducatorNotes: &educator.String,
		IsActive:      &isActive,
	}); err != nil {
		return nil, fmt.Errorf("reconstruct dog profile: %w", err)
	}
	return d, nil
}

func mapCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrForeignKeyViolation:
			return ErrInvalidUser
		case pgErrUniqueViolation:
			return ErrDuplicatePassport
		}
	}
	return fmt.Errorf("create dog: %w", err)
}

func mapUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			return ErrDuplicatePassport
		case pgErrForeignKeyViolation:
			return ErrInvalidUser
		}
	}
	return fmt.Errorf("update dog: %w", err)
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
