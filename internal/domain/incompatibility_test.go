package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewIncompatibility(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		scenarios := []struct {
			name           string
			id             int
			nombre         string
			tipo           domain.IncompatibilityLevel
			expectedID     int
			expectedNombre string
			expectedTipo   domain.IncompatibilityLevel
		}{
			{"absoluta", 1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, 1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta},
			{"media", 2, "No tolera cachorros", domain.IncompatibilityLevelMedia, 2, "No tolera cachorros", domain.IncompatibilityLevelMedia},
			{"baja", 3, "Le asustan los petardos", domain.IncompatibilityLevelBaja, 3, "Le asustan los petardos", domain.IncompatibilityLevelBaja},
			{"zero_id_is_valid_for_draft", 0, "Draft", domain.IncompatibilityLevelMedia, 0, "Draft", domain.IncompatibilityLevelMedia},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				i, err := domain.NewIncompatibility(s.id, s.nombre, s.tipo)
				assert.NoError(t, err)
				assert.NotNil(t, i)
				assert.Equal(t, s.expectedID, i.ID())
				assert.Equal(t, s.expectedNombre, i.Name())
				assert.Equal(t, s.expectedTipo, i.Type())
			})
		}
	})

	t.Run("validation_errors", func(t *testing.T) {
		scenarios := []struct {
			name          string
			id            int
			nombre        string
			tipo          domain.IncompatibilityLevel
			expectedError string
		}{
			{"negative_id", -1, "Valid name", domain.IncompatibilityLevelMedia, "id must not be negative"},
			{"empty_name", 1, "", domain.IncompatibilityLevelMedia, "name must not be empty"},
			{"invalid_level", 1, "Valid name", domain.IncompatibilityLevel("INVALID"), "invalid level"},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				i, err := domain.NewIncompatibility(s.id, s.nombre, s.tipo)
				assert.Error(t, err)
				assert.Nil(t, i)
				assert.Contains(t, err.Error(), s.expectedError)
			})
		}
	})
}

func TestIncompatibilityLevel_IsValid(t *testing.T) {
	scenarios := []struct {
		name     string
		level    domain.IncompatibilityLevel
		expected bool
	}{
		{"absoluta_is_valid", domain.IncompatibilityLevelAbsoluta, true},
		{"media_is_valid", domain.IncompatibilityLevelMedia, true},
		{"baja_is_valid", domain.IncompatibilityLevelBaja, true},
		{"invalid_level_is_not_valid", domain.IncompatibilityLevel("INVALID"), false},
		{"empty_level_is_not_valid", domain.IncompatibilityLevel(""), false},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, s.level.IsValid())
		})
	}
}

func TestIncompatibility_Getters(t *testing.T) {
	t.Run("getters_return_constructor_values", func(t *testing.T) {
		i, err := domain.NewIncompatibility(42, "Miedo a tormentas", domain.IncompatibilityLevelBaja)
		assert.NoError(t, err)
		assert.NotNil(t, i)
		assert.Equal(t, 42, i.ID())
		assert.Equal(t, "Miedo a tormentas", i.Name())
		assert.Equal(t, domain.IncompatibilityLevelBaja, i.Type())
	})
}
