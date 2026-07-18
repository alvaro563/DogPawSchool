package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterActivityInput {
	return RegisterActivityInput{
		Name:            "Paseo Río",
		Location:        "Parking Central",
		ActivityType:    domain.TypeRoute,
		MaxCapacity:     8,
		DurationInHours: 2,
		Date:            time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestRegisterActivityUseCase_Success(t *testing.T) {
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			assert.Equal(t, "Paseo Río", activity.Name())
			assert.Equal(t, domain.TypeRoute, activity.Type())
			assert.Equal(t, 8, activity.MaxCapacity())
			return 42, nil
		},
	}
	uc := NewRegisterActivityUseCase(repo)

	output, err := uc.Execute(context.Background(), validRegisterInput())

	assert.NoError(t, err)
	assert.Equal(t, 42, output.ID)
}

func TestRegisterActivityUseCase_ValidationErrors(t *testing.T) {
	base := validRegisterInput()
	tests := []struct {
		name      string
		mutate    func(input *RegisterActivityInput)
		wantField string
	}{
		{
			name:      "empty_name",
			mutate:    func(i *RegisterActivityInput) { i.Name = "" },
			wantField: "name",
		},
		{
			name:      "empty_location",
			mutate:    func(i *RegisterActivityInput) { i.Location = "" },
			wantField: "location",
		},
		{
			name:      "invalid_type",
			mutate:    func(i *RegisterActivityInput) { i.ActivityType = domain.ActivityType("INVALID") },
			wantField: "activity_type",
		},
		{
			name:      "zero_capacity",
			mutate:    func(i *RegisterActivityInput) { i.MaxCapacity = 0 },
			wantField: "max_capacity",
		},
		{
			name:      "negative_capacity",
			mutate:    func(i *RegisterActivityInput) { i.MaxCapacity = -1 },
			wantField: "max_capacity",
		},
		{
			name:      "zero_duration",
			mutate:    func(i *RegisterActivityInput) { i.DurationInHours = 0 },
			wantField: "duration_in_hours",
		},
		{
			name:      "zero_date",
			mutate:    func(i *RegisterActivityInput) { i.Date = time.Time{} },
			wantField: "date",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)
			repo := &mockActivityRepository{
				create: func(context.Context, *domain.Activity) (int, error) {
					t.Fatal("create should not be called on validation error")
					return 0, nil
				},
			}
			uc := NewRegisterActivityUseCase(repo)
			_, err := uc.Execute(context.Background(), input)
			assert.Error(t, err)
			var validationErr *ValidationError
			assert.True(t, errors.As(err, &validationErr))
			assert.Equal(t, tt.wantField, validationErr.Field)
		})
	}
}

func TestRegisterActivityUseCase_RepoError(t *testing.T) {
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			return 0, sentinelErr
		},
	}
	uc := NewRegisterActivityUseCase(repo)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "register activity")
}
