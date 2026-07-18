package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListAllActivitiesUseCase_Success(t *testing.T) {
	date1 := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	expected := []*domain.Activity{
		mustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, date1),
		mustNewActivity(2, "b", "l", domain.TypeRoute, 5, 1, date2),
		mustNewActivity(3, "c", "l", domain.TypeRoute, 5, 1, date3),
	}
	repo := &mockActivityRepository{
		list: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListAllActivitiesInput{})
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Activities)
}

func TestListAllActivitiesUseCase_Empty(t *testing.T) {
	repo := &mockActivityRepository{
		list: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			return nil, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListAllActivitiesInput{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.Activities))
}

func TestListAllActivitiesUseCase_PaginationNormalization(t *testing.T) {
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
			repo := &mockActivityRepository{
				list: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
					assert.Equal(t, tt.wantRepoLimit, limit)
					assert.Equal(t, tt.wantRepoOffset, offset)
					return nil, nil
				},
			}
			uc := NewListAllActivitiesUseCase(repo)
			_, err := uc.Execute(context.Background(), ListAllActivitiesInput{Limit: tt.inputLimit, Offset: tt.inputOffset})
			assert.NoError(t, err)
		})
	}
}

func TestListAllActivitiesUseCase_RepoError(t *testing.T) {
	repo := &mockActivityRepository{
		list: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	_, err := uc.Execute(context.Background(), ListAllActivitiesInput{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list all activities")
}
