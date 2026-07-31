package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

type mockIncompatibilityRepository struct {
	getIncompatibilityByID func(ctx context.Context, id int) (*domain.Incompatibility, error)
	getByCode              func(ctx context.Context, code string) (*domain.Incompatibility, error)
	create                 func(ctx context.Context, incomp *domain.Incompatibility) (int, error)
	list                   func(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error)
	update                 func(ctx context.Context, incomp *domain.Incompatibility) error
	delete                 func(ctx context.Context, id int) error
}

func (m *mockIncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	if m.getIncompatibilityByID != nil {
		return m.getIncompatibilityByID(ctx, id)
	}
	return nil, nil
}

func (m *mockIncompatibilityRepository) GetByCode(ctx context.Context, code string) (*domain.Incompatibility, error) {
	if m.getByCode != nil {
		return m.getByCode(ctx, code)
	}
	return nil, nil
}

func (m *mockIncompatibilityRepository) Create(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
	if m.create != nil {
		return m.create(ctx, incomp)
	}
	return 0, nil
}

func (m *mockIncompatibilityRepository) List(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error) {
	if m.list != nil {
		return m.list(ctx, level, kind)
	}
	return nil, nil
}

func (m *mockIncompatibilityRepository) Update(ctx context.Context, incomp *domain.Incompatibility) error {
	if m.update != nil {
		return m.update(ctx, incomp)
	}
	return nil
}

func (m *mockIncompatibilityRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func mustNewIncompatibility(id int, name string, level domain.IncompatibilityLevel) *domain.Incompatibility {
	in, err := domain.NewIncompatibility(id, name, level)
	if err != nil {
		panic(err)
	}
	return in
}

func mustNewTrigger(id int, name string, level domain.IncompatibilityLevel, target string) *domain.Incompatibility {
	in, err := domain.NewTriggerIncompatibility(id, name, level, target)
	if err != nil {
		panic(err)
	}
	return in
}

func mustNewTrait(id int, code, name string, level domain.IncompatibilityLevel) *domain.Incompatibility {
	in, err := domain.NewTraitIncompatibility(id, code, name, level)
	if err != nil {
		panic(err)
	}
	return in
}

// sentinelErr is a small, import-free error used in tests across this
// package to verify that repository errors are wrapped correctly.
var sentinelErr = errors.New("repo failure")

// assertValidationError is shared by every use case test in this
// package. It asserts err is a *ValidationError with the expected
// field name.
func assertValidationError(t *testing.T, err error, wantField string) {
	t.Helper()
	var validationErr *ValidationError
	if assert.True(t, errors.As(err, &validationErr), "expected ValidationError, got %T (%v)", err, err) {
		assert.Equal(t, wantField, validationErr.Field)
	}
}
