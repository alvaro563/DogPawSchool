package domain

import (
	"context"
	"fmt"
)

type Sex string

const (
	SexMale   Sex = "MALE"
	SexFemale Sex = "FEMALE"
)

func (s Sex) IsValid() bool {
	switch s {
	case SexMale, SexFemale:
		return true
	}
	return false
}

type AgeBracket string

const (
	AgeBracketChildren  AgeBracket = "CHILDREN"
	AgeBracketTeenager  AgeBracket = "TEENAGER"
	AgeBracketSemiAdult AgeBracket = "SEMI_ADULT"
	AgeBracketAdult     AgeBracket = "ADULT"
	AgeBracketUnknown   AgeBracket = "UNKNOWN"
)

type SizeBracket string

const (
	SizeBracketMini    SizeBracket = "MINI"
	SizeBracketMedium  SizeBracket = "MEDIUM"
	SizeBracketLarge   SizeBracket = "LARGE"
	SizeBracketUnknown SizeBracket = "UNKNOWN"
)

func (a AgeBracket) IsValid() bool {
	switch a {
	case AgeBracketChildren, AgeBracketTeenager, AgeBracketSemiAdult, AgeBracketAdult, AgeBracketUnknown:
		return true
	}
	return false
}

func (s SizeBracket) IsValid() bool {
	switch s {
	case SizeBracketMini, SizeBracketMedium, SizeBracketLarge, SizeBracketUnknown:
		return true
	}
	return false
}

const (
	AgeInfantMaxMonths     = 6
	AgeAdolescentMaxMonths = 18
	AgeYoungAdultMaxMonths = 36

	WeightMiniMaxKg   = 5.0
	WeightMediumMaxKg = 20.0
)

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

type UpdateDogInput struct {
	Neutered      bool
	Heat          bool
	WeightKg      float64
	PhotoURL      string
	MedicalNotes  string
	EducatorNotes string
	IsActive      bool
}

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

func (d *Dog) ID() int               { return d.id }
func (d *Dog) Name() string          { return d.name }
func (d *Dog) Breed() string         { return d.breed }
func (d *Dog) AgeInMonths() int      { return d.ageInMonths }
func (d *Dog) Sex() Sex              { return d.sex }
func (d *Dog) Neutered() bool        { return d.neutered }
func (d *Dog) Heat() bool            { return d.heat }
func (d *Dog) WeightKg() float64     { return d.weightKg }
func (d *Dog) PhotoURL() string      { return d.photoURL }
func (d *Dog) MedicalNotes() string  { return d.medicalNotes }
func (d *Dog) EducatorNotes() string { return d.educatorNotes }
func (d *Dog) Passport() string      { return d.passport }
func (d *Dog) UserID() int           { return d.userID }
func (d *Dog) IsActive() bool        { return d.isActive }

func (d *Dog) Incompatibilities() []Incompatibility {
	out := make([]Incompatibility, len(d.incompatibilities))
	copy(out, d.incompatibilities)
	return out
}

func (d *Dog) AgeBracket() AgeBracket {
	switch {
	case d.ageInMonths < 0:
		return AgeBracketUnknown
	case d.ageInMonths <= AgeInfantMaxMonths:
		return AgeBracketChildren
	case d.ageInMonths <= AgeAdolescentMaxMonths:
		return AgeBracketTeenager
	case d.ageInMonths <= AgeYoungAdultMaxMonths:
		return AgeBracketSemiAdult
	default:
		return AgeBracketAdult
	}
}

func (d *Dog) SizeBracket() SizeBracket {
	switch {
	case d.weightKg <= 0:
		return SizeBracketUnknown
	case d.weightKg <= WeightMiniMaxKg:
		return SizeBracketMini
	case d.weightKg <= WeightMediumMaxKg:
		return SizeBracketMedium
	default:
		return SizeBracketLarge
	}
}

func (d *Dog) IsIntactMale() bool {
	return d.sex == SexMale && !d.neutered
}

func containsIncompatibility(list []Incompatibility, id int) bool {
	for _, v := range list {
		if v.ID() == id {
			return true
		}
	}
	return false
}

func removeIncompatibility(list []Incompatibility, id int) []Incompatibility {
	out := make([]Incompatibility, 0, len(list))
	for _, v := range list {
		if v.ID() != id {
			out = append(out, v)
		}
	}
	return out
}

func (d *Dog) AddIncompatibility(incompat *Incompatibility) (bool, error) {
	if incompat == nil {
		return false, fmt.Errorf("dog: incompat cannot be nil")
	}
	if containsIncompatibility(d.incompatibilities, incompat.ID()) {
		return false, nil
	}
	d.incompatibilities = append(d.incompatibilities, *incompat)
	return true, nil
}

func (d *Dog) RemoveIncompatibility(id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("dog: id must be greater than 0")
	}
	if !containsIncompatibility(d.incompatibilities, id) {
		return false, nil
	}
	d.incompatibilities = removeIncompatibility(d.incompatibilities, id)
	return true, nil
}

func (d *Dog) UpdateProfile(input UpdateDogInput) error {
	if input.WeightKg < 0 {
		return fmt.Errorf("dog: weightKg must not be negative")
	}
	d.neutered = input.Neutered
	d.heat = input.Heat
	d.weightKg = input.WeightKg
	d.photoURL = input.PhotoURL
	d.medicalNotes = input.MedicalNotes
	d.educatorNotes = input.EducatorNotes
	d.isActive = input.IsActive
	return nil
}

func (d *Dog) Activate()   { d.isActive = true }
func (d *Dog) Deactivate() { d.isActive = false }

type DogRepository interface {
	Create(ctx context.Context, dog *Dog) error
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
	Delete(ctx context.Context, id int) error
}
