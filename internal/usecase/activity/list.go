package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllActivitiesInput is the paginated request for listing every
// activity in the system.
type ListAllActivitiesInput struct {
	Limit  int
	Offset int
}

// ListAllActivitiesOutput carries the result page.
type ListAllActivitiesOutput struct {
	Activities []*domain.Activity
}

// ListAllActivitiesUseCase returns a paginated list of all
// activities, most recent first.
type ListAllActivitiesUseCase struct {
	repo domain.ActivityRepository
}

func NewListAllActivitiesUseCase(repo domain.ActivityRepository) *ListAllActivitiesUseCase {
	return &ListAllActivitiesUseCase{repo: repo}
}

func (uc *ListAllActivitiesUseCase) Execute(ctx context.Context, input ListAllActivitiesInput) (ListAllActivitiesOutput, error) {
	limit, offset := NormalizePagination(input.Limit, input.Offset)
	activities, err := uc.repo.List(ctx, limit, offset)
	if err != nil {
		return ListAllActivitiesOutput{}, fmt.Errorf("list all activities: %w", err)
	}
	return ListAllActivitiesOutput{Activities: activities}, nil
}
