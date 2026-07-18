package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterActivityInput is the validated payload for creating a new
// school activity.
type RegisterActivityInput struct {
	Name            string
	Location        string
	ActivityType    domain.ActivityType
	MaxCapacity     int
	DurationInHours int
	Date            time.Time
}

// RegisterActivityOutput is the result of a successful create.
type RegisterActivityOutput struct {
	ID int
}

// RegisterActivityUseCase creates a new activity in the system. It
// validates the input, builds a domain.Activity, and asks the
// repository to persist it.
type RegisterActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewRegisterActivityUseCase(repo domain.ActivityRepository) *RegisterActivityUseCase {
	return &RegisterActivityUseCase{repo: repo}
}

func (uc *RegisterActivityUseCase) Execute(ctx context.Context, input RegisterActivityInput) (RegisterActivityOutput, error) {
	if err := input.validate(); err != nil {
		return RegisterActivityOutput{}, err
	}

	activity, err := domain.NewActivity(0, input.Name, input.Location, input.ActivityType, input.MaxCapacity, input.DurationInHours, input.Date)
	if err != nil {
		return RegisterActivityOutput{}, err
	}

	id, err := uc.repo.Create(ctx, activity)
	if err != nil {
		return RegisterActivityOutput{}, fmt.Errorf("register activity: %w", err)
	}
	return RegisterActivityOutput{ID: id}, nil
}

func (input RegisterActivityInput) validate() error {
	if input.Name == "" {
		return &ValidationError{Field: "name"}
	}
	if input.Location == "" {
		return &ValidationError{Field: "location"}
	}
	if !input.ActivityType.IsValid() {
		return &ValidationError{Field: "activity_type"}
	}
	if input.MaxCapacity <= 0 {
		return &ValidationError{Field: "max_capacity"}
	}
	if input.DurationInHours <= 0 {
		return &ValidationError{Field: "duration_in_hours"}
	}
	if input.Date.IsZero() {
		return &ValidationError{Field: "date"}
	}
	return nil
}

// sentinelErr is a small, import-free error used in tests to verify
// that repository errors are wrapped correctly.
var sentinelErr = errors.New("repo failure")
