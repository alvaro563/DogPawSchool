package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListAllPassesUseCase_Success(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := []*domain.Pass{
		domain.MustNewPass(1, 10, 10, 100, domain.PassGeneric, 1, now, now, nil),
		domain.MustNewPass(2, 5, 5, 50, domain.PassSpecial, 2, now, now, nil),
	}
	repo := &mockPassRepository{
		listAll: func(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}
	uc := NewListAllPassesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListAllPassesInput{})
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Passes)
}

func TestListAllPassesUseCase_Empty(t *testing.T) {
	repo := &mockPassRepository{
		listAll: func(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
			return nil, nil
		},
	}
	uc := NewListAllPassesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListAllPassesInput{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.Passes))
}

func TestListAllPassesUseCase_PaginationNormalization(t *testing.T) {
	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		wantRepoLimit  int
		wantRepoOffset int
	}{
		{"zero_limit_becomes_default", 0, 0, 50, 0},
		{"negative_limit_becomes_default", -1, 0, 50, 0},
		{"over_max_is_capped", 500, 0, 100, 0},
		{"negative_offset_is_zero", 50, -5, 50, 0},
		{"exact_max_kept", 100, 0, 100, 0},
		{"custom_values_preserved", 25, 10, 25, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPassRepository{
				listAll: func(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
					assert.Equal(t, tt.wantRepoLimit, limit)
					assert.Equal(t, tt.wantRepoOffset, offset)
					return nil, nil
				},
			}
			uc := NewListAllPassesUseCase(repo)
			_, err := uc.Execute(context.Background(), ListAllPassesInput{Limit: tt.inputLimit, Offset: tt.inputOffset})
			assert.NoError(t, err)
		})
	}
}

func TestListAllPassesUseCase_RepoError(t *testing.T) {
	repo := &mockPassRepository{
		listAll: func(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListAllPassesUseCase(repo)
	_, err := uc.Execute(context.Background(), ListAllPassesInput{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list all passes")
}
