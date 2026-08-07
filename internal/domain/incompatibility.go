package domain

import (
	"context"
	"fmt"
)

// IncompatibilityLevel classifies how strict an incompatibility is — the
// higher the level, the more careful the matching must be at the
// scheduler.
type IncompatibilityLevel string

const (
	IncompatibilityLevelAbsoluta IncompatibilityLevel = "ABSOLUTA"
	IncompatibilityLevelMedia    IncompatibilityLevel = "MEDIA"
	IncompatibilityLevelBaja     IncompatibilityLevel = "BAJA"
)

// IsValid reports whether the value is a recognized IncompatibilityLevel.
func (levelType IncompatibilityLevel) IsValid() bool {
	switch levelType {
	case IncompatibilityLevelAbsoluta,
		IncompatibilityLevelMedia,
		IncompatibilityLevelBaja:
		return true
	}
	return false
}

// Incompatibility is a category that may be attached to one or more dogs
// (via the dog_incompatibilities join table). Its role is inferred from
// its fields rather than an explicit kind column:
//
//   - A row with code != "" is a trait: a characteristic the dog presents
//     (MACHO_ENTERO, ALTA_ENERGIA...).
//   - A row with targetTraitCode != "" is a trigger: something the dog
//     does not tolerate in class.
//
// The relationship is by code: a trigger fires when its TargetTraitCode()
// equals another dog's trait Code() (see Incompatibility.FiresOn).
type Incompatibility struct {
	id              int
	code            string
	targetTraitCode string
	name            string
	levelType       IncompatibilityLevel
}

// NewTraitIncompatibility creates an incompatibility that acts as a TRAIT,
// identified by its stable code.
func NewTraitIncompatibility(id int, code, name string, levelType IncompatibilityLevel) (*Incompatibility, error) {
	if id < 0 {
		return nil, fmt.Errorf("incompatibility: id must not be negative")
	}
	if code == "" {
		return nil, fmt.Errorf("incompatibility: trait code must not be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("incompatibility: name must not be empty")
	}
	if !levelType.IsValid() {
		return nil, fmt.Errorf("incompatibility: invalid level %q", levelType)
	}
	return &Incompatibility{
		id:        id,
		code:      code,
		name:      name,
		levelType: levelType,
	}, nil
}

// NewTriggerIncompatibility creates an incompatibility that acts as a
// TRIGGER pointing at the code of the trait it reacts to.
func NewTriggerIncompatibility(id int, name string, levelType IncompatibilityLevel, targetTraitCode string) (*Incompatibility, error) {
	if id < 0 {
		return nil, fmt.Errorf("incompatibility: id must not be negative")
	}
	if targetTraitCode == "" {
		return nil, fmt.Errorf("incompatibility: trigger target trait code must not be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("incompatibility: name must not be empty")
	}
	if !levelType.IsValid() {
		return nil, fmt.Errorf("incompatibility: invalid level %q", levelType)
	}
	return &Incompatibility{
		id:              id,
		targetTraitCode: targetTraitCode,
		name:            name,
		levelType:       levelType,
	}, nil
}

func (incompat *Incompatibility) ID() int                    { return incompat.id }
func (incompat *Incompatibility) Name() string               { return incompat.name }
func (incompat *Incompatibility) Type() IncompatibilityLevel { return incompat.levelType }
func (incompat *Incompatibility) Code() string               { return incompat.code }
func (incompat *Incompatibility) TargetTraitCode() string    { return incompat.targetTraitCode }

// FiresOn reports whether this incompatibility fires as a trigger on the
// traits of target. A row with a targetTraitCode acts as a trigger when
// that code matches one of the traits the target dog presents.
func (incompat *Incompatibility) FiresOn(target *Dog) bool {
	if target == nil || incompat.targetTraitCode == "" {
		return false
	}
	return target.hasTraitCode(incompat.targetTraitCode)
}

// IncompatibilityPatch is a partial update: only the non-nil fields are
// mutated. See ApplyPatch for per-field validation.
type IncompatibilityPatch struct {
	Name            *string
	Level           *IncompatibilityLevel
	Code            *string
	TargetTraitCode *string
}

// IncompatibilityValidationError is returned by ApplyPatch when a supplied
// value is invalid.
type IncompatibilityValidationError struct {
	Field string
}

func (validationError *IncompatibilityValidationError) Error() string {
	return fmt.Sprintf("incompatibility: invalid value for %s", validationError.Field)
}

// ApplyPatch mutates the incompatibility in place with the fields present
// in the patch. An empty patch is a no-op. Field values are validated
// individually; callers should invoke Validate before persisting.
func (incompat *Incompatibility) ApplyPatch(patch IncompatibilityPatch) error {
	if patch.Name != nil {
		if *patch.Name == "" {
			return &IncompatibilityValidationError{Field: "name"}
		}
		incompat.name = *patch.Name
	}
	if patch.Level != nil {
		if !patch.Level.IsValid() {
			return &IncompatibilityValidationError{Field: "level"}
		}
		incompat.levelType = *patch.Level
	}
	if patch.Code != nil {
		if *patch.Code == "" {
			return &IncompatibilityValidationError{Field: "code"}
		}
		incompat.code = *patch.Code
	}
	if patch.TargetTraitCode != nil {
		if *patch.TargetTraitCode == "" {
			return &IncompatibilityValidationError{Field: "target_trait_code"}
		}
		incompat.targetTraitCode = *patch.TargetTraitCode
	}
	return nil
}

// Validate enforces code/target coherence after a patch: at most one of
// code and targetTraitCode should be set.
func (incompat *Incompatibility) Validate() error {
	return nil
}

// IncompatibilityRepository is the persistence contract for
// Incompatibility. Implemented by
// internal/repository/postgres.IncompatibilityRepository.
type IncompatibilityRepository interface {
	GetIncompatibilityByID(ctx context.Context, id int) (*Incompatibility, error)
	GetByCode(ctx context.Context, code string) (*Incompatibility, error)
	Create(ctx context.Context, incomp *Incompatibility) (int, error)
	List(ctx context.Context, level *IncompatibilityLevel) ([]*Incompatibility, error)
	Update(ctx context.Context, incomp *Incompatibility) error
	Delete(ctx context.Context, id int) error
}
