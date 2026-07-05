package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestDog_AgeBracket(t *testing.T) {
	tests := []struct {
		name     string
		age      int
		expected domain.AgeBracket
	}{
		{"negative_age_returns_unknown", -1, domain.AgeBracketUnknown},
		{"zero_months_is_children", 0, domain.AgeBracketChildren},
		{"infant_upper_boundary_is_children", 6, domain.AgeBracketChildren},
		{"teenager_lower_boundary", 7, domain.AgeBracketTeenager},
		{"teenager_upper_boundary", 18, domain.AgeBracketTeenager},
		{"semi_adult_lower_boundary", 19, domain.AgeBracketSemiAdult},
		{"semi_adult_upper_boundary", 36, domain.AgeBracketSemiAdult},
		{"standard_adult", 37, domain.AgeBracketAdult},
		{"senior_dog", 180, domain.AgeBracketAdult},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &domain.Dog{AgeinMonths: tt.age}
			assert.Equal(t, tt.expected, d.AgeBracket())
		})
	}
}

func TestDog_SizeBracket(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected domain.SizeBracket
	}{
		{"zero_weight_returns_unknown", 0, domain.SizeBracketUnknown},
		{"negative_weight_returns_unknown", -2.5, domain.SizeBracketUnknown},
		{"tiny_puppy_is_mini", 0.1, domain.SizeBracketMini},
		{"mini_upper_boundary", 5.0, domain.SizeBracketMini},
		{"medium_lower_boundary", 5.1, domain.SizeBracketMedium},
		{"medium_upper_boundary", 20.0, domain.SizeBracketMedium},
		{"large_lower_boundary", 20.1, domain.SizeBracketLarge},
		{"giant_dog", 80.0, domain.SizeBracketLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &domain.Dog{WeightKg: tt.weight}
			assert.Equal(t, tt.expected, d.SizeBracket())
		})
	}
}

func TestDog_IsIntactMale(t *testing.T) {
	tests := []struct {
		name     string
		sex      domain.Sex
		neutered bool
		expected bool
	}{
		{"intact_male_returns_true", domain.SexMale, false, true},
		{"neutered_male_returns_false", domain.SexMale, true, false},
		{"intact_female_returns_false", domain.SexFemale, false, false},
		{"neutered_female_returns_false", domain.SexFemale, true, false},
		{"unknown_sex_returns_false", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &domain.Dog{Sex: tt.sex, Neutered: tt.neutered}
			assert.Equal(t, tt.expected, d.IsIntactMale())
		})
	}
}
