package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestPassWithPaid(id int, isPaid bool) *domain.Pass {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	return domain.MustNewPass(id, 10, 10, 100, domain.PassGeneric, 1, now, now, nil, isPaid)
}

func TestSetPassPaidUseCase_Success_MarkPaid(t *testing.T) {
	t.Parallel()
	original := newTestPassWithPaid(1, false)
	var saved *domain.Pass
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(ctx context.Context, pass *domain.Pass, _ time.Time) error {
			saved = pass
			return nil
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	in := MustNewSetPassPaidInput(1, true)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.NotNil(t, output.Pass)
	assert.True(t, output.Pass.IsPaid())
	assert.NotNil(t, saved)
	assert.True(t, saved.IsPaid())
}

func TestSetPassPaidUseCase_Success_MarkUnpaid(t *testing.T) {
	t.Parallel()
	original := newTestPassWithPaid(1, true)
	var saved *domain.Pass
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(ctx context.Context, pass *domain.Pass, _ time.Time) error {
			saved = pass
			return nil
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	in := MustNewSetPassPaidInput(1, false)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.False(t, output.Pass.IsPaid())
	assert.False(t, saved.IsPaid())
}

func TestSetPassPaidUseCase_NotFound(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, nil
		},
		update: func(_ context.Context, _ *domain.Pass, _ time.Time) error {
			t.Fatal("update should not be called when pass is missing")
			return nil
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewSetPassPaidInput(99, true))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSetPassPaidUseCase_RepoError_OnGet(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewSetPassPaidInput(1, true))
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "get pass 1")
}

func TestSetPassPaidUseCase_RepoError_OnUpdate(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return newTestPassWithPaid(1, false), nil
		},
		update: func(ctx context.Context, pass *domain.Pass, _ time.Time) error {
			return sentinelErr
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewSetSetPassPaidInputForTest(1, true))
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "update pass 1")
}

func TestSetPassPaidUseCase_RepoError_OnGetUpdated(t *testing.T) {
	t.Parallel()
	callCount := 0
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			callCount++
			if callCount == 1 {
				return newTestPassWithPaid(1, false), nil
			}
			return nil, sentinelErr
		},
		update: func(ctx context.Context, pass *domain.Pass, _ time.Time) error {
			return nil
		},
	}
	uc := NewSetPassPaidUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewSetPassPaidInput(1, true))
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "get updated pass 1")
}

func TestNewSetPassPaidInput_InvalidID(t *testing.T) {
	t.Parallel()
	for _, id := range []int{0, -1} {
		_, err := NewSetPassPaidInput(id, true)
		assertValidationError(t, err, "id")
	}
}

// MustNewSetSetPassPaidInputForTest is a tiny alias to keep the
// failing-update test readable. It is the same as
// MustNewSetPassPaidInput but gives the failure path its own
// variable so it is clear which scenario we are exercising.
func MustNewSetSetPassPaidInputForTest(id int, isPaid bool) SetPassPaidInput {
	return MustNewSetPassPaidInput(id, isPaid)
}
