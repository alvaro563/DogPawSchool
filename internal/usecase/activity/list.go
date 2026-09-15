package activity

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// ListAllActivitiesInput is the paginated request for listing
// activities. Pagination is normalized by the factory.
//
// Filters (all optional, combined with AND):
//
//   - from / to:      *time.Time boundaries. nil = no bound on that side.
//                     Repo passes each as `nullableTime(*T)` so a nil
//                     produces SQL NULL and the
//                     `$N::timestamptz IS NULL` predicate short-circuits
//                     the comparison (avoids the zero-time gotcha
//                     where `time.Time{}` would serialise as
//                     `0001-01-01 00:00:00 UTC` and silently filter
//                     everything out).
//   - closedFilter:   *bool. nil = both open and closed. true = closed
//                     only. false = open only. Maps the ?closed=true|false
//                     query param.
//
// Viewer (the authenticated caller): required, drives the SQL
// visibility filter. Admin (is_admin=true) sees every row; non-admin
// users only see group activities and individual activities for
// dogs they own. The repo ALWAYS applies the visibility filter
// (even on the no-bound, no-closed path) so every read endpoint
// stays consistent.
//
// `from` is INCLUSIVE and `to` is INCLUSIVE in this listing. (Different
// from the reservation CreatedAt endpoints, which use the [from, to)
// half-open convention.)
type ListAllActivitiesInput struct {
	limit         int
	offset        int
	from          *time.Time
	to            *time.Time
	closedFilter  *bool
	viewerUserID  int
	viewerIsAdmin bool
}

func (in ListAllActivitiesInput) Limit() int         { return in.limit }
func (in ListAllActivitiesInput) Offset() int        { return in.offset }
func (in ListAllActivitiesInput) From() *time.Time   { return in.from }
func (in ListAllActivitiesInput) To() *time.Time     { return in.to }
func (in ListAllActivitiesInput) ClosedFilter() *bool {
	return in.closedFilter
}
func (in ListAllActivitiesInput) ViewerUserID() int  { return in.viewerUserID }
func (in ListAllActivitiesInput) ViewerIsAdmin() bool { return in.viewerIsAdmin }

// NewListAllActivitiesInput normalizes pagination: limit <= 0 falls
// back to the default 50, limit > 100 is capped, negative offset is
// reset to 0. Pass nil for from/to/closedFilter to omit each filter.
// Errors when from > to (both non-nil). viewerUserID and viewerIsAdmin
// are required so the SQL visibility filter can be applied at the
// repo level (use IsAdmin() on the Gin context; never pass 0/false
// unless the endpoint is genuinely admin-only).
func NewListAllActivitiesInput(
	limit, offset int,
	from, to *time.Time,
	closedFilter *bool,
	viewerUserID int,
	viewerIsAdmin bool,
) (ListAllActivitiesInput, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if from != nil && to != nil && from.After(*to) {
		return ListAllActivitiesInput{}, &ValidationError{Field: "from"}
	}
	return ListAllActivitiesInput{
		limit:         limit,
		offset:        offset,
		from:          from,
		to:            to,
		closedFilter:  closedFilter,
		viewerUserID:  viewerUserID,
		viewerIsAdmin: viewerIsAdmin,
	}, nil
}

// MustNewListAllActivitiesInput is the canonical test helper for
// ListAllActivitiesInput. It accepts the same positional arguments
// as the factory and panics on error.
func MustNewListAllActivitiesInput(
	limit, offset int,
	from, to *time.Time,
	closedFilter *bool,
	viewerUserID int,
	viewerIsAdmin bool,
) ListAllActivitiesInput {
	in, err := NewListAllActivitiesInput(limit, offset, from, to, closedFilter, viewerUserID, viewerIsAdmin)
	if err != nil {
		panic(err)
	}
	return in
}

// MustNewListAllActivitiesInputAdmin is a tiny convenience wrapper
// that calls the factory with the admin viewer (tests that don't
// care about visibility can use this).
func MustNewListAllActivitiesInputAdmin(limit, offset int) ListAllActivitiesInput {
	return MustNewListAllActivitiesInput(limit, offset, nil, nil, nil, 0, true)
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

// Execute picks the right repo method based on the closedFilter
// branch. All branches thread viewerUserID/viewerIsAdmin through to
// the repo so the SQL visibility filter runs on every read.
func (uc *ListAllActivitiesUseCase) Execute(ctx context.Context, input ListAllActivitiesInput) (ListAllActivitiesOutput, error) {
	var activities []*domain.Activity
	var err error
	switch {
	case input.closedFilter != nil:
		activities, err = uc.repo.ListByClosed(
			ctx, input.ViewerUserID(), input.ViewerIsAdmin(),
			*input.closedFilter, input.from, input.to, input.Limit(), input.Offset(),
		)
	case input.from != nil || input.to != nil:
		// Date range only — dispatch to ListByDateRange. The repo
		// still applies the visibility filter.
		var fromTime, toTime time.Time
		if input.from != nil {
			fromTime = *input.from
		}
		if input.to != nil {
			toTime = *input.to
		}
		activities, err = uc.repo.ListByDateRange(
			ctx, input.ViewerUserID(), input.ViewerIsAdmin(),
			fromTime, toTime, input.Limit(), input.Offset(),
		)
	default:
		activities, err = uc.repo.List(
			ctx, input.ViewerUserID(), input.ViewerIsAdmin(),
			input.Limit(), input.Offset(),
		)
	}
	if err != nil {
		return ListAllActivitiesOutput{}, fmt.Errorf("list all activities: %w", err)
	}
	return ListAllActivitiesOutput{Activities: activities}, nil
}
