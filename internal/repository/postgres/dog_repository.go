package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"dogpaw/internal/domain"
)

// These sentinels alias the domain-level persistence errors so that
// callers match them via errors.Is(err, domain.ErrX) without importing
// this package. They exist only for backward compatibility within the
// repository layer and tests; new code should reference domain.ErrX.
var (
	ErrNotFound          = domain.ErrNotFound
	ErrInvalidUser       = domain.ErrInvalidUserReference
	ErrDuplicatePassport = domain.ErrDuplicatePassport
)

const (
	pgErrForeignKeyViolation = "23503"
	pgErrUniqueViolation     = "23505"
)

// dogSelectClause is the 14-column projection reused by every list method.
// Keep the column order in lockstep with scanDog.
const dogSelectClause = `SELECT id, user_id, name, breed, age_in_months, sex,
	       neutered, heat, weight_kg,
	       photo_url, medical_notes, educator_notes,
	       passport, is_active
	FROM dogs`

// dogJoinSelectClause is the projection used by ListByIncompatibility. It
// qualifies the dog columns with `d.` so the planner can disambiguate them
// from the joined dog_incompatibilities columns.
const dogJoinSelectClause = `SELECT d.id, d.user_id, d.name, d.breed, d.age_in_months, d.sex,
	       d.neutered, d.heat, d.weight_kg,
	       d.photo_url, d.medical_notes, d.educator_notes,
	       d.passport, d.is_active
	FROM dogs d`

type DogRepository struct {
	db *sql.DB
}

func NewDogRepository(db *sql.DB) *DogRepository {
	return &DogRepository{db: db}
}

func (repo *DogRepository) Create(ctx context.Context, dog *domain.Dog) (int, error) {
	const query = `
		INSERT INTO dogs (
			user_id, name, breed, age_in_months, sex,
			neutered, heat, weight_kg,
			photo_url, medical_notes, educator_notes,
			passport, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	var newDogID int64
	err := runner(ctx, repo.db).QueryRowContext(ctx, query,
		dog.UserID(), dog.Name(), dog.Breed(), dog.AgeInMonths(), dog.Sex(),
		dog.Neutered(), dog.Heat(), dog.WeightKg(),
		nullString(dog.PhotoURL()), nullString(dog.MedicalNotes()), nullString(dog.EducatorNotes()),
		dog.Passport(), dog.IsActive(),
	).Scan(&newDogID)
	if err != nil {
		return 0, mapCreateError(err)
	}
	return int(newDogID), nil
}

func (repo *DogRepository) GetByID(ctx context.Context, id int) (*domain.Dog, error) {
	const query = `
		SELECT id, user_id, name, breed, age_in_months, sex,
		       neutered, heat, weight_kg,
		       photo_url, medical_notes, educator_notes,
		       passport, is_active
		FROM dogs WHERE id = $1
	`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	dog, err := scanDog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := repo.loadIncompatibilitiesForDogs(ctx, []*domain.Dog{dog}); err != nil {
		return nil, fmt.Errorf("load dog incompatibilities: %w", err)
	}
	return dog, nil
}

// GetByIDForUpdate fetches a single dog by id, locks the row with FOR
// UPDATE until the transaction commits, and loads its incompatibility
// list. Returns ErrNotFound when no row matches.
func (repo *DogRepository) GetByIDForUpdate(ctx context.Context, id int) (*domain.Dog, error) {
	const query = `
		SELECT id, user_id, name, breed, age_in_months, sex,
		       neutered, heat, weight_kg,
		       photo_url, medical_notes, educator_notes,
		       passport, is_active
		FROM dogs WHERE id = $1 FOR UPDATE
	`
	row := runner(ctx, repo.db).QueryRowContext(ctx, query, id)
	dog, err := scanDog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := repo.loadIncompatibilitiesForDogs(ctx, []*domain.Dog{dog}); err != nil {
		return nil, fmt.Errorf("load dog incompatibilities: %w", err)
	}
	return dog, nil
}

// loadIncompatibilitiesForDogs fetches traits and triggers for every dog
// in the slice in a single query (no N+1), then attaches them to each dog
// via AddTrait/AddIncompatibility depending on the row kind. The dogs slice
// is modified in place.
//
// Called by every List* method right after the main query, so the response
// always includes the dog's full traits and triggers. Total round-trips per
// list call: 2 (one for the dogs, one for the associations) regardless of
// the result size.
func (repo *DogRepository) loadIncompatibilitiesForDogs(ctx context.Context, dogs []*domain.Dog) error {
	if len(dogs) == 0 {
		return nil
	}
	// Build "($1, $2, ..., $N)" placeholders dynamically for the IN clause.
	// database/sql does not support array params without a driver-specific
	// extension; per-id placeholders are portable and Postgres optimizes
	// them as well as array params for small N.
	placeholders := make([]string, len(dogs))
	args := make([]any, len(dogs))
	for i, dog := range dogs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = dog.ID()
	}
	query := `
		SELECT di.dog_id, i.id, i.name, i.level_type, i.kind, i.code, i.target_trait_code
		FROM dog_incompatibilities di
		JOIN incompatibilities i ON i.id = di.incompatibility_id
		WHERE di.dog_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY di.dog_id ASC, di.created_at ASC`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("load incompatibilities for %d dogs: %w", len(dogs), err)
	}
	defer rows.Close()

	// Index dogs by ID for O(1) lookup while scanning the result set.
	dogByID := make(map[int]*domain.Dog, len(dogs))
	for _, dog := range dogs {
		dogByID[dog.ID()] = dog
	}

	for rows.Next() {
		var dogID, incompID int
		var incompatName, levelType, kind string
		var code, target sql.NullString
		if err := rows.Scan(&dogID, &incompID, &incompatName, &levelType, &kind, &code, &target); err != nil {
			return err
		}
		dog, ok := dogByID[dogID]
		if !ok {
			// The JOIN guarantees dog_id matches a dog we just loaded.
			// If this fires, the DB has a row referencing a missing dog.
			return fmt.Errorf("incompatibility %d references unknown dog %d", incompID, dogID)
		}
		switch domain.IncompatibilityKind(kind) {
		case domain.IncompatibilityKindTrait:
			trait, err := domain.NewTraitIncompatibility(incompID, code.String, incompatName, domain.IncompatibilityLevel(levelType))
			if err != nil {
				return fmt.Errorf("reconstruct trait %d: %w", incompID, err)
			}
			if _, err := dog.AddTrait(trait); err != nil {
				return fmt.Errorf("attach trait %d to dog %d: %w", incompID, dogID, err)
			}
		case domain.IncompatibilityKindTrigger:
			trigger, err := domain.NewTriggerIncompatibility(incompID, incompatName, domain.IncompatibilityLevel(levelType), target.String)
			if err != nil {
				return fmt.Errorf("reconstruct trigger %d: %w", incompID, err)
			}
			if _, err := dog.AddIncompatibility(trigger); err != nil {
				return fmt.Errorf("attach trigger %d to dog %d: %w", incompID, dogID, err)
			}
		default:
			return fmt.Errorf("incompatibility %d has unknown kind %q", incompID, kind)
		}
	}
	return rows.Err()
}

func (repo *DogRepository) ListByOwner(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by owner: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

// Update persists a Dog aggregate: the dog row plus all its incompatibilities.
// The aggregate must be written atomically:
//  1. UPDATE the dog row
//  2. DELETE all existing dog_incompatibilities rows for this dog
//  3. INSERT the current dog_incompatibilities rows
//
// This is the "aggregate root persistence" pattern: the Dog is the aggregate
// root and its Incompatibility[] slice is part of the aggregate. A single
// Update call must persist the whole aggregate atomically, so the work runs
// through Transactor.WithinTx (which also lets the method join an ambient
// transaction started by a use case).
func (repo *DogRepository) Update(ctx context.Context, dog *domain.Dog) error {
	return NewTransactor(repo.db).WithinTx(ctx, func(txCtx context.Context) error {
		const updateQuery = `
			UPDATE dogs SET
				name = $1, breed = $2, age_in_months = $3, sex = $4,
				weight_kg = $5, neutered = $6, heat = $7,
				photo_url = $8, medical_notes = $9, educator_notes = $10,
				passport = $11, is_active = $12
			WHERE id = $13
		`
		queryResult, err := runner(txCtx, repo.db).ExecContext(txCtx, updateQuery,
			dog.Name(), dog.Breed(), dog.AgeInMonths(), dog.Sex(),
			dog.WeightKg(), dog.Neutered(), dog.Heat(),
			nullString(dog.PhotoURL()), nullString(dog.MedicalNotes()), nullString(dog.EducatorNotes()),
			dog.Passport(), dog.IsActive(), dog.ID(),
		)
		if err != nil {
			return mapUpdateError(err)
		}
		rowsAffected, err := queryResult.RowsAffected()
		if err != nil {
			return fmt.Errorf("update dog: rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return ErrNotFound
		}

		if _, err := runner(txCtx, repo.db).ExecContext(txCtx,
			`DELETE FROM dog_incompatibilities WHERE dog_id = $1`, dog.ID()); err != nil {
			return fmt.Errorf("delete dog incompatibilities: %w", err)
		}

		for _, incompat := range dog.Incompatibilities() {
			if _, err := runner(txCtx, repo.db).ExecContext(txCtx,
				`INSERT INTO dog_incompatibilities (dog_id, incompatibility_id) VALUES ($1, $2)`,
				dog.ID(), incompat.ID()); err != nil {
				return fmt.Errorf("insert dog incompatibility %d: %w", incompat.ID(), err)
			}
		}
		for _, trait := range dog.Traits() {
			if _, err := runner(txCtx, repo.db).ExecContext(txCtx,
				`INSERT INTO dog_incompatibilities (dog_id, incompatibility_id) VALUES ($1, $2)`,
				dog.ID(), trait.ID()); err != nil {
				return fmt.Errorf("insert dog trait %d: %w", trait.ID(), err)
			}
		}
		return nil
	})
}

// GetByIDs fetches the dogs with the given ids, ordered by id ascending,
// with their full traits and triggers loaded. Returns a non-nil empty slice
// when ids is empty or none match. Used by the reservation register flow to
// load the compatibility profiles of the dogs already holding a slot.
func (repo *DogRepository) GetByIDs(ctx context.Context, ids []int) ([]*domain.Dog, error) {
	if len(ids) == 0 {
		return []*domain.Dog{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := dogSelectClause + `
		WHERE id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY id ASC`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query dogs by ids: %w", err)
	}
	defer rows.Close()

	dogs := make([]*domain.Dog, 0, len(ids))
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for dogs by ids: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
	var (
		query string
		args  []any
	)
	if activeOnly {
		query = dogSelectClause + `
			WHERE is_active = $1
			ORDER BY id DESC
			LIMIT $2 OFFSET $3`
		args = []any{true, limit, offset}
	} else {
		query = dogSelectClause + `
			ORDER BY id DESC
			LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query all dogs: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByIncompatibility(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
	const query = dogJoinSelectClause + `
		INNER JOIN dog_incompatibilities di ON di.dog_id = d.id
		WHERE di.incompatibility_id = $1
		ORDER BY d.id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, incompatibilityID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by incompatibility: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByBreed(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE breed = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, breed, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by breed: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListBySex(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE sex = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, string(sex), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by sex: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByNeutered(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE neutered = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, neutered, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by neutered: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByHeat(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE heat = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, heat, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by heat: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByIsActive(ctx context.Context, isActive bool, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE is_active = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, isActive, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by is_active: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListByAgeBracket(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE age_bracket = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, string(bracket), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by age bracket: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

func (repo *DogRepository) ListBySizeBracket(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
	const query = dogSelectClause + `
		WHERE size_bracket = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`
	rows, err := runner(ctx, repo.db).QueryContext(ctx, query, string(bracket), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dogs by size bracket: %w", err)
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
	if err := repo.loadIncompatibilitiesForDogs(ctx, dogs); err != nil {
		return nil, fmt.Errorf("load incompatibilities for listed dogs: %w", err)
	}
	return dogs, nil
}

// Delete removes a dog by id. Cascades are handled at the DB level by the
// ON DELETE CASCADE foreign keys on dog_incompatibilities.dog_id and
// reservations.dog_id, so the join rows are removed atomically with the dog.
// Returns ErrNotFound if no dog with the given id exists.
func (repo *DogRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM dogs WHERE id = $1`
	queryResult, err := runner(ctx, repo.db).ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete dog: %w", err)
	}
	rowsAffected, err := queryResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete dog: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDog(scanner rowScanner) (*domain.Dog, error) {
	var (
		id                                    int64
		userID                                int64
		name, breed                           string
		ageInMonths                           int
		sex                                   string
		neutered, heat, isActive              bool
		weightKg                              float64
		photoURL, medicalNotes, educatorNotes sql.NullString
		passport                              string
	)
	if err := scanner.Scan(&id, &userID, &name, &breed, &ageInMonths, &sex,
		&neutered, &heat, &weightKg, &photoURL, &medicalNotes, &educatorNotes,
		&passport, &isActive); err != nil {
		return nil, err
	}
	dog, err := domain.NewDog(int(id), name, breed, passport, ageInMonths,
		domain.Sex(sex), weightKg, int(userID))
	if err != nil {
		return nil, fmt.Errorf("reconstruct dog: %w", err)
	}
	if err := dog.ApplyPatch(domain.DogPatch{
		Neutered:      &neutered,
		Heat:          &heat,
		WeightKg:      &weightKg,
		PhotoURL:      &photoURL.String,
		MedicalNotes:  &medicalNotes.String,
		EducatorNotes: &educatorNotes.String,
		IsActive:      &isActive,
	}); err != nil {
		return nil, fmt.Errorf("reconstruct dog profile: %w", err)
	}
	return dog, nil
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

// Compile-time assertion that *DogRepository satisfies the domain contract.
// If a method signature drifts, the build fails here instead of at runtime.
var _ domain.DogRepository = (*DogRepository)(nil)
