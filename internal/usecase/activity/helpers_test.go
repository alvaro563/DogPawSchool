package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

// mockActivityRepository is a hand-rolled mock used across the use
// case tests. Each field is a function that the test sets; nil fields
// fall back to a sensible no-op so a test only needs to stub the
// methods it cares about.
type mockActivityRepository struct {
	create          func(ctx context.Context, activity *domain.Activity) (int, error)
	getByID         func(ctx context.Context, id int) (*domain.Activity, error)
	update          func(ctx context.Context, activity *domain.Activity) error
	delete          func(ctx context.Context, id int) error
	list            func(ctx context.Context, limit, offset int) ([]*domain.Activity, error)
	listByDateRange func(ctx context.Context, from, to time.Time, limit, offset int) ([]*domain.Activity, error)
	listUpcoming    func(ctx context.Context, limit, offset int) ([]*domain.Activity, error)
}

func (m *mockActivityRepository) Create(ctx context.Context, activity *domain.Activity) (int, error) {
	if m.create != nil {
		return m.create(ctx, activity)
	}
	return 0, nil
}

func (m *mockActivityRepository) GetByID(ctx context.Context, id int) (*domain.Activity, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockActivityRepository) GetByIDForUpdate(ctx context.Context, id int) (*domain.Activity, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockActivityRepository) Update(ctx context.Context, activity *domain.Activity) error {
	if m.update != nil {
		return m.update(ctx, activity)
	}
	return nil
}

func (m *mockActivityRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func (m *mockActivityRepository) List(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
	if m.list != nil {
		return m.list(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockActivityRepository) ListByDateRange(ctx context.Context, from, to time.Time, limit, offset int) ([]*domain.Activity, error) {
	if m.listByDateRange != nil {
		return m.listByDateRange(ctx, from, to, limit, offset)
	}
	return nil, nil
}

func (m *mockActivityRepository) ListUpcoming(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
	if m.listUpcoming != nil {
		return m.listUpcoming(ctx, limit, offset)
	}
	return nil, nil
}

// mustNewActivity is a test helper that panics on construction error.
// Use it inside tests where the input is known to be valid.
func mustNewActivity(id int, name, location string, activityType domain.ActivityType, maxCapacity, durationInHours int, date time.Time) *domain.Activity {
	return domain.MustNewActivity(id, name, "", location, activityType, maxCapacity, durationInHours, date)
}

// sentinelErr is a small, import-free error used in tests to verify
// that repository errors are wrapped correctly.
var sentinelErr = errors.New("repo failure")

// assertValidationError is a small helper shared by all use case
// tests: it asserts err is a *ValidationError with the expected field.
func assertValidationError(t *testing.T, err error, wantField string) {
	t.Helper()
	var validationErr *ValidationError
	if assert.True(t, errors.As(err, &validationErr), "expected ValidationError, got %T (%v)", err, err) {
		assert.Equal(t, wantField, validationErr.Field)
	}
}
