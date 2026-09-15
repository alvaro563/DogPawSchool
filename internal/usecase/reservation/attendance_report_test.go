package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func newAttendanceEntry(id int, dogName string, activityName string, when time.Time) *domain.AttendanceReportEntry {
	return &domain.AttendanceReportEntry{
		ReservationID: id,
		ActivityID:    100 + id,
		ActivityName:  activityName,
		ActivityDate:  when,
		DogID:         200 + id,
		DogName:       dogName,
		DogPassport:   "ES-TEST-" + dogName,
	}
}

func TestNewListAttendanceReportInput(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	dayAfter := day.Add(24 * time.Hour)

	t.Run("default_limit_applied", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAttendanceReportInput(nil, nil, 0, 0)
		require.NoError(t, err)
		assert.Equal(t, AttendanceReportDefaultLimit, in.Limit())
		assert.Equal(t, 0, in.Offset())
	})

	t.Run("ui_cap_applied", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(nil, nil, AttendanceReportMaxLimitUI+1, 0)
		require.Error(t, err)
		assertValidationError(t, err, "limit")
	})

	t.Run("negative_limit_field_is_limit", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(nil, nil, -1, 0)
		require.Error(t, err)
		assertValidationError(t, err, "limit")
	})

	t.Run("negative_limit_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(nil, nil, -1, 0)
		require.Error(t, err)
	})

	t.Run("negative_offset_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(nil, nil, 100, -1)
		require.Error(t, err)
	})

	t.Run("from_after_to_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(&dayAfter, &day, 100, 0)
		require.Error(t, err)
		assertValidationError(t, err, "to")
	})

	t.Run("from_eq_to_accepted", func(t *testing.T) {
		t.Parallel()
		_, err := NewListAttendanceReportInput(&day, &day, 100, 0)
		require.NoError(t, err)
	})

	t.Run("nil_bounds_pass_through", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAttendanceReportInput(nil, nil, 250, 0)
		require.NoError(t, err)
		assert.Equal(t, 250, in.Limit())
		assert.Nil(t, in.From())
		assert.Nil(t, in.To())
	})

	t.Run("custom_pagination_preserved", func(t *testing.T) {
		t.Parallel()
		in, err := NewListAttendanceReportInput(&day, &dayAfter, 250, 50)
		require.NoError(t, err)
		assert.Equal(t, 250, in.Limit())
		assert.Equal(t, 50, in.Offset())
		assert.Equal(t, &day, in.From())
		assert.Equal(t, &dayAfter, in.To())
	})
}

func TestNewExportAttendanceReportInput(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	dayAfter := day.Add(24 * time.Hour)

	t.Run("default_export_cap", func(t *testing.T) {
		t.Parallel()
		in, err := NewExportAttendanceReportInput(&day, &dayAfter)
		require.NoError(t, err)
		assert.Equal(t, AttendanceReportMaxLimitExport, in.Limit())
		assert.Equal(t, 0, in.Offset())
	})

	t.Run("nil_bounds_accepted", func(t *testing.T) {
		t.Parallel()
		in, err := NewExportAttendanceReportInput(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, AttendanceReportMaxLimitExport, in.Limit())
	})

	t.Run("single_day_accepted", func(t *testing.T) {
		t.Parallel()
		in, err := NewExportAttendanceReportInput(&day, &day)
		require.NoError(t, err)
		assert.Equal(t, &day, in.From())
		assert.Equal(t, &day, in.To())
	})

	t.Run("from_after_to_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := NewExportAttendanceReportInput(&dayAfter, &day)
		require.Error(t, err)
		assertValidationError(t, err, "to")
	})
}

func TestListAttendanceReportUseCase_Execute(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	e1 := newAttendanceEntry(1, "Luna", "Paseo Social Matutino", day)
	e2 := newAttendanceEntry(2, "Toby", "Ruta Río", day.Add(time.Hour))

	t.Run("returns_entries", func(t *testing.T) {
		t.Parallel()
		expected := []*domain.AttendanceReportEntry{e1, e2}
		repo := &mockReservationRepository{
			listAttendanceReport: func(_ context.Context, from, to *time.Time, limit, offset int) ([]*domain.AttendanceReportEntry, error) {
				assert.Nil(t, from)
				assert.Nil(t, to)
				assert.Equal(t, AttendanceReportDefaultLimit, limit)
				assert.Equal(t, 0, offset)
				return expected, nil
			},
		}
		uc := NewListAttendanceReportUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListAttendanceReportInput(nil, nil, 0, 0))
		require.NoError(t, err)
		assert.Equal(t, expected, out.Entries)
		assert.Equal(t, AttendanceReportDefaultLimit, out.Limit)
		assert.Equal(t, 0, out.Offset)
	})

	t.Run("propagates_from_to", func(t *testing.T) {
		t.Parallel()
		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
		repo := &mockReservationRepository{
			listAttendanceReport: func(_ context.Context, f, toArg *time.Time, _, _ int) ([]*domain.AttendanceReportEntry, error) {
				require.NotNil(t, f)
				require.NotNil(t, toArg)
				assert.Equal(t, from, *f)
				assert.Equal(t, to, *toArg)
				return []*domain.AttendanceReportEntry{e1}, nil
			},
		}
		uc := NewListAttendanceReportUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListAttendanceReportInput(&from, &to, 100, 0))
		require.NoError(t, err)
		assert.Equal(t, 1, len(out.Entries))
	})

	t.Run("empty_list", func(t *testing.T) {
		t.Parallel()
		repo := &mockReservationRepository{
			listAttendanceReport: func(_ context.Context, _, _ *time.Time, _, _ int) ([]*domain.AttendanceReportEntry, error) {
				return nil, nil
			},
		}
		uc := NewListAttendanceReportUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListAttendanceReportInput(nil, nil, 0, 0))
		require.NoError(t, err)
		assert.Empty(t, out.Entries)
	})

	t.Run("repo_error_wrapped", func(t *testing.T) {
		t.Parallel()
		repoErr := errors.New("db failure")
		repo := &mockReservationRepository{
			listAttendanceReport: func(_ context.Context, _, _ *time.Time, _, _ int) ([]*domain.AttendanceReportEntry, error) {
				return nil, repoErr
			},
		}
		uc := NewListAttendanceReportUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewListAttendanceReportInput(nil, nil, 0, 0))
		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})
}
