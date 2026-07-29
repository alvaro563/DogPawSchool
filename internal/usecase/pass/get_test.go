package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestGetPassUseCase_Success(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := domain.MustNewPass(7, 10, 10, 100, domain.PassGeneric, 1, now, now, nil)
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			assert.Equal(t, 7, id)
			return expected, nil
		},
	}
	uc := NewGetPassUseCase(repo)

	in := MustNewGetPassInput(7)
	output, err := uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.Same(t, expected, output.Pass)
}

func TestGetPassUseCase_NotFound(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, nil
		},
	}
	uc := NewGetPassUseCase(repo)
	in := MustNewGetPassInput(99)
	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNewGetPassInput_InvalidID(t *testing.T) {
	t.Parallel()
	for _, id := range []int{0, -5} {
		_, err := NewGetPassInput(id)
		assertValidationError(t, err, "id")
	}
}

func TestGetPassUseCase_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewGetPassUseCase(repo)
	in := MustNewGetPassInput(1)
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "get pass 1")
}
