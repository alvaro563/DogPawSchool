package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingActivitiesInput is the paginated request for listing
// activities scheduled at or after the current time.
type ListUpcomingActivitiesInput struct {
	Limit  int
	Offset int
}

// ListUpcomingActivitiesOutput carries the result page, soonest first.
type ListUpcomingActivitiesOutput struct {
	Activities []*domain.Activity
}

// ListUpcomingActivitiesUseCase returns a paginated list of upcoming
// activities, soonest first.
type ListUpcomingActivitiesUseCase struct {
	repo domain.ActivityRepository
}

func NewListUpcomingActivitiesUseCase(repo domain.ActivityRepository) *ListUpcomingActivitiesUseCase {
	return &ListUpcomingActivitiesUseCase{repo: repo}
}

func (uc *ListUpcomingActivitiesUseCase) Execute(ctx context.Context, input ListUpcomingActivitiesInput) (ListUpcomingActivitiesOutput, error) {
	limit, offset := NormalizePagination(input.Limit, input.Offset)
	activities, err := uc.repo.ListUpcoming(ctx, limit, offset)
	if err != nil {
		return ListUpcomingActivitiesOutput{}, fmt.Errorf("list upcoming activities: %w", err)
	}
	return ListUpcomingActivitiesOutput{Activities: activities}, nil
}
