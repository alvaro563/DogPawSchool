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
	return MustNewRegisterActivityInput(
		"Paseo Río", "Parking Central",
		domain.TypeRoute, 8, 2,
		time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
	)
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

func TestNewRegisterActivityInput(t *testing.T) {
	fixedDate := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	base := func() RegisterActivityInput {
		return MustNewRegisterActivityInput("n", "l", domain.TypeRoute, 8, 2, fixedDate)
	}

	scenarios := []struct {
		name      string
		factory   func() (RegisterActivityInput, error)
		wantField string
	}{
		{"empty_name", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("", "l", domain.TypeRoute, 8, 2, fixedDate)
		}, "name"},
		{"empty_location", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", domain.TypeRoute, 8, 2, fixedDate)
		}, "location"},
		{"invalid_type", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "l", domain.ActivityType("INVALID"), 8, 2, fixedDate)
		}, "activity_type"},
		{"zero_capacity", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "l", domain.TypeRoute, 0, 2, fixedDate)
		}, "max_capacity"},
		{"negative_capacity", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "l", domain.TypeRoute, -1, 2, fixedDate)
		}, "max_capacity"},
		{"zero_duration", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "l", domain.TypeRoute, 8, 0, fixedDate)
		}, "duration_in_hours"},
		{"zero_date", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "l", domain.TypeRoute, 8, 2, time.Time{})
		}, "date"},
	}
	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.factory()
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.wantField, verr.Field)
		})
	}
	_ = base // silence unused if the slice above is empty in some refactor
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
