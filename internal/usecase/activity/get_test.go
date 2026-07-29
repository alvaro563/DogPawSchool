package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestGetActivityUseCase_Success(t *testing.T) {
	t.Parallel()
	fixedDate := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := mustNewActivity(7, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedDate)
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 7, id)
			return expected, nil
		},
	}
	uc := NewGetActivityUseCase(repo)

	in := MustNewGetActivityInput(7)
	output, err := uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.Same(t, expected, output.Activity)
}

func TestGetActivityUseCase_NotFound(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return nil, nil
		},
	}
	uc := NewGetActivityUseCase(repo)
	in := MustNewGetActivityInput(99)
	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNewGetActivityInput_InvalidID(t *testing.T) {
	t.Parallel()
	for _, id := range []int{0, -5} {
		_, err := NewGetActivityInput(id)
		assertValidationError(t, err, "id")
	}
}

func TestGetActivityUseCase_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return nil, sentinelErr
		},
	}
	uc := NewGetActivityUseCase(repo)
	in := MustNewGetActivityInput(1)
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
}
