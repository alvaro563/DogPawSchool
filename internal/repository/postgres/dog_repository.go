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
	return dog, nil
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

func (r *DogRepository) Update(ctx context.Context, dog *domain.Dog) error {
	return errors.New("postgres: not implemented")
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
	if err := d.UpdateProfile(domain.UpdateDogInput{
		Neutered:      neutered,
		Heat:          heat,
		WeightKg:      weightKg,
		PhotoURL:      photoURL.String,
		MedicalNotes:  medical.String,
		EducatorNotes: educator.String,
		IsActive:      isActive,
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

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
