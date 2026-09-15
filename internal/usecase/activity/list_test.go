package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestListAllActivitiesUseCase_Success(t *testing.T) {
	t.Parallel()
	date1 := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	expected := []*domain.Activity{
		mustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, date1),
		mustNewActivity(2, "b", "l", domain.TypeRoute, 5, 1, date2),
		mustNewActivity(3, "c", "l", domain.TypeRoute, 5, 1, date3),
	}
	repo := &mockActivityRepository{
		list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in := MustNewListAllActivitiesInput(0, 0, nil, nil, nil, 0, true)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Activities)
}

func TestListAllActivitiesUseCase_Empty(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			return nil, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in := MustNewListAllActivitiesInput(0, 0, nil, nil, nil, 0, true)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.Activities))
}

func TestListAllActivitiesUseCase_PaginationNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		wantRepoLimit  int
		wantRepoOffset int
	}{
		{"zero_limit_becomes_default", 0, 0, 50, 0},
		{"negative_limit_becomes_default", -1, 0, 50, 0},
		{"over_max_is_capped", 500, 0, 100, 0},
		{"negative_offset_is_zero", 50, -5, 50, 0},
		{"exact_max_kept", 100, 0, 100, 0},
		{"custom_values_preserved", 25, 10, 25, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockActivityRepository{
				list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
					assert.Equal(t, tt.wantRepoLimit, limit)
					assert.Equal(t, tt.wantRepoOffset, offset)
					return nil, nil
				},
			}
			uc := NewListAllActivitiesUseCase(repo)
			in := MustNewListAllActivitiesInputAdmin(tt.inputLimit, tt.inputOffset)
			_, err := uc.Execute(context.Background(), in)
			assert.NoError(t, err)
		})
	}
}

func TestListAllActivitiesUseCase_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			return nil, sentinelErr
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in := MustNewListAllActivitiesInput(0, 0, nil, nil, nil, 0, true)
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "list all activities")
}

// ============================================================================
// New factory: filtered list (closedFilter, from/to pointers)
// ============================================================================

func mustParseT(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestNewListAllActivitiesInput_Filters(t *testing.T) {
	t.Parallel()
	from := mustParseT("2026-02-01T00:00:00Z")
	to := mustParseT("2026-02-28T23:59:59Z")
	closedTrue := true
	closedFalse := false

	t.Run("no_filters_keeps_limit_default", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAllActivitiesInput(0, 0, nil, nil, nil, 0, true)
		require_NoErr(t, err)
		assert.Equal(t, 50, in.Limit())
		assert.Equal(t, 0, in.Offset())
		assert.Nil(t, in.From())
		assert.Nil(t, in.To())
		assert.Nil(t, in.ClosedFilter())
	})

	t.Run("closed_true_with_dates", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAllActivitiesInput(100, 5, from, to, &closedTrue, 0, true)
		require_NoErr(t, err)
		assert.Equal(t, 100, in.Limit())
		assert.Equal(t, 5, in.Offset())
		require_PtrEq(t, in.From(), from)
		require_PtrEq(t, in.To(), to)
		require_Closed(t, in.ClosedFilter(), true)
	})

	t.Run("closed_false_open_only", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAllActivitiesInput(50, 0, nil, nil, &closedFalse, 0, true)
		require_NoErr(t, err)
		require_Closed(t, in.ClosedFilter(), false)
	})

	t.Run("from_after_to_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAllActivitiesInput(50, 0, to, from, nil, 0, true)
		require_Error(t, err)
		assertValidationError(t, err, "from")
	})
}

// Helper assertions — names start with require_ to opt out of the
// testify import collision with `require`.
func require_NoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func require_Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
func require_PtrEq(t *testing.T, got, want *time.Time) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("pointer mismatch: got=%v want=%v", got, want)
		}
		return
	}
	if !got.Equal(*want) {
		t.Fatalf("time pointer mismatch: got=%v want=%v", *got, *want)
	}
}
func require_Closed(t *testing.T, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected ClosedFilter=%v, got nil", want)
	}
	if *got != want {
		t.Fatalf("expected ClosedFilter=%v, got %v", want, *got)
	}
}

// ============================================================================
// Use case dispatch: closedFilter takes precedence over from/to
// ============================================================================

func TestListAllActivitiesUseCase_Dispatch_ClosedFilter(t *testing.T) {
	t.Parallel()
	closedTrue := true
	expected := []*domain.Activity{
		mustNewActivity(1, "x", "loc", domain.TypeRoute, 5, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	repo := &mockActivityRepository{
		listByClosed: func(_ context.Context, _ int, _ bool, closed bool, from, to *time.Time, limit, offset int) ([]*domain.Activity, error) {
			assert.True(t, closed)
			assert.Nil(t, from)
			assert.Nil(t, to)
			assert.Equal(t, 50, limit)
			return expected, nil
		},
		list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			t.Fatal("List must NOT be called when closedFilter is set")
			return nil, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in, err := NewListAllActivitiesInput(50, 0, nil, nil, &closedTrue, 0, true)
	require_NoErr(t, err)
	out, err := uc.Execute(context.Background(), in)
	require_NoErr(t, err)
	assert.Equal(t, expected, out.Activities)
}

func TestListAllActivitiesUseCase_Dispatch_DateOnly(t *testing.T) {
	t.Parallel()
	expected := []*domain.Activity{mustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, time.Now()) }
	// from/to set + closedFilter=nil → ListByDateRange wins.
	from := mustParseT("2026-01-01T00:00:00Z")
	to := mustParseT("2026-12-31T23:59:59Z")
	repo := &mockActivityRepository{
		listByDateRange: func(_ context.Context, _ int, _ bool, gotFrom, gotTo time.Time, limit, offset int) ([]*domain.Activity, error) {
			assert.True(t, gotFrom.Equal(*from), "from must be propagated")
			assert.True(t, gotTo.Equal(*to), "to must be propagated")
			assert.Equal(t, 50, limit)
			return expected, nil
		},
		list: func(ctx context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			t.Fatal("List must NOT be called when from/to are set")
			return nil, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in, err := NewListAllActivitiesInput(50, 0, from, to, nil, 0, true)
	require_NoErr(t, err)
	out, err := uc.Execute(context.Background(), in)
	require_NoErr(t, err)
	assert.Equal(t, expected, out.Activities)
}

func TestListAllActivitiesUseCase_Dispatch_NoFilter(t *testing.T) {
	t.Parallel()
	expected := []*domain.Activity{mustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, time.Now()) }
	repo := &mockActivityRepository{
		list: func(_ context.Context, _ int, _ bool, limit, offset int) ([]*domain.Activity, error) {
			assert.Equal(t, 50, limit)
			return expected, nil
		},
		listByDateRange: func(_ context.Context, _ int, _ bool, _, _ time.Time, _, _ int) ([]*domain.Activity, error) {
			t.Fatal("ListByDateRange must NOT be called without date filter")
			return nil, nil
		},
		listByClosed: func(_ context.Context, _ int, _ bool, _ bool, _, _ *time.Time, _, _ int) ([]*domain.Activity, error) {
			t.Fatal("ListByClosed must NOT be called without closedFilter")
			return nil, nil
		},
	}
	uc := NewListAllActivitiesUseCase(repo)
	in, err := NewListAllActivitiesInput(50, 0, nil, nil, nil, 0, true)
	require_NoErr(t, err)
	out, err := uc.Execute(context.Background(), in)
	require_NoErr(t, err)
	assert.Equal(t, expected, out.Activities)
}
