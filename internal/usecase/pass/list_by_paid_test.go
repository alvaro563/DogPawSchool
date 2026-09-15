package pass

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListByPaidPassesUseCase_Success(t *testing.T) {
	t.Parallel()
	paid := newTestPassWithPaid(1, true)
	unpaid := newTestPassWithPaid(2, false)
	repo := &mockPassRepository{
		listByPaid: func(ctx context.Context, isPaid bool, limit, offset int) ([]*domain.Pass, error) {
			assert.True(t, isPaid)
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return []*domain.Pass{paid}, nil
		},
	}
	uc := NewListByPaidPassesUseCase(repo)
	output, err := uc.Execute(context.Background(), MustNewListByPaidPassesInput(true, 0, 0))
	assert.NoError(t, err)
	assert.Len(t, output.Passes, 1)
	assert.True(t, output.Passes[0].IsPaid())

	_ = unpaid // silence unused warning when reordering
}

func TestListByPaidPassesUseCase_Empty(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		listByPaid: func(ctx context.Context, isPaid bool, limit, offset int) ([]*domain.Pass, error) {
			assert.False(t, isPaid)
			return []*domain.Pass{}, nil
		},
	}
	uc := NewListByPaidPassesUseCase(repo)
	output, err := uc.Execute(context.Background(), MustNewListByPaidPassesInput(false, 50, 0))
	assert.NoError(t, err)
	assert.Empty(t, output.Passes)
}

func TestListByPaidPassesUseCase_PaginationNormalized(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		listByPaid: func(ctx context.Context, isPaid bool, limit, offset int) ([]*domain.Pass, error) {
			assert.Equal(t, 100, limit, "limit above max should be capped to maxPageLimit")
			assert.Equal(t, 0, offset, "negative offset should be normalized to 0")
			return []*domain.Pass{}, nil
		},
	}
	uc := NewListByPaidPassesUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewListByPaidPassesInput(true, 500, -10))
	assert.NoError(t, err)
}

func TestListByPaidPassesUseCase_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		listByPaid: func(ctx context.Context, isPaid bool, limit, offset int) ([]*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListByPaidPassesUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewListByPaidPassesInput(true, 50, 0))
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list passes by paid=true")
}

func TestNewListByPaidPassesInput_Defaults(t *testing.T) {
	t.Parallel()
	in, err := NewListByPaidPassesInput(true, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 50, in.Limit())
	assert.Equal(t, 0, in.Offset())
	assert.True(t, in.IsPaid())
}

func TestListByPaidPassesUseCase_BothValues(t *testing.T) {
	t.Parallel()
	// Confirms the is_paid flag is forwarded as-is: calling with
	// false returns the unpaid cohort, with true returns the paid one.
	captured := []bool{}
	repo := &mockPassRepository{
		listByPaid: func(ctx context.Context, isPaid bool, limit, offset int) ([]*domain.Pass, error) {
			captured = append(captured, isPaid)
			return []*domain.Pass{}, nil
		},
	}
	uc := NewListByPaidPassesUseCase(repo)
	_, err := uc.Execute(context.Background(), MustNewListByPaidPassesInput(false, 50, 0))
	assert.NoError(t, err)
	_, err = uc.Execute(context.Background(), MustNewListByPaidPassesInput(true, 50, 0))
	assert.NoError(t, err)
	assert.Equal(t, []bool{false, true}, captured)
}

// silence unused import linter when errors is only referenced via
// assert.ErrorIs above in some test files
var _ = errors.New
