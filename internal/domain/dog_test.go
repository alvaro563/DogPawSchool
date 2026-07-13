package domain_test

import (
	"errors"
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
		neuteredVal := true
		_ = d.ApplyPatch(domain.DogPatch{Neutered: &neuteredVal})
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

func TestDog_ApplyPatch(t *testing.T) {
	t.Run("empty_patch_is_noop", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, false)
		originalName := d.Name()
		err := d.ApplyPatch(domain.DogPatch{})
		assert.NoError(t, err)
		assert.Equal(t, originalName, d.Name())
		assert.False(t, d.Neutered())
	})

	t.Run("partial_update_preserves_other_fields", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, false)
		newName := "Buddie"
		err := d.ApplyPatch(domain.DogPatch{Name: &newName})
		assert.NoError(t, err)
		assert.Equal(t, "Buddie", d.Name())
		assert.Equal(t, "TestBreed", d.Breed(), "breed preserved")
		assert.Equal(t, 20.0, d.WeightKg(), "weight preserved")
		assert.Equal(t, domain.SexMale, d.Sex(), "sex preserved")
	})

	t.Run("multiple_fields_updated", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, false)
		newName := "Luna"
		newBreed := "Husky"
		newWeight := 25.0
		neutered := true
		err := d.ApplyPatch(domain.DogPatch{
			Name:     &newName,
			Breed:    &newBreed,
			WeightKg: &newWeight,
			Neutered: &neutered,
		})
		assert.NoError(t, err)
		assert.Equal(t, "Luna", d.Name())
		assert.Equal(t, "Husky", d.Breed())
		assert.Equal(t, 25.0, d.WeightKg())
		assert.True(t, d.Neutered())
	})

	t.Run("bool_field_distinguishes_false_from_unset", func(t *testing.T) {
		d := newTestDog(t, 24, domain.SexMale, 20.0, true)
		neutered := false
		err := d.ApplyPatch(domain.DogPatch{Neutered: &neutered})
		assert.NoError(t, err)
		assert.False(t, d.Neutered(), "explicit false must be applied")
	})

	t.Run("validation_errors", func(t *testing.T) {
		empty := ""
		negAge := -1
		negWeight := -1.0
		invalidSex := domain.Sex("OTHER")
		tests := []struct {
			name      string
			patch     domain.DogPatch
			wantField string
		}{
			{"empty_name", domain.DogPatch{Name: &empty}, "name"},
			{"empty_breed", domain.DogPatch{Breed: &empty}, "breed"},
			{"empty_passport", domain.DogPatch{Passport: &empty}, "passport"},
			{"negative_age", domain.DogPatch{AgeInMonths: &negAge}, "age_in_months"},
			{"negative_weight", domain.DogPatch{WeightKg: &negWeight}, "weight_kg"},
			{"invalid_sex", domain.DogPatch{Sex: &invalidSex}, "sex"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				d := newTestDog(t, 24, domain.SexMale, 20.0, false)
				err := d.ApplyPatch(tt.patch)
				assert.Error(t, err)
				var dverr *domain.DogValidationError
				assert.True(t, errors.As(err, &dverr), "expected DogValidationError, got %T", err)
				assert.Equal(t, tt.wantField, dverr.Field)
			})
		}
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
