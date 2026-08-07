package activity

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterActivityInput is the validated command to create a new
// school activity. All fields are private: only NewRegisterActivityInput
// can construct one.
type RegisterActivityInput struct {
	name            string
	description     string
	location        string
	activityType    domain.ActivityType
	maxCapacity     int
	durationInHours int
	date            time.Time
}

func (in RegisterActivityInput) Name() string                      { return in.name }
func (in RegisterActivityInput) Description() string               { return in.description }
func (in RegisterActivityInput) Location() string                  { return in.location }
func (in RegisterActivityInput) ActivityType() domain.ActivityType { return in.activityType }
func (in RegisterActivityInput) MaxCapacity() int                  { return in.maxCapacity }
func (in RegisterActivityInput) DurationInHours() int              { return in.durationInHours }
func (in RegisterActivityInput) Date() time.Time                   { return in.date }

// NewRegisterActivityInput is the validating factory. Returns the
// first *ValidationError encountered. The returned input is, by
// construction, always valid.
func NewRegisterActivityInput(
	name, description, location string,
	activityType domain.ActivityType,
	maxCapacity, durationInHours int,
	date time.Time,
) (RegisterActivityInput, error) {
	if name == "" {
		return RegisterActivityInput{}, &ValidationError{Field: "name"}
	}
	if location == "" {
		return RegisterActivityInput{}, &ValidationError{Field: "location"}
	}
	if !activityType.IsValid() {
		return RegisterActivityInput{}, &ValidationError{Field: "activity_type"}
	}
	if maxCapacity <= 0 {
		return RegisterActivityInput{}, &ValidationError{Field: "max_capacity"}
	}
	if durationInHours <= 0 {
		return RegisterActivityInput{}, &ValidationError{Field: "duration_in_hours"}
	}
	if date.IsZero() {
		return RegisterActivityInput{}, &ValidationError{Field: "date"}
	}
	return RegisterActivityInput{
		name: name, description: description, location: location, activityType: activityType,
		maxCapacity: maxCapacity, durationInHours: durationInHours, date: date,
	}, nil
}

// MustNewRegisterActivityInput panics on validation error. For tests.
func MustNewRegisterActivityInput(
	name, description, location string,
	activityType domain.ActivityType,
	maxCapacity, durationInHours int,
	date time.Time,
) RegisterActivityInput {
	in, err := NewRegisterActivityInput(name, description, location, activityType, maxCapacity, durationInHours, date)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterActivityOutput is the result of a successful create.
type RegisterActivityOutput struct {
	ID int
}

// RegisterActivityUseCase creates a new activity in the system.
type RegisterActivityUseCase struct {
	repo domain.ActivityRepository
}

func NewRegisterActivityUseCase(repo domain.ActivityRepository) *RegisterActivityUseCase {
	return &RegisterActivityUseCase{repo: repo}
}

func (uc *RegisterActivityUseCase) Execute(ctx context.Context, input RegisterActivityInput) (RegisterActivityOutput, error) {
	activity, err := domain.NewActivity(0, input.Name(), input.Description(), input.Location(), input.ActivityType(),
		input.MaxCapacity(), input.DurationInHours(), input.Date())
	if err != nil {
		return RegisterActivityOutput{}, err
	}
	id, err := uc.repo.Create(ctx, activity)
	if err != nil {
		return RegisterActivityOutput{}, fmt.Errorf("register activity: %w", err)
	}
	return RegisterActivityOutput{ID: id}, nil
}
