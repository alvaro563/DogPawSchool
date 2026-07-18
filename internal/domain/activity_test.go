package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewActivity(t *testing.T) {
	date := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

	t.Run("happy_path", func(t *testing.T) {
		a, err := domain.NewActivity(1, "Paseo Río", "Parking Central", domain.TypeRoute, 8, 2, date)
		assert.NoError(t, err)
		assert.Equal(t, 1, a.ID())
		assert.Equal(t, "Paseo Río", a.Name())
		assert.Equal(t, 8, a.MaxCapacity())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			n         string
			loc       string
			at        domain.ActivityType
			cap       int
			dur       int
			date      time.Time
			wantInErr string
		}{
			{"negative_id", -1, "n", "l", domain.TypeRoute, 8, 2, date, "id must not be negative"},
			{"empty_name", 1, "", "l", domain.TypeRoute, 8, 2, date, "name must not be empty"},
			{"empty_location", 1, "n", "", domain.TypeRoute, 8, 2, date, "location must not be empty"},
			{"invalid_type", 1, "n", "l", domain.ActivityType("INVALID"), 8, 2, date, "invalid activityType"},
			{"zero_capacity", 1, "n", "l", domain.TypeRoute, 0, 2, date, "maxCapacity must be greater than 0"},
			{"zero_duration", 1, "n", "l", domain.TypeRoute, 8, 0, date, "durationInHours must be greater than 0"},
			{"zero_date", 1, "n", "l", domain.TypeRoute, 8, 2, time.Time{}, "date must be a valid time"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewActivity(tt.id, tt.n, tt.loc, tt.at, tt.cap, tt.dur, tt.date)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestMustNewActivity(t *testing.T) {
	date := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

	t.Run("happy_path", func(t *testing.T) {
		activity := domain.MustNewActivity(1, "Paseo Río", "Parking Central", domain.TypeRoute, 8, 2, date)
		assert.NotNil(t, activity)
		assert.Equal(t, 1, activity.ID())
		assert.Equal(t, "Paseo Río", activity.Name())
	})

	t.Run("panics_on_invalid_input", func(t *testing.T) {
		assert.Panics(t, func() {
			domain.MustNewActivity(1, "", "l", domain.TypeRoute, 8, 2, date)
		})
	})
}

func TestActivity_IsFull(t *testing.T) {
	date := time.Now()
	a, _ := domain.NewActivity(1, "n", "l", domain.TypeRoute, 5, 2, date)
	assert.False(t, a.IsFull(0))
	assert.False(t, a.IsFull(4))
	assert.True(t, a.IsFull(5))
	assert.True(t, a.IsFull(6))
}

func TestActivity_IsInThePast_IsUpcoming(t *testing.T) {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ap, _ := domain.NewActivity(1, "n", "l", domain.TypeRoute, 5, 2, past)
	af, _ := domain.NewActivity(2, "n", "l", domain.TypeRoute, 5, 2, future)

	assert.True(t, ap.IsInThePast(now))
	assert.False(t, ap.IsUpcoming(now))
	assert.False(t, af.IsInThePast(now))
	assert.True(t, af.IsUpcoming(now))
}

func TestActivity_TypePredicates(t *testing.T) {
	date := time.Now()
	individual, _ := domain.NewActivity(1, "n", "l", domain.TypeIndividual, 1, 1, date)
	social, _ := domain.NewActivity(2, "n", "l", domain.TypeSocialization, 5, 1, date)
	route, _ := domain.NewActivity(3, "n", "l", domain.TypeRoute, 8, 2, date)

	assert.True(t, individual.IsIndividualClass())
	assert.False(t, individual.IsSocializationGroup())
	assert.False(t, individual.IsRoute())

	assert.True(t, social.IsSocializationGroup())
	assert.False(t, social.IsIndividualClass())
	assert.False(t, social.IsRoute())

	assert.True(t, route.IsRoute())
	assert.False(t, route.IsIndividualClass())
	assert.False(t, route.IsSocializationGroup())
}

func TestActivityType_IsValid(t *testing.T) {
	assert.True(t, domain.TypeSocialization.IsValid())
	assert.True(t, domain.TypeRoute.IsValid())
	assert.True(t, domain.TypeIndividual.IsValid())
	assert.True(t, domain.TypeExtra.IsValid())
	assert.False(t, domain.ActivityType("").IsValid())
	assert.False(t, domain.ActivityType("OTHER").IsValid())
}

func TestActivity_ApplyPatch(t *testing.T) {
	originalDate := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	newDate := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)

	t.Run("empty_patch_is_noop", func(t *testing.T) {
		activity := domain.MustNewActivity(1, "Paseo", "Central", domain.TypeRoute, 8, 2, originalDate)
		err := activity.ApplyPatch(domain.ActivityPatch{})
		assert.NoError(t, err)
		assert.Equal(t, "Paseo", activity.Name())
		assert.Equal(t, "Central", activity.Location())
		assert.Equal(t, 8, activity.MaxCapacity())
	})

	t.Run("applies_all_fields", func(t *testing.T) {
		activity := domain.MustNewActivity(1, "Paseo", "Central", domain.TypeRoute, 8, 2, originalDate)
		newName := "Paseo Largo"
		newLocation := "Río"
		newType := domain.TypeSocialization
		newCapacity := 15
		newDuration := 3
		patch := domain.ActivityPatch{
			Name:            &newName,
			Location:        &newLocation,
			ActivityType:    &newType,
			MaxCapacity:     &newCapacity,
			DurationInHours: &newDuration,
			Date:            &newDate,
		}
		err := activity.ApplyPatch(patch)
		assert.NoError(t, err)
		assert.Equal(t, "Paseo Largo", activity.Name())
		assert.Equal(t, "Río", activity.Location())
		assert.Equal(t, domain.TypeSocialization, activity.Type())
		assert.Equal(t, 15, activity.MaxCapacity())
		assert.Equal(t, 3, activity.DurationInHours())
		assert.Equal(t, newDate, activity.Date())
	})

	t.Run("validation_errors", func(t *testing.T) {
		activity := domain.MustNewActivity(1, "Paseo", "Central", domain.TypeRoute, 8, 2, originalDate)
		emptyName := ""
		emptyLocation := ""
		invalidType := domain.ActivityType("INVALID")
		zeroCapacity := 0
		zeroDuration := 0
		zeroDate := time.Time{}
		validName := "Valid"

		tests := []struct {
			name      string
			patch     domain.ActivityPatch
			wantField string
		}{
			{"empty_name", domain.ActivityPatch{Name: &emptyName}, "name"},
			{"empty_location", domain.ActivityPatch{Location: &emptyLocation}, "location"},
			{"invalid_type", domain.ActivityPatch{ActivityType: &invalidType}, "activity_type"},
			{"zero_capacity", domain.ActivityPatch{MaxCapacity: &zeroCapacity}, "max_capacity"},
			{"zero_duration", domain.ActivityPatch{DurationInHours: &zeroDuration}, "duration_in_hours"},
			{"zero_date", domain.ActivityPatch{Date: &zeroDate}, "date"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := activity.ApplyPatch(tt.patch)
				assert.Error(t, err)
				var validationErr *domain.ActivityValidationError
				assert.ErrorAs(t, err, &validationErr)
				assert.Equal(t, tt.wantField, validationErr.Field)
			})
		}

		// Activity should not have been mutated by any failed patch.
		assert.Equal(t, "Paseo", activity.Name())
		assert.Equal(t, "Central", activity.Location())
		assert.Equal(t, 8, activity.MaxCapacity())
		assert.Equal(t, 2, activity.DurationInHours())
		assert.Equal(t, originalDate, activity.Date())

		// A valid patch should still work after the failed ones.
		err := activity.ApplyPatch(domain.ActivityPatch{Name: &validName})
		assert.NoError(t, err)
		assert.Equal(t, "Valid", activity.Name())
	})
}

func TestActivityValidationError_Error(t *testing.T) {
	err := &domain.ActivityValidationError{Field: "name"}
	assert.Equal(t, "activity: invalid value for name", err.Error())
}
