package domain

import (
	"context"
	"errors"
	"fmt"
)

type IncompatibilityLevel string

const (
	IncompatibilityLevelAbsoluta IncompatibilityLevel = "ABSOLUTA"
	IncompatibilityLevelMedia    IncompatibilityLevel = "MEDIA"
	IncompatibilityLevelBaja     IncompatibilityLevel = "BAJA"
)

func (l IncompatibilityLevel) IsValid() bool {
	switch l {
	case IncompatibilityLevelAbsoluta,
		IncompatibilityLevelMedia,
		IncompatibilityLevelBaja:
		return true
	}
	return false
}

type Incompatibility struct {
	id        int
	name      string
	levelType IncompatibilityLevel
}

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
	in, err := NewIncompatibility(id, name, levelType)
	if err != nil {
		panic(err)
	}
	return in
}

func (i *Incompatibility) ID() int                    { return i.id }
func (i *Incompatibility) Name() string               { return i.name }
func (i *Incompatibility) Type() IncompatibilityLevel { return i.levelType }

// IncompatibilityPatch is a partial update: only the non-nil fields are mutated.
type IncompatibilityPatch struct {
	Name  *string
	Level *IncompatibilityLevel
}

// IncompatibilityValidationError is returned by ApplyPatch when a supplied value is invalid.
type IncompatibilityValidationError struct {
	Field string
}

func (e *IncompatibilityValidationError) Error() string {
	return fmt.Sprintf("incompatibility: invalid value for %s", e.Field)
}

func (i *Incompatibility) ApplyPatch(p IncompatibilityPatch) error {
	if p.Name != nil {
		if *p.Name == "" {
			return &IncompatibilityValidationError{Field: "name"}
		}
		i.name = *p.Name
	}
	if p.Level != nil {
		if !p.Level.IsValid() {
			return &IncompatibilityValidationError{Field: "level"}
		}
		i.levelType = *p.Level
	}
	return nil
}

// ErrNotFound is returned by repository implementations when a row does not exist.
var ErrNotFound = errors.New("not found")

type IncompatibilityRepository interface {
	GetIncompatibilityByID(ctx context.Context, id int) (*Incompatibility, error)
	Create(ctx context.Context, incomp *Incompatibility) (int, error)
	List(ctx context.Context, level *IncompatibilityLevel) ([]*Incompatibility, error)
	Update(ctx context.Context, incomp *Incompatibility) error
	Delete(ctx context.Context, id int) error
}
