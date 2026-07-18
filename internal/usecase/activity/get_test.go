package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestGetActivityUseCase_Success(t *testing.T) {
	fixedDate := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := mustNewActivity(7, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedDate)
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 7, id)
			return expected, nil
		},
	}
	uc := NewGetActivityUseCase(repo)

	output, err := uc.Execute(context.Background(), GetActivityInput{ID: 7})

	require.NoError(t, err)
	assert.Same(t, expected, output.Activity)
}

func TestGetActivityUseCase_NotFound(t *testing.T) {
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return nil, nil
		},
	}
	uc := NewGetActivityUseCase(repo)
	_, err := uc.Execute(context.Background(), GetActivityInput{ID: 99})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetActivityUseCase_InvalidID(t *testing.T) {
	repo := &mockActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			t.Fatal("repo should not be called on invalid id")
			return nil, nil
		},
	}
	uc := NewGetActivityUseCase(repo)

	tests := []struct {
		name string
		id   int
	}{
		{"zero", 0},
		{"negative", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), GetActivityInput{ID: tt.id})
			assertValidationError(t, err, "id")
		})
	}
}

func TestGetActivityUseCase_RepoError(t *testing.T) {
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return nil, sentinelErr
		},
	}
	uc := NewGetActivityUseCase(repo)
	_, err := uc.Execute(context.Background(), GetActivityInput{ID: 1})
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
}

// assertValidationError is a small helper shared by all use case
// tests: it asserts err is a *ValidationError with the expected field.
func assertValidationError(t *testing.T, err error, wantField string) {
	t.Helper()
	var validationErr *ValidationError
	if assert.True(t, errors.As(err, &validationErr), "expected ValidationError, got %T (%v)", err, err) {
		assert.Equal(t, wantField, validationErr.Field)
	}
}
