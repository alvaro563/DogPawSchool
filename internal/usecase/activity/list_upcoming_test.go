package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListUpcomingActivitiesUseCase_Success(t *testing.T) {
	future1 := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	future2 := time.Date(2030, 2, 1, 10, 0, 0, 0, time.UTC)
	expected := []*domain.Activity{
		mustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, future1),
		mustNewActivity(2, "b", "l", domain.TypeRoute, 5, 1, future2),
	}
	repo := &mockActivityRepository{
		listUpcoming: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}
	uc := NewListUpcomingActivitiesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListUpcomingActivitiesInput{})
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Activities)
}

func TestListUpcomingActivitiesUseCase_Empty(t *testing.T) {
	repo := &mockActivityRepository{
		listUpcoming: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			return nil, nil
		},
	}
	uc := NewListUpcomingActivitiesUseCase(repo)
	output, err := uc.Execute(context.Background(), ListUpcomingActivitiesInput{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.Activities))
}

func TestListUpcomingActivitiesUseCase_RepoError(t *testing.T) {
	repo := &mockActivityRepository{
		listUpcoming: func(ctx context.Context, limit, offset int) ([]*domain.Activity, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListUpcomingActivitiesUseCase(repo)
	_, err := uc.Execute(context.Background(), ListUpcomingActivitiesInput{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list upcoming activities")
}
