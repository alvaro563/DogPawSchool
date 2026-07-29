package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListByUserPassesUseCase_Success(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	expected := []*domain.Pass{
		domain.MustNewPass(1, 10, 10, 100, domain.PassGeneric, 1, now, now, nil),
		domain.MustNewPass(3, 5, 5, 50, domain.PassSpecial, 1, now, now, nil),
	}
	repo := &mockPassRepository{
		listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
			assert.Equal(t, 1, userID)
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}
	uc := NewListByUserPassesUseCase(repo)
	in := MustNewListByUserPassesInput(1, 0, 0)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Passes)
}

func TestListByUserPassesUseCase_Empty(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
			return nil, nil
		},
	}
	uc := NewListByUserPassesUseCase(repo)
	in := MustNewListByUserPassesInput(999, 0, 0)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.Passes))
}

func TestNewListByUserPassesInput_InvalidUserID(t *testing.T) {
	t.Parallel()
	for _, userID := range []int{0, -5} {
		_, err := NewListByUserPassesInput(userID, 0, 0)
		assertValidationError(t, err, "user_id")
	}
}

func TestListByUserPassesUseCase_PaginationNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		wantRepoLimit  int
		wantRepoOffset int
	}{
		{"zero_limit_becomes_default", 0, 0, 50, 0},
		{"over_max_is_capped", 500, 0, 100, 0},
		{"negative_offset_is_zero", 25, -10, 25, 0},
		{"custom_values_preserved", 10, 5, 10, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPassRepository{
				listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
					assert.Equal(t, tt.wantRepoLimit, limit)
					assert.Equal(t, tt.wantRepoOffset, offset)
					return nil, nil
				},
			}
			uc := NewListByUserPassesUseCase(repo)
			in := MustNewListByUserPassesInput(1, tt.inputLimit, tt.inputOffset)
			_, err := uc.Execute(context.Background(), in)
			assert.NoError(t, err)
		})
	}
}

func TestListByUserPassesUseCase_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListByUserPassesUseCase(repo)
	in := MustNewListByUserPassesInput(1, 0, 0)
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list passes by user 1")
}
