package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllActivitiesInput is the paginated request for listing every
// activity. The pagination is already normalized by the factory.
type ListAllActivitiesInput struct {
	limit  int
	offset int
}

func (in ListAllActivitiesInput) Limit() int  { return in.limit }
func (in ListAllActivitiesInput) Offset() int { return in.offset }

// NewListAllActivitiesInput normalizes pagination. Error is always
// nil for pure-pagination inputs; it is returned to keep the
// factory signature uniform.
func NewListAllActivitiesInput(limit, offset int) (ListAllActivitiesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListAllActivitiesInput{limit: limit, offset: offset}, nil
}

// MustNewListAllActivitiesInput panics on error. For tests.
func MustNewListAllActivitiesInput(limit, offset int) ListAllActivitiesInput {
	in, err := NewListAllActivitiesInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListAllActivitiesOutput struct {
	Activities []*domain.Activity
}

type ListAllActivitiesUseCase struct {
	repo domain.ActivityRepository
}

func NewListAllActivitiesUseCase(repo domain.ActivityRepository) *ListAllActivitiesUseCase {
	return &ListAllActivitiesUseCase{repo: repo}
}

func (uc *ListAllActivitiesUseCase) Execute(ctx context.Context, input ListAllActivitiesInput) (ListAllActivitiesOutput, error) {
	activities, err := uc.repo.List(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListAllActivitiesOutput{}, fmt.Errorf("list all activities: %w", err)
	}
	return ListAllActivitiesOutput{Activities: activities}, nil
}
