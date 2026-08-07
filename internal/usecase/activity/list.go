package activity

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ListAllActivitiesInput is the paginated request for listing
// activities. When both From and To are non-zero, the result is
// filtered to activities within [From, To). Pagination is already
// normalized by the factory.
type ListAllActivitiesInput struct {
	limit  int
	offset int
	from   time.Time
	to     time.Time
	filter bool
}

func (in ListAllActivitiesInput) Limit() int  { return in.limit }
func (in ListAllActivitiesInput) Offset() int { return in.offset }

// NewListAllActivitiesInput normalizes pagination. Pass zero values for
// from/to to omit the date-range filter.
func NewListAllActivitiesInput(limit, offset int, from, to time.Time) (ListAllActivitiesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	filter := !from.IsZero() && !to.IsZero()
	if filter && !from.Before(to) {
		return ListAllActivitiesInput{}, &ValidationError{Field: "from"}
	}
	return ListAllActivitiesInput{
		limit:  limit,
		offset: offset,
		from:   from,
		to:     to,
		filter: filter,
	}, nil
}

// MustNewListAllActivitiesInput panics on error. For tests.
func MustNewListAllActivitiesInput(limit, offset int) ListAllActivitiesInput {
	in, err := NewListAllActivitiesInput(limit, offset, time.Time{}, time.Time{})
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
	var activities []*domain.Activity
	var err error
	if input.filter {
		activities, err = uc.repo.ListByDateRange(ctx, input.from, input.to, input.Limit(), input.Offset())
	} else {
		activities, err = uc.repo.List(ctx, input.Limit(), input.Offset())
	}
	if err != nil {
		return ListAllActivitiesOutput{}, fmt.Errorf("list all activities: %w", err)
	}
	return ListAllActivitiesOutput{Activities: activities}, nil
}
