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
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := domain.MustNewPass(7, 10, 10, 100, domain.PassGeneric, 1, now, now, nil)
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			assert.Equal(t, 7, id)
			return expected, nil
		},
	}
	uc := NewGetPassUseCase(repo)

	output, err := uc.Execute(context.Background(), GetPassInput{ID: 7})

	require.NoError(t, err)
	assert.Same(t, expected, output.Pass)
}

func TestGetPassUseCase_NotFound(t *testing.T) {
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, nil
		},
	}
	uc := NewGetPassUseCase(repo)
	_, err := uc.Execute(context.Background(), GetPassInput{ID: 99})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetPassUseCase_InvalidID(t *testing.T) {
	repo := &mockPassRepository{
		getByID: func(context.Context, int) (*domain.Pass, error) {
			t.Fatal("repo should not be called on invalid id")
			return nil, nil
		},
	}
	uc := NewGetPassUseCase(repo)
	tests := []struct {
		name string
		id   int
	}{
		{"zero", 0},
		{"negative", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), GetPassInput{ID: tt.id})
			assertValidationError(t, err, "id")
		})
	}
}

func TestGetPassUseCase_RepoError(t *testing.T) {
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewGetPassUseCase(repo)
	_, err := uc.Execute(context.Background(), GetPassInput{ID: 1})
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "get pass 1")
}
