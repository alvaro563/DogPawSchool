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
	assert.False(t, domain.ActivityType("").IsValid())
	assert.False(t, domain.ActivityType("OTHER").IsValid())
}
