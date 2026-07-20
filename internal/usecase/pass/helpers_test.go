package pass

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

// mockPassRepository is a hand-rolled mock used across the use case
// tests. Each field is a function that the test sets; nil fields
// fall back to a sensible no-op so a test only needs to stub the
// methods it cares about.
type mockPassRepository struct {
	create      func(ctx context.Context, pass *domain.Pass) (int, error)
	getByID     func(ctx context.Context, id int) (*domain.Pass, error)
	update      func(ctx context.Context, pass *domain.Pass) error
	listAll     func(ctx context.Context, limit, offset int) ([]*domain.Pass, error)
	listByOwner func(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error)
	addMovement func(ctx context.Context, movement *domain.PassMovement) error
}

func (m *mockPassRepository) Create(ctx context.Context, pass *domain.Pass) (int, error) {
	if m.create != nil {
		return m.create(ctx, pass)
	}
	return 0, nil
}

func (m *mockPassRepository) GetByID(ctx context.Context, id int) (*domain.Pass, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockPassRepository) Update(ctx context.Context, pass *domain.Pass) error {
	if m.update != nil {
		return m.update(ctx, pass)
	}
	return nil
}

func (m *mockPassRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
	if m.listAll != nil {
		return m.listAll(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockPassRepository) ListByOwner(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
	if m.listByOwner != nil {
		return m.listByOwner(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *mockPassRepository) AddMovement(ctx context.Context, movement *domain.PassMovement) error {
	if m.addMovement != nil {
		return m.addMovement(ctx, movement)
	}
	return nil
}

// mustNewPass is a test helper that panics on construction error.
// Use it inside tests where the input is known to be valid.
// The updatedAt argument is set equal to createdAt because unit tests
// never exercise the DB trigger. The remainingSessions argument is
// set equal to numOfSessions because every test starts from a
// fully-available pass and then consumes explicitly.
func mustNewPass(id, numOfSessions, price int, passType domain.PassType, userID int, createdAt time.Time, expiresAt *time.Time) *domain.Pass {
	return domain.MustNewPass(id, numOfSessions, numOfSessions, price, passType, userID, createdAt, createdAt, expiresAt)
}

// sentinelErr is a small, import-free error used in tests across this
// package to verify that repository errors are wrapped correctly. It
// lives in a _test.go file so it is not part of the compiled binary.
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
