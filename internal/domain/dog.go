package domain

import "context"

type Incompatibility string

const (
	IncompatibilityReactivoMachos    Incompatibility = "REACTIVO_MACHOS"
	IncompatibilityNoToleraCachorros Incompatibility = "NO_TOLERA_CACHORROS"
)

type Sex string

const (
	SexMale   Sex = "MALE"
	SexFemale Sex = "FEMALE"
)

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

const (
	AgeInfantMaxMonths     = 6
	AgeAdolescentMaxMonths = 18
	AgeYoungAdultMaxMonths = 36

	WeightMiniMaxKg   = 5.0
	WeightMediumMaxKg = 20.0
)

type Dog struct {
	ID                int
	Name              string
	Breed             string
	AgeinMonths       int
	Sex               Sex
	Neutered          bool
	Heat              bool
	WeightKg          float64
	PhotoURL          string
	MedicalNotes      string
	EducatorNotes     string
	Passport          string
	Incompatibilities []Incompatibility
	UserID            int
	IsActive          bool
}

func (d *Dog) AgeBracket() AgeBracket {
	switch {
	case d.AgeinMonths < 0:
		return AgeBracketUnknown
	case d.AgeinMonths <= AgeInfantMaxMonths:
		return AgeBracketChildren
	case d.AgeinMonths <= AgeAdolescentMaxMonths:
		return AgeBracketTeenager
	case d.AgeinMonths <= AgeYoungAdultMaxMonths:
		return AgeBracketSemiAdult
	default:
		return AgeBracketAdult
	}
}

func (d *Dog) SizeBracket() SizeBracket {
	switch {
	case d.WeightKg <= 0:
		return SizeBracketUnknown
	case d.WeightKg <= WeightMiniMaxKg:
		return SizeBracketMini
	case d.WeightKg <= WeightMediumMaxKg:
		return SizeBracketMedium
	default:
		return SizeBracketLarge
	}
}

func (d *Dog) IsIntactMale() bool {
	return d.Sex == SexMale && !d.Neutered
}

type DogRepository interface {
	Create(ctx context.Context, dog *Dog) error
	Update(ctx context.Context, dog *Dog) error
	GetByID(ctx context.Context, id int) (*Dog, error)
	ListByOwner(ctx context.Context, userID int) ([]*Dog, error)
	Delete(ctx context.Context, id int) error
}
