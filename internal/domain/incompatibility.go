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

// IncompatibilityKind discriminates the two roles an incompatibility can
// play in the Traits & Triggers model:
//
//   - Trait: a characteristic/tag the dog presents (MACHO_ENTERO,
//     ALTA_ENERGIA...). Identified by a stable `code`.
//   - Trigger: something the dog does not tolerate in class. It carries a
//     `targetTraitCode` that points to the code of the trait it reacts to.
//
// The relationship is by code (a plain string), not by foreign key, so a
// trigger fires when its targetTraitCode equals the trait code of another
// dog (see Incompatibility.FiresOn).
type IncompatibilityKind string

const (
	IncompatibilityKindTrait   IncompatibilityKind = "TRAIT"
	IncompatibilityKindTrigger IncompatibilityKind = "TRIGGER"
)

// IsValid reports whether the value is a recognized IncompatibilityKind.
func (kind IncompatibilityKind) IsValid() bool {
	switch kind {
	case IncompatibilityKindTrait, IncompatibilityKindTrigger:
		return true
	}
	return false
}

// Incompatibility is a category that may be attached to one or more dogs
// (via the dog_incompatibilities join table). Depending on its Kind it is
// either a trait (what the dog is) or a trigger (what the dog reacts to).
type Incompatibility struct {
	id              int
	kind            IncompatibilityKind
	code            string
	targetTraitCode string
	name            string
	levelType       IncompatibilityLevel
}

// NewIncompatibility is the legacy constructor: it builds a TRIGGER with
// no target trait (purely informational — it never fires). It exists for
// backward compatibility with existing tests and helpers; persisted rows
// must always carry either a code (TRAIT) or a target (TRIGGER), which is
// guaranteed by the two constructors below.
func NewIncompatibility(id int, name string, levelType IncompatibilityLevel) (*Incompatibility, error) {
	if id < 0 {
		return nil, fmt.Errorf("incompatibility: id must not be negative")
	}
	if name == "" {
		return nil, fmt.Errorf("incompatibility: name must not be empty")
	}
	if !levelType.IsValid() {
		return nil, fmt.Errorf("incompatibility: invalid level %q", levelType)
	}
	return &Incompatibility{
		id:        id,
		kind:      IncompatibilityKindTrigger,
		name:      name,
		levelType: levelType,
	}, nil
}

// NewTraitIncompatibility creates a TRAIT with its identifying code.
func NewTraitIncompatibility(id int, code, name string, levelType IncompatibilityLevel) (*Incompatibility, error) {
	if code == "" {
		return nil, fmt.Errorf("incompatibility: trait code must not be empty")
	}
	incompat, err := NewIncompatibility(id, name, levelType)
	if err != nil {
		return nil, err
	}
	incompat.kind = IncompatibilityKindTrait
	incompat.code = code
	return incompat, nil
}

// NewTriggerIncompatibility creates a TRIGGER pointing at the code of the
// trait it reacts to.
func NewTriggerIncompatibility(id int, name string, levelType IncompatibilityLevel, targetTraitCode string) (*Incompatibility, error) {
	if targetTraitCode == "" {
		return nil, fmt.Errorf("incompatibility: trigger target trait code must not be empty")
	}
	incompat, err := NewIncompatibility(id, name, levelType)
	if err != nil {
		return nil, err
	}
	incompat.targetTraitCode = targetTraitCode
	return incompat, nil
}

// MustNewIncompatibility is like NewIncompatibility but panics on error.
// Intended for tests and seed data where the inputs are known to be valid.
func MustNewIncompatibility(id int, name string, levelType IncompatibilityLevel) *Incompatibility {
	incompat, err := NewIncompatibility(id, name, levelType)
	if err != nil {
		panic(err)
	}
	return incompat
}

func (incompat *Incompatibility) ID() int                    { return incompat.id }
func (incompat *Incompatibility) Name() string               { return incompat.name }
func (incompat *Incompatibility) Type() IncompatibilityLevel { return incompat.levelType }
func (incompat *Incompatibility) Kind() IncompatibilityKind  { return incompat.kind }
func (incompat *Incompatibility) Code() string               { return incompat.code }
func (incompat *Incompatibility) TargetTraitCode() string    { return incompat.targetTraitCode }

// IsTrait reports whether this incompatibility is a TRAIT.
func (incompat *Incompatibility) IsTrait() bool { return incompat.kind == IncompatibilityKindTrait }

// IsTrigger reports whether this incompatibility is a TRIGGER.
func (incompat *Incompatibility) IsTrigger() bool {
	return incompat.kind == IncompatibilityKindTrigger
}

// FiresOn reports whether this trigger fires on the traits of target: the
// receiver must be a TRIGGER whose target trait code matches one of the
// traits the target dog presents. Traits (and nil targets) never fire.
func (incompat *Incompatibility) FiresOn(target *Dog) bool {
	if target == nil || !incompat.IsTrigger() {
		return false
	}
	return target.hasTraitCode(incompat.targetTraitCode)
}

// IncompatibilityPatch is a partial update: only the non-nil fields are
// mutated. See ApplyPatch for per-field validation.
type IncompatibilityPatch struct {
	Name            *string
	Level           *IncompatibilityLevel
	Kind            *IncompatibilityKind
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
// individually; the resulting kind/code/target coherence is enforced by
// Validate, which callers must invoke before persisting.
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
	if patch.Kind != nil {
		if !patch.Kind.IsValid() {
			return &IncompatibilityValidationError{Field: "kind"}
		}
		incompat.kind = *patch.Kind
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

// Validate enforces the kind/code/target coherence after a patch: a TRAIT
// must carry a code, a TRIGGER must carry a target trait code. Returns an
// *IncompatibilityValidationError on failure.
func (incompat *Incompatibility) Validate() error {
	if !incompat.kind.IsValid() {
		return &IncompatibilityValidationError{Field: "kind"}
	}
	if incompat.IsTrait() && incompat.code == "" {
		return &IncompatibilityValidationError{Field: "code"}
	}
	if incompat.IsTrigger() && incompat.targetTraitCode == "" {
		return &IncompatibilityValidationError{Field: "target_trait_code"}
	}
	return nil
}

// IncompatibilityRepository is the persistence contract for
// Incompatibility. Implemented by
// internal/repository/postgres.IncompatibilityRepository.
type IncompatibilityRepository interface {
	GetIncompatibilityByID(ctx context.Context, id int) (*Incompatibility, error)
	GetByCode(ctx context.Context, code string) (*Incompatibility, error)
	Create(ctx context.Context, incomp *Incompatibility) (int, error)
	List(ctx context.Context, level *IncompatibilityLevel, kind *IncompatibilityKind) ([]*Incompatibility, error)
	Update(ctx context.Context, incomp *Incompatibility) error
	Delete(ctx context.Context, id int) error
}
