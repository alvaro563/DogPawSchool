package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestDog(t *testing.T, ageInMonths int, sex domain.Sex, weightKg float64, neutered bool) *domain.Dog {
	t.Helper()
	d, err := domain.NewDog(1, "Test", "TestBreed", "TEST123", ageInMonths, sex, weightKg, 1)
	if err != nil {
		t.Fatalf("newTestDog: %v", err)
	}
	if neutered {
		_ = d.UpdateProfile(domain.UpdateDogInput{Neutered: true})
	}
	return d
}

func TestNewDog(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		d, err := domain.NewDog(1, "Buddy", "Labrador", "ES12345", 24, domain.SexMale, 25.0, 1)
		assert.NoError(t, err)
		assert.NotNil(t, d)
		assert.Equal(t, 1, d.ID())
		assert.Equal(t, "Buddy", d.Name())
		assert.True(t, d.IsActive())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			n         string
			b         string
			p         string
			age       int
			sex       domain.Sex
			weight    float64
			userID    int
			wantInErr string
		}{
			{"negative_id", -1, "n", "b", "p", 24, domain.SexMale, 10, 1, "id must not be negative"},
			{"empty_name", 1, "", "b", "p", 24, domain.SexMale, 10, 1, "name must not be empty"},
			{"empty_breed", 1, "n", "", "p", 24, domain.SexMale, 10, 1, "breed must not be empty"},
			{"empty_passport", 1, "n", "b", "", 24, domain.SexMale, 10, 1, "passport must not be empty"},
			{"negative_age", 1, "n", "b", "p", -1, domain.SexMale, 10, 1, "ageInMonths must not be negative"},
			{"negative_weight", 1, "n", "b", "p", 24, domain.SexMale, -1, 1, "weightKg must not be negative"},
			{"invalid_sex", 1, "n", "b", "p", 24, domain.Sex(""), 10, 1, "invalid sex"},
			{"zero_user_id", 1, "n", "b", "p", 24, domain.SexMale, 10, 0, "userID must be greater than 0"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewDog(tt.id, tt.n, tt.b, tt.p, tt.age, tt.sex, tt.weight, tt.userID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestDog_AgeBracket(t *testing.T) {
	tests := []struct {
		name     string
		age      int
		expected domain.AgeBracket
	}{
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
			d := newTestDog(t, tt.age, domain.SexMale, 10.0, false)
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
		{"tiny_puppy_is_mini", 0.1, domain.SizeBracketMini},
		{"mini_upper_boundary", 5.0, domain.SizeBracketMini},
		{"medium_lower_boundary", 5.1, domain.SizeBracketMedium},
		{"medium_upper_boundary", 20.0, domain.SizeBracketMedium},
		{"large_lower_boundary", 20.1, domain.SizeBracketLarge},
		{"giant_dog", 80.0, domain.SizeBracketLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDog(t, 24, domain.SexMale, tt.weight, false)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDog(t, 24, tt.sex, 10.0, tt.neutered)
			assert.Equal(t, tt.expected, d.IsIntactMale())
		})
	}
}

func TestDog_AddIncompatibility(t *testing.T) {
	incompat, _ := domain.NewIncompatibility(1, "Reactivo a machos", domain.IncompatibilityLevelAbsoluta)
	incompat2, _ := domain.NewIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia)

	t.Run("happy_path_adds", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		added, err := d.AddIncompatibility(incompat)
		assert.NoError(t, err)
		assert.True(t, added)
		assert.Len(t, d.Incompatibilities(), 1)
	})

	t.Run("nil_incompat_returns_error", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		added, err := d.AddIncompatibility(nil)
		assert.Error(t, err)
		assert.False(t, added)
	})

	t.Run("idempotent_when_already_present", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		_, _ = d.AddIncompatibility(incompat)
		added, err := d.AddIncompatibility(incompat)
		assert.NoError(t, err)
		assert.False(t, added)
		assert.Len(t, d.Incompatibilities(), 1)
	})

	t.Run("multiple_distinct", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		_, _ = d.AddIncompatibility(incompat)
		added, err := d.AddIncompatibility(incompat2)
		assert.NoError(t, err)
		assert.True(t, added)
		assert.Len(t, d.Incompatibilities(), 2)
	})
}

func TestDog_RemoveIncompatibility(t *testing.T) {
	incompat, _ := domain.NewIncompatibility(1, "Reactivo a machos", domain.IncompatibilityLevelAbsoluta)
	incompat2, _ := domain.NewIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia)

	t.Run("happy_path_removes", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		_, _ = d.AddIncompatibility(incompat)
		removed, err := d.RemoveIncompatibility(1)
		assert.NoError(t, err)
		assert.True(t, removed)
		assert.Empty(t, d.Incompatibilities())
	})

	t.Run("zero_id_returns_error", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		removed, err := d.RemoveIncompatibility(0)
		assert.Error(t, err)
		assert.False(t, removed)
	})

	t.Run("idempotent_when_not_present", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		_, _ = d.AddIncompatibility(incompat)
		removed, err := d.RemoveIncompatibility(99)
		assert.NoError(t, err)
		assert.False(t, removed)
		assert.Len(t, d.Incompatibilities(), 1)
	})

	t.Run("preserves_other_incompatibilities", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		_, _ = d.AddIncompatibility(incompat)
		_, _ = d.AddIncompatibility(incompat2)
		removed, _ := d.RemoveIncompatibility(1)
		assert.True(t, removed)
		assert.Len(t, d.Incompatibilities(), 1)
		assert.Equal(t, 2, d.Incompatibilities()[0].ID())
	})
}

func TestDog_UpdateProfile(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, false)
		err := d.UpdateProfile(domain.UpdateDogInput{
			Neutered:      true,
			Heat:          true,
			WeightKg:      30.0,
			PhotoURL:      "url",
			MedicalNotes:  "notes",
			EducatorNotes: "edu",
			IsActive:      false,
		})
		assert.NoError(t, err)
		assert.True(t, d.Neutered())
		assert.True(t, d.Heat())
		assert.Equal(t, 30.0, d.WeightKg())
		assert.Equal(t, "url", d.PhotoURL())
		assert.Equal(t, "notes", d.MedicalNotes())
		assert.Equal(t, "edu", d.EducatorNotes())
		assert.False(t, d.IsActive())
	})

	t.Run("negative_weight_returns_error", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, false)
		err := d.UpdateProfile(domain.UpdateDogInput{WeightKg: -1})
		assert.Error(t, err)
	})
}

func TestDog_Activate_Deactivate(t *testing.T) {
	t.Run("activate_sets_isactive_true", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 10.0, false)
		d.Deactivate()
		assert.False(t, d.IsActive())
		d.Activate()
		assert.True(t, d.IsActive())
	})
}

func TestSex_IsValid(t *testing.T) {
	assert.True(t, domain.SexMale.IsValid())
	assert.True(t, domain.SexFemale.IsValid())
	assert.False(t, domain.Sex("").IsValid())
	assert.False(t, domain.Sex("UNKNOWN").IsValid())
}
