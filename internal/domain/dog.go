package domain

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Field length caps for dog attributes. These mirror the limits the
// frontend enforces in the schema (src/domain/schemas/dog-schema.ts)
// and the body-size middleware (1 MB) — the domain enforces them
// here as a last line of defense, so an out-of-band caller cannot
// stuff multi-megabyte strings into the database via direct repo
// writes (admin scripts, internal tools, future endpoints).
const (
	maxDogNameLen        = 100
	maxDogBreedLen       = 100
	maxDogPassportLen    = 100
	maxDogPhotoURLLen    = 2048 // standard practical URL ceiling
	maxDogNotesLen       = 2000
	maxPendingReasonCode = 64
)

// validatePhotoURL accepts only absolute http(s) URLs within the
// length cap. Empty string is allowed (= "no photo") and skips
// validation. Anything else returns a validation error pointing at
// the offending field. The url.Parse error string is intentionally
// not propagated — callers do not need to know whether the URL
// failed because of scheme, host, or whitespace; only that it did.
func validatePhotoURL(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > maxDogPhotoURLLen {
		return &DogValidationError{Field: "photo_url"}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return &DogValidationError{Field: "photo_url"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &DogValidationError{Field: "photo_url"}
	}
	if strings.TrimSpace(raw) != raw {
		return &DogValidationError{Field: "photo_url"}
	}
	return nil
}

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
	incompatibilities     []Incompatibility
	traits                []Incompatibility
	userID                int
	isActive              bool
	hasSpecialCondition   bool
}

// DogPatch is a partial update for Dog: only the non-nil fields are
// applied. Each field has its own validation rules; see ApplyPatch.
type DogPatch struct {
	Name                *string
	Breed               *string
	AgeInMonths         *int
	Sex                 *Sex
	Passport            *string
	WeightKg            *float64
	Neutered            *bool
	Heat                *bool
	PhotoURL            *string
	MedicalNotes        *string
	EducatorNotes       *string
	IsActive            *bool
	HasSpecialCondition *bool
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
// invalid. A new dog starts as is_active=true. Age and weight must be
// strictly positive: a registered dog is always a real animal with a
// known (non-zero) age and weight.
func NewDog(id int, name, breed, passport string, ageInMonths int, sex Sex, weightKg float64, userID int) (*Dog, error) {
	if id < 0 {
		return nil, fmt.Errorf("dog: id must not be negative")
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxDogNameLen {
		return nil, fmt.Errorf("dog: name must be 1..%d chars", maxDogNameLen)
	}
	breed = strings.TrimSpace(breed)
	if breed == "" || len(breed) > maxDogBreedLen {
		return nil, fmt.Errorf("dog: breed must be 1..%d chars", maxDogBreedLen)
	}
	passport = strings.TrimSpace(passport)
	if passport == "" || len(passport) > maxDogPassportLen {
		return nil, fmt.Errorf("dog: passport must be 1..%d chars", maxDogPassportLen)
	}
	if ageInMonths <= 0 {
		return nil, fmt.Errorf("dog: ageInMonths must be greater than 0")
	}
	if weightKg <= 0 {
		return nil, fmt.Errorf("dog: weightKg must be greater than 0")
	}
	if !sex.IsValid() {
		return nil, fmt.Errorf("dog: invalid sex %q", sex)
	}
	if userID <= 0 {
		return nil, fmt.Errorf("dog: userID must be greater than 0")
	}
	return &Dog{
		id:                  id,
		name:                name,
		breed:               breed,
		ageInMonths:         ageInMonths,
		sex:                 sex,
		weightKg:            weightKg,
		passport:            passport,
		userID:              userID,
		isActive:            true,
		hasSpecialCondition: false,
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
func (dog *Dog) IsActive() bool              { return dog.isActive }
func (dog *Dog) HasSpecialCondition() bool   { return dog.hasSpecialCondition }

// Incompatibilities returns a defensive copy of the dog's incompatibilities.
func (dog *Dog) Incompatibilities() []Incompatibility {
	out := make([]Incompatibility, len(dog.incompatibilities))
	copy(out, dog.incompatibilities)
	return out
}

// Traits returns a defensive copy of the dog's traits. Traits are the
// tags the dog presents (MACHO_ENTERO, ALTA_ENERGIA...); they are the
// targets that other dogs' triggers may fire on.
func (dog *Dog) Traits() []Incompatibility {
	out := make([]Incompatibility, len(dog.traits))
	copy(out, dog.traits)
	return out
}

// hasTraitCode reports whether the dog presents a trait with the given code.
func (dog *Dog) hasTraitCode(code string) bool {
	if code == "" {
		return false
	}
	for _, trait := range dog.traits {
		if trait.Code() == code {
			return true
		}
	}
	return false
}

// CompatibilityConflict is one detected trigger->trait match between two
// dogs: TriggerDog carries a trigger whose target trait code matches one of
// TargetDog's traits.
type CompatibilityConflict struct {
	TriggerName     string
	TriggerLevel    IncompatibilityLevel
	TriggerDogID    int
	TriggerDogName  string
	TargetTraitCode string
	TargetTraitName string
	TargetDogID     int
	TargetDogName   string
}

// ConflictsWith returns every trigger->trait collision between the
// receiver and other, in both directions (the receiver's triggers against
// the other's traits, and vice versa). Nil and self are no-ops. An empty
// result means no conflict.
func (dog *Dog) ConflictsWith(other *Dog) []CompatibilityConflict {
	if other == nil || other.ID() == dog.ID() {
		return nil
	}
	conflicts := make([]CompatibilityConflict, 0)
	conflicts = append(conflicts, conflictsFrom(dog, other)...)
	conflicts = append(conflicts, conflictsFrom(other, dog)...)
	return conflicts
}

// conflictsFrom returns the collisions where triggerDog's triggers fire on
// targetDog's traits. A trigger fires when its target trait code equals one
// of targetDog's trait codes (see Incompatibility.FiresOn).
func conflictsFrom(triggerDog, targetDog *Dog) []CompatibilityConflict {
	targetTraits := make(map[string]Incompatibility, len(targetDog.traits))
	for _, trait := range targetDog.traits {
		if trait.Code() != "" {
			targetTraits[trait.Code()] = trait
		}
	}
	if len(targetTraits) == 0 {
		return nil
	}

	conflicts := make([]CompatibilityConflict, 0)
	for _, trigger := range triggerDog.incompatibilities {
		if !trigger.FiresOn(targetDog) {
			continue
		}
		target := targetTraits[trigger.TargetTraitCode()]
		conflicts = append(conflicts, CompatibilityConflict{
			TriggerName:     trigger.Name(),
			TriggerLevel:    trigger.Type(),
			TriggerDogID:    triggerDog.ID(),
			TriggerDogName:  triggerDog.Name(),
			TargetTraitCode: target.Code(),
			TargetTraitName: target.Name(),
			TargetDogID:     targetDog.ID(),
			TargetDogName:   targetDog.Name(),
		})
	}
	return conflicts
}

// Reason codes for SexNeuteredConflict. The domain owns these stable
// identifiers; presentation layers (handler / frontend) translate them
// to user-facing text. Keep the names and string values stable — they
// are part of the API contract.
const (
	ReasonIntactVsIntact       = "intact_vs_intact"
	ReasonIntactVsCastrated    = "intact_vs_castrated"
	ReasonCastratedVsIntact    = "castrated_vs_intact"
	ReasonCastratedVsCastrated = "castrated_vs_castrated"
)

// ReasonHasSpecialCondition is the code emitted when a dog is flagged
// as having a special condition and therefore requires admin review
// before the reservation can be confirmed.
const ReasonHasSpecialCondition = "has_special_condition"

// SexNeuteredConflict describes a single pair-level conflict between
// two dogs based on sex and neutered state. The rule is symmetric: the
// same Reason() is produced regardless of which dog is the "incoming"
// one and which is the "existing" one, because the constraint is
// unconditional on the pair. Callers should never depend on field
// ordering for behavior, only on Reason() / IsBlocker().
type SexNeuteredConflict struct {
	IncomingDogID    int
	IncomingDogName  string
	IncomingSex      Sex
	IncomingNeutered bool
	ExistingDogID    int
	ExistingDogName  string
	ExistingSex      Sex
	ExistingNeutered bool
}

// Reason returns one of the Reason* constants above, or the empty
// string when the conflict is not well-formed (e.g. zero value, or a
// pair that does not involve two males). Treat the empty string as
// "no conflict".
func (c SexNeuteredConflict) Reason() string {
	if c.IncomingSex != SexMale || c.ExistingSex != SexMale {
		return ""
	}
	switch {
	case !c.IncomingNeutered && !c.ExistingNeutered:
		return ReasonIntactVsIntact
	case !c.IncomingNeutered && c.ExistingNeutered:
		return ReasonIntactVsCastrated
	case c.IncomingNeutered && !c.ExistingNeutered:
		return ReasonCastratedVsIntact
	case c.IncomingNeutered && c.ExistingNeutered:
		return ReasonCastratedVsCastrated
	}
	return ""
}

// IsBlocker reports whether this conflict must reject the reservation
// outright. The only blocking case under the current business rule is
// two intact (non-castrated) males. Every other male-male combination
// is "pending review".
func (c SexNeuteredConflict) IsBlocker() bool {
	return c.Reason() == ReasonIntactVsIntact
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

// SexNeuteredConflictsWith returns every sex/neutered conflict between
// the receiver and other. The rule is SYMMETRIC: a single call per pair
// covers both directions ("incoming vs existing" and "existing vs
// incoming"), because the rule is unconditional on the pair — calling
// this method from either perspective yields the same conflict.
//
// Nil and self are no-ops. An empty result means no conflict of this
// type. Only male-male pairs can ever produce a conflict; any other
// combination returns nil.
//
// The receiver is treated as the "incoming" dog; the other argument as
// the "existing" dog. Callers iterating over existing slot holders
// should pass the candidate as the receiver so IncomingDogID matches
// the dog being registered.
func (dog *Dog) SexNeuteredConflictsWith(other *Dog) []SexNeuteredConflict {
	if dog == nil || other == nil || dog.ID() == other.ID() {
		return nil
	}
	if dog.sex != SexMale || other.sex != SexMale {
		return nil
	}
	return []SexNeuteredConflict{{
		IncomingDogID:    dog.id,
		IncomingDogName:  dog.name,
		IncomingSex:      dog.sex,
		IncomingNeutered: dog.neutered,
		ExistingDogID:    other.id,
		ExistingDogName:  other.name,
		ExistingSex:      other.sex,
		ExistingNeutered: other.neutered,
	}}
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

// AddTrait attaches a trait to the dog. Returns (false, nil) if it is
// already attached — AddTrait is idempotent on duplicates.
func (dog *Dog) AddTrait(trait *Incompatibility) (bool, error) {
	if trait == nil {
		return false, fmt.Errorf("dog: trait cannot be nil")
	}
	if containsIncompatibility(dog.traits, trait.ID()) {
		return false, nil
	}
	dog.traits = append(dog.traits, *trait)
	return true, nil
}

// RemoveTrait detaches the trait with the given id. Returns (false, nil)
// if the id is not attached.
func (dog *Dog) RemoveTrait(id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("dog: id must be greater than 0")
	}
	if !containsIncompatibility(dog.traits, id) {
		return false, nil
	}
	dog.traits = removeIncompatibility(dog.traits, id)
	return true, nil
}

// ApplyPatch mutates the dog in place with the fields present in the
// patch. Each field has its own validation. An empty patch is a no-op.
func (dog *Dog) ApplyPatch(patch DogPatch) error {
	if patch.Name != nil {
		trimmed := strings.TrimSpace(*patch.Name)
		if trimmed == "" || len(trimmed) > maxDogNameLen {
			return &DogValidationError{Field: "name"}
		}
		dog.name = trimmed
	}
	if patch.Breed != nil {
		trimmed := strings.TrimSpace(*patch.Breed)
		if trimmed == "" || len(trimmed) > maxDogBreedLen {
			return &DogValidationError{Field: "breed"}
		}
		dog.breed = trimmed
	}
	if patch.Passport != nil {
		trimmed := strings.TrimSpace(*patch.Passport)
		if trimmed == "" || len(trimmed) > maxDogPassportLen {
			return &DogValidationError{Field: "passport"}
		}
		dog.passport = trimmed
	}
	if patch.AgeInMonths != nil {
		if *patch.AgeInMonths <= 0 {
			return &DogValidationError{Field: "age_in_months"}
		}
		dog.ageInMonths = *patch.AgeInMonths
	}
	if patch.WeightKg != nil {
		if *patch.WeightKg <= 0 {
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
		dog.SetNeutered(*patch.Neutered)
	}
	if patch.Heat != nil {
		if err := dog.SetHeat(*patch.Heat); err != nil {
			return err
		}
	}
	if patch.PhotoURL != nil {
		if err := validatePhotoURL(*patch.PhotoURL); err != nil {
			return err
		}
		dog.photoURL = *patch.PhotoURL
	}
	if patch.MedicalNotes != nil {
		if len(*patch.MedicalNotes) > maxDogNotesLen {
			return &DogValidationError{Field: "medical_notes"}
		}
		dog.medicalNotes = *patch.MedicalNotes
	}
	if patch.EducatorNotes != nil {
		if len(*patch.EducatorNotes) > maxDogNotesLen {
			return &DogValidationError{Field: "educator_notes"}
		}
		dog.educatorNotes = *patch.EducatorNotes
	}
	if patch.IsActive != nil {
		dog.isActive = *patch.IsActive
	}
	if patch.HasSpecialCondition != nil {
		dog.hasSpecialCondition = *patch.HasSpecialCondition
	}
	return nil
}

// SetNeutered sets the neutered flag. Neutered carries no additional
// invariant, so this never fails; it exists so every state change goes
// through a domain method rather than a raw column write.
func (dog *Dog) SetNeutered(neutered bool) { dog.neutered = neutered }

// SetHeat sets the heat flag. Business invariant: only a female dog can
// be in heat, so heat=true on a non-female dog is rejected. heat=false
// is always allowed (any dog can leave heat). Centralizing the rule here
// guarantees every write path (SetHeat, ApplyPatch) enforces it.
func (dog *Dog) SetHeat(heat bool) error {
	if heat && dog.sex != SexFemale {
		return &DogValidationError{Field: "heat"}
	}
	dog.heat = heat
	return nil
}

// SetPhotoURL sets the profile photo URL. An empty string clears the
// photo. Non-empty values are validated against the http(s) scheme
// + max length cap (see validatePhotoURL); on failure the photo is
// left untouched and a *DogValidationError is returned.
func (dog *Dog) SetPhotoURL(raw string) error {
	if err := validatePhotoURL(raw); err != nil {
		return err
	}
	dog.photoURL = raw
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
	GetByIDForUpdate(ctx context.Context, id int) (*Dog, error)
	GetByIDs(ctx context.Context, ids []int) ([]*Dog, error)
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
