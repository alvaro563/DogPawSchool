package domain

import (
	"context"
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

func (i *Incompatibility) ID() int                    { return i.id }
func (i *Incompatibility) Name() string               { return i.name }
func (i *Incompatibility) Type() IncompatibilityLevel { return i.levelType }

type IncompatibilityRepository interface {
	GetIncompatibilityByID(ctx context.Context, id int) (*Incompatibility, error)
}
