package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetActivityInput is the input for fetching a single activity by id.
type GetActivityInput struct {
	ID int
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
	if input.ID <= 0 {
		return GetActivityOutput{}, &ValidationError{Field: "id"}
	}
	activity, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return GetActivityOutput{}, fmt.Errorf("get activity %d: %w", input.ID, err)
	}
	if activity == nil {
		return GetActivityOutput{}, ErrNotFound
	}
	return GetActivityOutput{Activity: activity}, nil
}
