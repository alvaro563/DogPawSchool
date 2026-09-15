package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetActivityInput is the validated input for fetching a single
// activity by id.
//
// viewerUserID + viewerIsAdmin drive the SQL-level visibility filter.
// Admin viewers see every row; non-admin viewers only see group
// activities and individual activities for dogs they own.
type GetActivityInput struct {
	id             int
	viewerUserID   int
	viewerIsAdmin  bool
}

func (in GetActivityInput) ID() int               { return in.id }
func (in GetActivityInput) ViewerUserID() int     { return in.viewerUserID }
func (in GetActivityInput) ViewerIsAdmin() bool  { return in.viewerIsAdmin }

// NewGetActivityInput validates id > 0.
func NewGetActivityInput(id, viewerUserID int, viewerIsAdmin bool) (GetActivityInput, error) {
	if id <= 0 {
		return GetActivityInput{}, &ValidationError{Field: "id"}
	}
	return GetActivityInput{id: id, viewerUserID: viewerUserID, viewerIsAdmin: viewerIsAdmin}, nil
}

// MustNewGetActivityInput panics on validation error. For tests.
func MustNewGetActivityInput(id, viewerUserID int, viewerIsAdmin bool) GetActivityInput {
	in, err := NewGetActivityInput(id, viewerUserID, viewerIsAdmin)
	if err != nil {
		panic(err)
	}
	return in
}

// GetActivityOutput carries the requested activity.
type GetActivityOutput struct {
	Activity *domain.Activity
}

// GetActivityUseCase returns a single activity or ErrNotFound.
type GetActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewGetActivityUseCase(repo domain.ActivityRepository) *GetActivityUseCase {
	return &GetActivityUseCase{repo: repo}
}

func (uc *GetActivityUseCase) Execute(ctx context.Context, input GetActivityInput) (GetActivityOutput, error) {
	activity, err := uc.repo.GetByID(ctx, input.ID(), input.ViewerUserID(), input.ViewerIsAdmin())
	if err != nil {
		return GetActivityOutput{}, fmt.Errorf("get activity %d: %w", input.ID(), err)
	}
	if activity == nil {
		return GetActivityOutput{}, ErrNotFound
	}
	return GetActivityOutput{Activity: activity}, nil
}
