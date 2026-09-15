package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUpcomingActivitiesInput is the paginated request for listing
// activities scheduled at or after the current time. The SQL
// visibility filter is applied to non-admin viewers.
type ListUpcomingActivitiesInput struct {
	limit         int
	offset        int
	viewerUserID  int
	viewerIsAdmin bool
}

func (in ListUpcomingActivitiesInput) Limit() int         { return in.limit }
func (in ListUpcomingActivitiesInput) Offset() int        { return in.offset }
func (in ListUpcomingActivitiesInput) ViewerUserID() int  { return in.viewerUserID }
func (in ListUpcomingActivitiesInput) ViewerIsAdmin() bool { return in.viewerIsAdmin }

// NewListUpcomingActivitiesInput normalizes pagination. Error is
// always nil; the factory exists for uniform signature.
func NewListUpcomingActivitiesInput(limit, offset int, viewerUserID int, viewerIsAdmin bool) (ListUpcomingActivitiesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListUpcomingActivitiesInput{
		limit:         limit,
		offset:        offset,
		viewerUserID:  viewerUserID,
		viewerIsAdmin: viewerIsAdmin,
	}, nil
}

// MustNewListUpcomingActivitiesInput panics on error. For tests.
func MustNewListUpcomingActivitiesInput(limit, offset int, viewerUserID int, viewerIsAdmin bool) ListUpcomingActivitiesInput {
	in, err := NewListUpcomingActivitiesInput(limit, offset, viewerUserID, viewerIsAdmin)
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
	activities, err := uc.repo.ListUpcoming(
		ctx, input.ViewerUserID(), input.ViewerIsAdmin(),
		input.Limit(), input.Offset(),
	)
	if err != nil {
		return ListUpcomingActivitiesOutput{}, fmt.Errorf("list upcoming activities: %w", err)
	}
	return ListUpcomingActivitiesOutput{Activities: activities}, nil
}
