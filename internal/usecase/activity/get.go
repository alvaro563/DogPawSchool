package activity

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// GetActivityInput is the validated input for fetching a single
// activity by id.
type GetActivityInput struct {
	id int
}

func (in GetActivityInput) ID() int { return in.id }

// NewGetActivityInput validates id > 0.
func NewGetActivityInput(id int) (GetActivityInput, error) {
	if id <= 0 {
		return GetActivityInput{}, &ValidationError{Field: "id"}
	}
	return GetActivityInput{id: id}, nil
}

// MustNewGetActivityInput panics on validation error. For tests.
func MustNewGetActivityInput(id int) GetActivityInput {
	in, err := NewGetActivityInput(id)
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
	activity, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return GetActivityOutput{}, fmt.Errorf("get activity %d: %w", input.ID(), err)
	}
	if activity == nil {
		return GetActivityOutput{}, ErrNotFound
	}
	return GetActivityOutput{Activity: activity}, nil
}
