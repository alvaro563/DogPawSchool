package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewReservation(t *testing.T) {
	now := time.Now()

	t.Run("happy_path_forces_confirmed", func(t *testing.T) {
		r, err := domain.NewReservation(1, 10, 20, 30, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusConfirmed, r.Status())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			actID     int
			dogID     int
			passID    int
			now       time.Time
			wantInErr string
		}{
			{"negative_id", -1, 10, 20, 30, now, "id must not be negative"},
			{"zero_actID", 1, 0, 20, 30, now, "activityID must be greater than 0"},
			{"zero_dogID", 1, 10, 0, 30, now, "dogID must be greater than 0"},
			{"zero_passID", 1, 10, 20, 0, now, "passID must be greater than 0"},
			{"zero_time", 1, 10, 20, 30, time.Time{}, "createdAt must be a valid time"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewReservation(tt.id, tt.actID, tt.dogID, tt.passID, tt.now)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestNewReservationWithStatus(t *testing.T) {
	now := time.Now()

	t.Run("happy_path", func(t *testing.T) {
		r, err := domain.NewReservationWithStatus(1, 10, 20, 30, domain.StatusCancelledLate, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusCancelledLate, r.Status())
	})

	t.Run("invalid_status", func(t *testing.T) {
		_, err := domain.NewReservationWithStatus(1, 10, 20, 30, domain.ReservationStatus("BOGUS"), now)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}

func TestReservation_Cancel(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	activityDate := now.Add(3 * time.Hour)

	t.Run("cancel_in_time_when_more_than_2h_before", func(t *testing.T) {
		r, _ := domain.NewReservation(1, 10, 20, 30, now)
		err := r.Cancel(activityDate, now)
		assert.NoError(t, err)
		assert.True(t, r.WasCancelledInTime())
		assert.Equal(t, domain.StatusCancelledInTime, r.Status())
	})

	t.Run("cancel_late_when_less_than_2h_before", func(t *testing.T) {
		r, _ := domain.NewReservation(1, 10, 20, 30, now)
		later := activityDate.Add(-1 * time.Hour)
		err := r.Cancel(activityDate, later)
		assert.NoError(t, err)
		assert.True(t, r.WasCancelledLate())
		assert.Equal(t, domain.StatusCancelledLate, r.Status())
	})

	t.Run("cancel_exactly_at_2h_window_is_in_time", func(t *testing.T) {
		r, _ := domain.NewReservation(1, 10, 20, 30, now)
		exactly := activityDate.Add(-2 * time.Hour)
		err := r.Cancel(activityDate, exactly)
		assert.NoError(t, err)
		assert.True(t, r.WasCancelledInTime())
	})

	t.Run("cannot_cancel_when_already_cancelled", func(t *testing.T) {
		r, _ := domain.NewReservation(1, 10, 20, 30, now)
		_ = r.Cancel(activityDate, now)
		err := r.Cancel(activityDate, now)
		assert.Error(t, err)
	})

	t.Run("cannot_cancel_when_completed", func(t *testing.T) {
		r, _ := domain.NewReservation(1, 10, 20, 30, now)
		_ = r.Complete()
		err := r.Cancel(activityDate, now)
		assert.Error(t, err)
	})
}

func TestReservation_Complete(t *testing.T) {
	now := time.Now()
	r, _ := domain.NewReservation(1, 10, 20, 30, now)
	err := r.Complete()
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusCompleted, r.Status())

	err = r.Complete()
	assert.Error(t, err)
}

func TestReservation_MarkNoShow(t *testing.T) {
	now := time.Now()
	r, _ := domain.NewReservation(1, 10, 20, 30, now)
	err := r.MarkNoShow()
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusNoShow, r.Status())
}

func TestReservation_Forgive(t *testing.T) {
	now := time.Now()
	r, _ := domain.NewReservation(1, 10, 20, 30, now)
	_ = r.Cancel(now.Add(time.Hour), now.Add(3*time.Hour)) // sets to CancelledLate
	assert.Equal(t, domain.StatusCancelledLate, r.Status())
	err := r.Forgive()
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusForgiven, r.Status())

	r2, _ := domain.NewReservation(2, 10, 20, 30, now)
	err = r2.Forgive()
	assert.Error(t, err, "cannot forgive a reservation that is not CancelledLate")
}

func TestReservation_StatePredicates(t *testing.T) {
	now := time.Now()

	confirmed, _ := domain.NewReservation(1, 10, 20, 30, now)
	assert.True(t, confirmed.IsConfirmed())
	assert.False(t, confirmed.IsCancelled())

	cancelledInTime, _ := domain.NewReservationWithStatus(2, 10, 20, 30, domain.StatusCancelledInTime, now)
	assert.False(t, cancelledInTime.IsConfirmed())
	assert.True(t, cancelledInTime.IsCancelled())
	assert.True(t, cancelledInTime.WasCancelledInTime())
	assert.False(t, cancelledInTime.WasCancelledLate())

	cancelledLate, _ := domain.NewReservationWithStatus(3, 10, 20, 30, domain.StatusCancelledLate, now)
	assert.True(t, cancelledLate.IsCancelled())
	assert.False(t, cancelledLate.WasCancelledInTime())
	assert.True(t, cancelledLate.WasCancelledLate())

	forgiven, _ := domain.NewReservationWithStatus(4, 10, 20, 30, domain.StatusForgiven, now)
	assert.True(t, forgiven.IsCancelled())
}

func TestReservationStatus_IsValid(t *testing.T) {
	assert.True(t, domain.StatusConfirmed.IsValid())
	assert.True(t, domain.StatusCompleted.IsValid())
	assert.True(t, domain.StatusCancelledInTime.IsValid())
	assert.True(t, domain.StatusCancelledLate.IsValid())
	assert.True(t, domain.StatusForgiven.IsValid())
	assert.True(t, domain.StatusNoShow.IsValid())
	assert.False(t, domain.ReservationStatus("").IsValid())
	assert.False(t, domain.ReservationStatus("BOGUS").IsValid())
}
