package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newModifyActivity(id int) *domain.Activity {
	fixedDate := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	return mustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedDate)
}

func TestModifyActivityUseCase_Success(t *testing.T) {
	original := newModifyActivity(1)
	var saved *domain.Activity
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return original, nil
		},
		update: func(ctx context.Context, activity *domain.Activity) error {
			saved = activity
			return nil
		},
	}
	uc := NewModifyActivityUseCase(repo)

	newName := "Paseo Largo"
	newCapacity := 12
	patch := domain.ActivityPatch{
		Name:        &newName,
		MaxCapacity: &newCapacity,
	}
	output, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 1, Patch: patch})

	assert.NoError(t, err)
	assert.Equal(t, "Paseo Largo", output.Activity.Name())
	assert.Equal(t, 12, output.Activity.MaxCapacity())
	assert.NotNil(t, saved, "update should be called on non-empty patch")
	assert.Equal(t, "Paseo Largo", saved.Name())
}

func TestModifyActivityUseCase_EmptyPatchIsNoOp(t *testing.T) {
	original := newModifyActivity(1)
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return original, nil
		},
		update: func(context.Context, *domain.Activity) error {
			t.Fatal("update should not be called on empty patch")
			return nil
		},
	}
	uc := NewModifyActivityUseCase(repo)
	output, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 1, Patch: domain.ActivityPatch{}})
	assert.NoError(t, err)
	assert.Equal(t, "Paseo", output.Activity.Name())
}

func TestModifyActivityUseCase_NotFound(t *testing.T) {
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return nil, nil
		},
		update: func(context.Context, *domain.Activity) error {
			t.Fatal("update should not be called when activity is missing")
			return nil
		},
	}
	uc := NewModifyActivityUseCase(repo)
	_, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 99, Patch: domain.ActivityPatch{}})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestModifyActivityUseCase_InvalidID(t *testing.T) {
	uc := NewModifyActivityUseCase(&mockActivityRepository{})
	_, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 0, Patch: domain.ActivityPatch{}})
	assertValidationError(t, err, "id")
	_, err = uc.Execute(context.Background(), ModifyActivityInput{ID: -1, Patch: domain.ActivityPatch{}})
	assertValidationError(t, err, "id")
}

func TestModifyActivityUseCase_PatchValidationErrors(t *testing.T) {
	original := newModifyActivity(1)
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return original, nil
		},
		update: func(context.Context, *domain.Activity) error {
			t.Fatal("update should not be called on patch validation error")
			return nil
		},
	}
	uc := NewModifyActivityUseCase(repo)

	emptyName := ""
	emptyLocation := ""
	invalidType := domain.ActivityType("INVALID")
	zeroCapacity := 0
	zeroDuration := 0
	zeroDate := time.Time{}

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
			_, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 1, Patch: tt.patch})
			assertValidationError(t, err, tt.wantField)
		})
	}

	// After the failed patches, the original activity is still intact.
	assert.Equal(t, "Paseo", original.Name())
	assert.Equal(t, "Central", original.Location())
	assert.Equal(t, 5, original.MaxCapacity())
}

func TestModifyActivityUseCase_RepoError(t *testing.T) {
	repo := &mockActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return newModifyActivity(1), nil
		},
		update: func(ctx context.Context, activity *domain.Activity) error {
			return sentinelErr
		},
	}
	uc := NewModifyActivityUseCase(repo)
	newName := "Whatever"
	_, err := uc.Execute(context.Background(), ModifyActivityInput{ID: 1, Patch: domain.ActivityPatch{Name: &newName}})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sentinelErr))
	assert.Contains(t, err.Error(), "update activity 1")
}
