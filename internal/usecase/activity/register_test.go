package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

// mockDogRepository satisfies domain.DogRepository with no-ops so
// RegisterActivityUseCase can call GetByID to verify the target dog
// exists. The tests in this file don't exercise dog validation
// (input.DogID is always nil), but the use case still calls the
// repo when the input passes nil.
//
// All other methods of domain.DogRepository are stubbed to satisfy
// the interface; they should never be called in these tests.
type mockDogRepository struct{}

func (m *mockDogRepository) GetByID(_ context.Context, _ int) (*domain.Dog, error) {
	// Not used by these tests because dogID is always nil in the input.
	// Return a sentinel error so a test that accidentally passes a
	// non-nil dogID fails loudly instead of silently passing.
	return nil, errors.New("mockDogRepository.GetByID unexpectedly called")
}

func (m *mockDogRepository) GetByIDForUpdate(_ context.Context, _ int) (*domain.Dog, error) {
	return nil, errors.New("mockDogRepository.GetByIDForUpdate unexpectedly called")
}

func (m *mockDogRepository) Create(_ context.Context, _ *domain.Dog) (int, error) {
	return 0, errors.New("mockDogRepository.Create unexpectedly called")
}
func (m *mockDogRepository) Update(_ context.Context, _ *domain.Dog) error {
	return errors.New("mockDogRepository.Update unexpectedly called")
}
func (m *mockDogRepository) Delete(_ context.Context, _ int) error {
	return errors.New("mockDogRepository.Delete unexpectedly called")
}
func (m *mockDogRepository) GetByIDs(_ context.Context, _ []int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByOwner(_ context.Context, _ int, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) List(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByIncompatibility(_ context.Context, _ int, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByBreed(_ context.Context, _ string, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListBySex(_ context.Context, _ domain.Sex, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByNeutered(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByHeat(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByIsActive(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListByAgeBracket(_ context.Context, _ domain.AgeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListBySizeBracket(_ context.Context, _ domain.SizeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *mockDogRepository) ListAll(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}

func validRegisterInput() RegisterActivityInput {
	return MustNewRegisterActivityInput(
		"Paseo Río", "", "Parking Central",
		domain.TypeRoute, 8, 2,
		time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
		nil, nil,
	)
}

func TestRegisterActivityUseCase_Success(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			assert.Equal(t, "Paseo Río", activity.Name())
			assert.Equal(t, domain.TypeRoute, activity.Type())
			assert.Equal(t, 8, activity.MaxCapacity())
			return 42, nil
		},
	}
	uc := NewRegisterActivityUseCase(repo, &mockDogRepository{})

	output, err := uc.Execute(context.Background(), validRegisterInput())

	assert.NoError(t, err)
	assert.Equal(t, 42, output.ID)
}

func TestNewRegisterActivityInput(t *testing.T) {
	t.Parallel()
	fixedDate := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	base := func() RegisterActivityInput {
		return MustNewRegisterActivityInput("n", "", "l", domain.TypeRoute, 8, 2, fixedDate, nil, nil)
	}

	scenarios := []struct {
		name      string
		factory   func() (RegisterActivityInput, error)
		wantField string
	}{
		{"empty_name", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("", "", "l", domain.TypeRoute, 8, 2, fixedDate, nil, nil)
		}, "name"},
		{"empty_location", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "", domain.TypeRoute, 8, 2, fixedDate, nil, nil)
		}, "location"},
		{"invalid_type", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "l", domain.ActivityType("INVALID"), 8, 2, fixedDate, nil, nil)
		}, "activity_type"},
		{"zero_capacity", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "l", domain.TypeRoute, 0, 2, fixedDate, nil, nil)
		}, "max_capacity"},
		{"negative_capacity", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "l", domain.TypeRoute, -1, 2, fixedDate, nil, nil)
		}, "max_capacity"},
		{"zero_duration", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "l", domain.TypeRoute, 8, 0, fixedDate, nil, nil)
		}, "duration_in_hours"},
		{"zero_date", func() (RegisterActivityInput, error) {
			return NewRegisterActivityInput("n", "", "l", domain.TypeRoute, 8, 2, time.Time{}, nil, nil)
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
	t.Parallel()
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			return 0, sentinelErr
		},
	}
	uc := NewRegisterActivityUseCase(repo, &mockDogRepository{})
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "register activity")
}
