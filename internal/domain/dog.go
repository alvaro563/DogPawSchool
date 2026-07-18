package domain

import (
	"context"
	"fmt"
)

// Sex identifies the biological sex of a dog.
type Sex string

const (
	SexMale   Sex = "MALE"
	SexFemale Sex = "FEMALE"
)

// IsValid reports whether the value is a recognized Sex.
func (sex Sex) IsValid() bool {
	switch sex {
	case SexMale, SexFemale:
		return true
	}
	return false
}

// AgeBracket is a coarse age category used for grouping and filtering.
type AgeBracket string

const (
	AgeBracketChildren  AgeBracket = "CHILDREN"
	AgeBracketTeenager  AgeBracket = "TEENAGER"
	AgeBracketSemiAdult AgeBracket = "SEMI_ADULT"
	AgeBracketAdult     AgeBracket = "ADULT"
	AgeBracketUnknown   AgeBracket = "UNKNOWN"
)

// SizeBracket is a coarse size category derived from weight.
type SizeBracket string

const (
	SizeBracketMini    SizeBracket = "MINI"
	SizeBracketMedium  SizeBracket = "MEDIUM"
	SizeBracketLarge   SizeBracket = "LARGE"
	SizeBracketUnknown SizeBracket = "UNKNOWN"
)

// IsValid reports whether the value is a recognized AgeBracket.
func (bracket AgeBracket) IsValid() bool {
	switch bracket {
	case AgeBracketChildren, AgeBracketTeenager, AgeBracketSemiAdult, AgeBracketAdult, AgeBracketUnknown:
		return true
	}
	return false
}

// IsValid reports whether the value is a recognized SizeBracket.
func (sizeBracket SizeBracket) IsValid() bool {
	switch sizeBracket {
	case SizeBracketMini, SizeBracketMedium, SizeBracketLarge, SizeBracketUnknown:
		return true
	}
	return false
}

// Age and weight thresholds used to derive AgeBracket and SizeBracket.
// Keep these in sync with the GENERATED ... AS expressions in the
// migrations (000001_initial_schema).
const (
	AgeInfantMaxMonths     = 6
	AgeAdolescentMaxMonths = 18
	AgeYoungAdultMaxMonths = 36

	WeightMiniMaxKg   = 5.0
	WeightMediumMaxKg = 20.0
)

// Dog is the central aggregate. A Dog is owned by one User and may carry
// many Incompatibility associations.
type Dog struct {
	id                int
	name              string
	breed             string
	ageInMonths       int
	sex               Sex
	neutered          bool
	heat              bool
	weightKg          float64
	photoURL          string
	medicalNotes      string
	educatorNotes     string
	passport          string
	incompatibilities []Incompatibility
	userID            int
	isActive          bool
}

// DogPatch is a partial update for Dog: only the non-nil fields are
// applied. Each field has its own validation rules; see ApplyPatch.
type DogPatch struct {
	Name          *string
	Breed         *string
	AgeInMonths   *int
	Sex           *Sex
	Passport      *string
	WeightKg      *float64
	Neutered      *bool
	Heat          *bool
	PhotoURL      *string
	MedicalNotes  *string
	EducatorNotes *string
	IsActive      *bool
}

// DogValidationError is returned by ApplyPatch when a supplied value is
// invalid (empty string, negative number, etc.).
type DogValidationError struct {
	Field string
}

func (validationError *DogValidationError) Error() string {
	return fmt.Sprintf("dog: invalid value for %s", validationError.Field)
}

// NewDog creates a Dog with the required invariants. Returns a
// DogValidationError-equivalent error (plain fmt.Errorf) if any field is
// invalid. A new dog starts as is_active=true.
func NewDog(id int, name, breed, passport string, ageInMonths int, sex Sex, weightKg float64, userID int) (*Dog, error) {
	if id < 0 {
		return nil, fmt.Errorf("dog: id must not be negative")
	}
	if name == "" {
		return nil, fmt.Errorf("dog: name must not be empty")
	}
	if breed == "" {
		return nil, fmt.Errorf("dog: breed must not be empty")
	}
	if passport == "" {
		return nil, fmt.Errorf("dog: passport must not be empty")
	}
	if ageInMonths < 0 {
		return nil, fmt.Errorf("dog: ageInMonths must not be negative")
	}
	if weightKg < 0 {
		return nil, fmt.Errorf("dog: weightKg must not be negative")
	}
	if !sex.IsValid() {
		return nil, fmt.Errorf("dog: invalid sex %q", sex)
	}
	if userID <= 0 {
		return nil, fmt.Errorf("dog: userID must be greater than 0")
	}
	return &Dog{
		id:          id,
		name:        name,
		breed:       breed,
		ageInMonths: ageInMonths,
		sex:         sex,
		weightKg:    weightKg,
		passport:    passport,
		userID:      userID,
		isActive:    true,
	}, nil
}

func (dog *Dog) ID() int               { return dog.id }
func (dog *Dog) Name() string          { return dog.name }
func (dog *Dog) Breed() string         { return dog.breed }
func (dog *Dog) AgeInMonths() int      { return dog.ageInMonths }
func (dog *Dog) Sex() Sex              { return dog.sex }
func (dog *Dog) Neutered() bool        { return dog.neutered }
func (dog *Dog) Heat() bool            { return dog.heat }
func (dog *Dog) WeightKg() float64     { return dog.weightKg }
func (dog *Dog) PhotoURL() string      { return dog.photoURL }
func (dog *Dog) MedicalNotes() string  { return dog.medicalNotes }
func (dog *Dog) EducatorNotes() string { return dog.educatorNotes }
func (dog *Dog) Passport() string      { return dog.passport }
func (dog *Dog) UserID() int           { return dog.userID }
func (dog *Dog) IsActive() bool        { return dog.isActive }

// Incompatibilities returns a defensive copy of the dog's incompatibilities.
func (dog *Dog) Incompatibilities() []Incompatibility {
	out := make([]Incompatibility, len(dog.incompatibilities))
	copy(out, dog.incompatibilities)
	return out
}

// AgeBracket derives the age category from ageInMonths.
func (dog *Dog) AgeBracket() AgeBracket {
	switch {
	case dog.ageInMonths < 0:
		return AgeBracketUnknown
	case dog.ageInMonths <= AgeInfantMaxMonths:
		return AgeBracketChildren
	case dog.ageInMonths <= AgeAdolescentMaxMonths:
		return AgeBracketTeenager
	case dog.ageInMonths <= AgeYoungAdultMaxMonths:
		return AgeBracketSemiAdult
	default:
		return AgeBracketAdult
	}
}

// SizeBracket derives the size category from weightKg.
func (dog *Dog) SizeBracket() SizeBracket {
	switch {
	case dog.weightKg <= 0:
		return SizeBracketUnknown
	case dog.weightKg <= WeightMiniMaxKg:
		return SizeBracketMini
	case dog.weightKg <= WeightMediumMaxKg:
		return SizeBracketMedium
	default:
		return SizeBracketLarge
	}
}

// IsIntactMale reports whether the dog is a non-neutered male.
func (dog *Dog) IsIntactMale() bool {
	return dog.sex == SexMale && !dog.neutered
}

func containsIncompatibility(list []Incompatibility, id int) bool {
	for _, value := range list {
		if value.ID() == id {
			return true
		}
	}
	return false
}

func removeIncompatibility(list []Incompatibility, id int) []Incompatibility {
	out := make([]Incompatibility, 0, len(list))
	for _, value := range list {
		if value.ID() != id {
			out = append(out, value)
		}
	}
	return out
}

// AddIncompatibility attaches an incompatibility to the dog. Returns
// (false, nil) if it is already attached — AddIncompatibility is
// idempotent on duplicates.
func (dog *Dog) AddIncompatibility(incompat *Incompatibility) (bool, error) {
	if incompat == nil {
		return false, fmt.Errorf("dog: incompat cannot be nil")
	}
	if containsIncompatibility(dog.incompatibilities, incompat.ID()) {
		return false, nil
	}
	dog.incompatibilities = append(dog.incompatibilities, *incompat)
	return true, nil
}

// RemoveIncompatibility detaches the incompatibility with the given id.
// Returns (false, nil) if the id is not attached.
func (dog *Dog) RemoveIncompatibility(id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("dog: id must be greater than 0")
	}
	if !containsIncompatibility(dog.incompatibilities, id) {
		return false, nil
	}
	dog.incompatibilities = removeIncompatibility(dog.incompatibilities, id)
	return true, nil
}

// ApplyPatch mutates the dog in place with the fields present in the
// patch. Each field has its own validation. An empty patch is a no-op.
func (dog *Dog) ApplyPatch(patch DogPatch) error {
	if patch.Name != nil {
		if *patch.Name == "" {
			return &DogValidationError{Field: "name"}
		}
		dog.name = *patch.Name
	}
	if patch.Breed != nil {
		if *patch.Breed == "" {
			return &DogValidationError{Field: "breed"}
		}
		dog.breed = *patch.Breed
	}
	if patch.Passport != nil {
		if *patch.Passport == "" {
			return &DogValidationError{Field: "passport"}
		}
		dog.passport = *patch.Passport
	}
	if patch.AgeInMonths != nil {
		if *patch.AgeInMonths < 0 {
			return &DogValidationError{Field: "age_in_months"}
		}
		dog.ageInMonths = *patch.AgeInMonths
	}
	if patch.WeightKg != nil {
		if *patch.WeightKg < 0 {
			return &DogValidationError{Field: "weight_kg"}
		}
		dog.weightKg = *patch.WeightKg
	}
	if patch.Sex != nil {
		if !patch.Sex.IsValid() {
			return &DogValidationError{Field: "sex"}
		}
		dog.sex = *patch.Sex
	}
	if patch.Neutered != nil {
		dog.neutered = *patch.Neutered
	}
	if patch.Heat != nil {
		dog.heat = *patch.Heat
	}
	if patch.PhotoURL != nil {
		dog.photoURL = *patch.PhotoURL
	}
	if patch.MedicalNotes != nil {
		dog.medicalNotes = *patch.MedicalNotes
	}
	if patch.EducatorNotes != nil {
		dog.educatorNotes = *patch.EducatorNotes
	}
	if patch.IsActive != nil {
		dog.isActive = *patch.IsActive
	}
	return nil
}

// Activate marks the dog as active.
func (dog *Dog) Activate() { dog.isActive = true }

// Deactivate marks the dog as inactive.
func (dog *Dog) Deactivate() { dog.isActive = false }

// DogRepository is the persistence contract for Dog. Implemented by
// internal/repository/postgres.DogRepository. The domain declares the
// interface; the outer layer implements it (Dependency Inversion).
type DogRepository interface {
	Create(ctx context.Context, dog *Dog) (int, error)
	Update(ctx context.Context, dog *Dog) error
	GetByID(ctx context.Context, id int) (*Dog, error)
	ListByOwner(ctx context.Context, userID, limit, offset int) ([]*Dog, error)
	ListAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*Dog, error)
	ListByIncompatibility(ctx context.Context, incompatibilityID, limit, offset int) ([]*Dog, error)
	ListByBreed(ctx context.Context, breed string, limit, offset int) ([]*Dog, error)
	ListBySex(ctx context.Context, sex Sex, limit, offset int) ([]*Dog, error)
	ListByNeutered(ctx context.Context, neutered bool, limit, offset int) ([]*Dog, error)
	ListByHeat(ctx context.Context, heat bool, limit, offset int) ([]*Dog, error)
	ListByIsActive(ctx context.Context, isActive bool, limit, offset int) ([]*Dog, error)
	ListByAgeBracket(ctx context.Context, bracket AgeBracket, limit, offset int) ([]*Dog, error)
	ListBySizeBracket(ctx context.Context, bracket SizeBracket, limit, offset int) ([]*Dog, error)
	SetDogNeutered(ctx context.Context, id int, neutered bool) error
	SetDogHeat(ctx context.Context, id int, heat bool) error
	Delete(ctx context.Context, id int) error
}
