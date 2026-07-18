package domain

import (
	"context"
	"errors"
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
// (via the dog_incompatibilities join table).
type Incompatibility struct {
	id        int
	name      string
	levelType IncompatibilityLevel
}

// NewIncompatibility creates an Incompatibility with validated fields.
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
	return &Incompatibility{id: id, name: name, levelType: levelType}, nil
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

// IncompatibilityPatch is a partial update: only the non-nil fields are
// mutated. See ApplyPatch for per-field validation.
type IncompatibilityPatch struct {
	Name  *string
	Level *IncompatibilityLevel
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
// in the patch. An empty patch is a no-op.
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
	return nil
}

// ErrNotFound is returned by repository implementations when a row does
// not exist.
var ErrNotFound = errors.New("not found")

// IncompatibilityRepository is the persistence contract for
// Incompatibility. Implemented by
// internal/repository/postgres.IncompatibilityRepository.
type IncompatibilityRepository interface {
	GetIncompatibilityByID(ctx context.Context, id int) (*Incompatibility, error)
	Create(ctx context.Context, incomp *Incompatibility) (int, error)
	List(ctx context.Context, level *IncompatibilityLevel) ([]*Incompatibility, error)
	Update(ctx context.Context, incomp *Incompatibility) error
	Delete(ctx context.Context, id int) error
}
