package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterActivityInput is the validated command to create a new
// school activity. All fields are private: only NewRegisterActivityInput
// can construct one.
//
// dogID is the target dog id, required when activityType is
// INDIVIDUAL_CLASS. It is optional (nil) for group classes and EXTRA.
type RegisterActivityInput struct {
	name            string
	description     string
	location        string
	activityType    domain.ActivityType
	maxCapacity     int
	durationInHours int
	date            time.Time
	dogID           *int
}

func (in RegisterActivityInput) Name() string                      { return in.name }
func (in RegisterActivityInput) Description() string               { return in.description }
func (in RegisterActivityInput) Location() string                  { return in.location }
func (in RegisterActivityInput) ActivityType() domain.ActivityType { return in.activityType }
func (in RegisterActivityInput) MaxCapacity() int                  { return in.maxCapacity }
func (in RegisterActivityInput) DurationInHours() int              { return in.durationInHours }
func (in RegisterActivityInput) Date() time.Time                   { return in.date }
func (in RegisterActivityInput) DogID() *int                       { return in.dogID }

// NewRegisterActivityInput is the validating factory. The dogID
// pass-through validation: if the caller passes dogID nil and
// activityType==INDIVIDUAL_CLASS, the factory accepts (the domain
// layer will reject); but it pre-trims obvious negatives.
func NewRegisterActivityInput(
	name, description, location string,
	activityType domain.ActivityType,
	maxCapacity, durationInHours int,
	date time.Time,
	dogID *int,
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
	if dogID != nil && *dogID <= 0 {
		return RegisterActivityInput{}, &ValidationError{Field: "dog_id"}
	}
	return RegisterActivityInput{
		name: name, description: description, location: location, activityType: activityType,
		maxCapacity: maxCapacity, durationInHours: durationInHours, date: date,
		dogID: dogID,
	}, nil
}

// MustNewRegisterActivityInput panics on validation error. For tests.
func MustNewRegisterActivityInput(
	name, description, location string,
	activityType domain.ActivityType,
	maxCapacity, durationInHours int,
	date time.Time,
	dogID *int,
) RegisterActivityInput {
	in, err := NewRegisterActivityInput(name, description, location, activityType, maxCapacity, durationInHours, date, dogID)
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
// When dogID is non-nil (an INDIVIDUAL_CLASS), the use case
// additionally verifies that the dog exists and is currently
// active so the activity can be booked.
type RegisterActivityUseCase struct {
	repo     domain.ActivityRepository
	dogRepo  domain.DogRepository
}

func NewRegisterActivityUseCase(repo domain.ActivityRepository, dogRepo domain.DogRepository) *RegisterActivityUseCase {
	return &RegisterActivityUseCase{repo: repo, dogRepo: dogRepo}
}

func (uc *RegisterActivityUseCase) Execute(ctx context.Context, input RegisterActivityInput) (RegisterActivityOutput, error) {
	// Verify the target dog exists and is active whenever a dog is
	// requested. The domain layer also enforces "INDIVIDUAL_CLASS
	// requires a dog", and the DB CHECK constraint mirrors it.
	if input.DogID() != nil {
		dog, err := uc.dogRepo.GetByID(ctx, *input.DogID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return RegisterActivityOutput{}, ErrInvalidDog
			}
			return RegisterActivityOutput{}, fmt.Errorf("lookup dog %d: %w", *input.DogID(), err)
		}
		if dog == nil || !dog.IsActive() {
			return RegisterActivityOutput{}, ErrInactiveDogForActivity
		}
	}

	activity, err := domain.NewActivity(
		0, input.Name(), input.Description(), input.Location(), input.ActivityType(),
		input.MaxCapacity(), input.DurationInHours(), input.Date(), input.DogID())
	if err != nil {
		return RegisterActivityOutput{}, err
	}
	id, err := uc.repo.Create(ctx, activity)
	if err != nil {
		return RegisterActivityOutput{}, fmt.Errorf("register activity: %w", err)
	}
	return RegisterActivityOutput{ID: id}, nil
}
