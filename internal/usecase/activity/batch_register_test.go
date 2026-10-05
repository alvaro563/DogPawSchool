package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func weeklyDates(start time.Time, n int) []time.Time {
	dates := make([]time.Time, n)
	for i := range dates {
		dates[i] = start.AddDate(0, 0, 7*i)
	}
	return dates
}

func TestNewBatchRegisterActivityInput(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC)
	register := validRegisterInput()

	scenarios := []struct {
		name      string
		dates     []time.Time
		wantField string
	}{
		{"empty", nil, "dates"},
		{"too_many", weeklyDates(base, 53), "dates"},
		{"zero_date", []time.Time{base, {}, base.AddDate(0, 0, 14)}, "dates[1]"},
		{"duplicate", []time.Time{base, base, base.AddDate(0, 0, 14)}, "dates"},
		{"out_of_order", []time.Time{base, base.AddDate(0, 0, 14), base.AddDate(0, 0, 7)}, "dates"},
	}
	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewBatchRegisterActivityInput(register, tt.dates)
			assertValidationError(t, err, tt.wantField)
		})
	}

	t.Run("valid_single", func(t *testing.T) {
		t.Parallel()
		in, err := NewBatchRegisterActivityInput(register, []time.Time{base})
		require.NoError(t, err)
		assert.Len(t, in.Dates(), 1)
	})
	t.Run("valid_max_52", func(t *testing.T) {
		t.Parallel()
		in, err := NewBatchRegisterActivityInput(register, weeklyDates(base, 52))
		require.NoError(t, err)
		assert.Len(t, in.Dates(), 52)
	})
}

func TestBatchRegisterActivityUseCase_Success(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC)
	dates := weeklyDates(start, 4)

	var gotDates []time.Time
	nextID := 41
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			gotDates = append(gotDates, activity.Date())
			id := nextID
			nextID++
			return id, nil
		},
	}
	uc := NewBatchRegisterActivityUseCase(&stubTransactorActivity{}, repo, &mockDogRepository{})

	output, err := uc.Execute(context.Background(),
		MustNewBatchRegisterActivityInput(validRegisterInput(), dates))

	require.NoError(t, err)
	assert.Equal(t, []int{41, 42, 43, 44}, output.IDs)
	require.Len(t, gotDates, 4)
	for i, got := range gotDates {
		assert.WithinDuration(t, dates[i], got, time.Microsecond)
	}
}

func TestBatchRegisterActivityUseCase_RollsBackOnError(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC)

	calls := 0
	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			calls++
			if calls == 2 {
				return 0, errors.New("insert failed")
			}
			return 41, nil
		},
	}
	uc := NewBatchRegisterActivityUseCase(&stubTransactorActivity{}, repo, &mockDogRepository{})

	output, err := uc.Execute(context.Background(),
		MustNewBatchRegisterActivityInput(validRegisterInput(), weeklyDates(start, 4)))

	require.Error(t, err)
	assert.Empty(t, output.IDs)
	assert.Equal(t, 2, calls, "stops at the first failure")
}

// missingDogRepo returns domain.ErrNotFound for every lookup, so the
// use case must abort before touching the activity repository.
type missingDogRepo struct{}

func (m *missingDogRepo) GetByID(_ context.Context, _ int) (*domain.Dog, error) {
	return nil, domain.ErrNotFound
}
func (m *missingDogRepo) GetByIDForUpdate(_ context.Context, _ int) (*domain.Dog, error) {
	return nil, errors.New("not expected")
}
func (m *missingDogRepo) Create(_ context.Context, _ *domain.Dog) (int, error) {
	return 0, errors.New("not expected")
}
func (m *missingDogRepo) Update(_ context.Context, _ *domain.Dog) error {
	return errors.New("not expected")
}
func (m *missingDogRepo) Delete(_ context.Context, _ int) error { return errors.New("not expected") }
func (m *missingDogRepo) GetByIDs(_ context.Context, _ []int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByOwner(_ context.Context, _ int, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) List(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByIncompatibility(_ context.Context, _ int, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByBreed(_ context.Context, _ string, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListBySex(_ context.Context, _ domain.Sex, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByNeutered(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByHeat(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByIsActive(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListByAgeBracket(_ context.Context, _ domain.AgeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListBySizeBracket(_ context.Context, _ domain.SizeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (m *missingDogRepo) ListAll(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}

func TestBatchRegisterActivityUseCase_InvalidDogAbortsBeforeCreate(t *testing.T) {
	t.Parallel()
	dogID := 99
	input := MustNewRegisterActivityInput(
		"Sesión individual", "", "Central", domain.TypeIndividual, 1, 1,
		time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC), &dogID, nil,
	)
	batch := MustNewBatchRegisterActivityInput(input, weeklyDates(time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC), 4))

	repo := &mockActivityRepository{
		create: func(ctx context.Context, activity *domain.Activity) (int, error) {
			t.Fatal("repo.Create must not be called when dog validation fails")
			return 0, nil
		},
	}
	uc := NewBatchRegisterActivityUseCase(&stubTransactorActivity{}, repo, &missingDogRepo{})

	_, err := uc.Execute(context.Background(), batch)

	assert.ErrorIs(t, err, ErrInvalidDog)
}
