package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingActivitiesInput is the paginated request for listing
// activities scheduled at or after the current time.
type ListUpcomingActivitiesInput struct {
	limit  int
	offset int
}

func (in ListUpcomingActivitiesInput) Limit() int  { return in.limit }
func (in ListUpcomingActivitiesInput) Offset() int { return in.offset }

// NewListUpcomingActivitiesInput normalizes pagination. Error is
// always nil; the factory exists for uniform signature.
func NewListUpcomingActivitiesInput(limit, offset int) (ListUpcomingActivitiesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListUpcomingActivitiesInput{limit: limit, offset: offset}, nil
}

// MustNewListUpcomingActivitiesInput panics on error. For tests.
func MustNewListUpcomingActivitiesInput(limit, offset int) ListUpcomingActivitiesInput {
	in, err := NewListUpcomingActivitiesInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListUpcomingActivitiesOutput struct {
	Activities []*domain.Activity
}

type ListUpcomingActivitiesUseCase struct {
	repo domain.ActivityRepository
}

func NewListUpcomingActivitiesUseCase(repo domain.ActivityRepository) *ListUpcomingActivitiesUseCase {
	return &ListUpcomingActivitiesUseCase{repo: repo}
}

func (uc *ListUpcomingActivitiesUseCase) Execute(ctx context.Context, input ListUpcomingActivitiesInput) (ListUpcomingActivitiesOutput, error) {
	activities, err := uc.repo.ListUpcoming(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListUpcomingActivitiesOutput{}, fmt.Errorf("list upcoming activities: %w", err)
	}
	return ListUpcomingActivitiesOutput{Activities: activities}, nil
}
