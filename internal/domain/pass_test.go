package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestPass(t *testing.T, opts ...func(*domain.Pass)) *domain.Pass {
	t.Helper()
	now := time.Now()
	p, err := domain.NewPass(1, 10, 100, domain.PassGeneric, 1, now, nil)
	if err != nil {
		t.Fatalf("newTestPass: %v", err)
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func TestNewPass(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		now := time.Now()
		exp := now.Add(30 * 24 * time.Hour)
		p, err := domain.NewPass(1, 10, 100, domain.PassGeneric, 1, now, &exp)
		assert.NoError(t, err)
		assert.Equal(t, 10, p.RemainingSessions())
		assert.Equal(t, &exp, p.ExpiresAt())
	})

	t.Run("validation_errors", func(t *testing.T) {
		now := time.Now()
		tests := []struct {
			name      string
			id        int
			n         int
			price     int
			pt        domain.PassType
			userID    int
			now       time.Time
			expires   *time.Time
			wantInErr string
		}{
			{"negative_id", -1, 10, 100, domain.PassGeneric, 1, now, nil, "id must not be negative"},
			{"zero_sessions", 1, 0, 100, domain.PassGeneric, 1, now, nil, "numOfSessions must be greater than 0"},
			{"negative_price", 1, 10, -1, domain.PassGeneric, 1, now, nil, "price must not be negative"},
			{"invalid_type", 1, 10, 100, domain.PassType("INVALID"), 1, now, nil, "invalid passType"},
			{"zero_user_id", 1, 10, 100, domain.PassGeneric, 0, now, nil, "userID must be greater than 0"},
			{"zero_time", 1, 10, 100, domain.PassGeneric, 1, time.Time{}, nil, "createdAt must be a valid time"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewPass(tt.id, tt.n, tt.price, tt.pt, tt.userID, tt.now, tt.expires)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestPass_IsExpired(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	creationTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		expiresAt *time.Time
		now       time.Time
		expected  bool
	}{
		{"nil_expiration_means_open_pass", nil, now, false},
		{"future_expiration_not_expired", ptrTime(now.Add(time.Hour)), now, false},
		{"past_expiration_is_expired", ptrTime(now.Add(-time.Hour)), now, true},
		{"expiration_exactly_at_now_is_not_expired", ptrTime(now), now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := domain.NewPass(1, 10, 100, domain.PassGeneric, 1, creationTime, tt.expiresAt)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, p.IsExpired(tt.now))
		})
	}
}

func TestPass_IsExhausted(t *testing.T) {
	tests := []struct {
		name      string
		remaining int
		expected  bool
	}{
		{"zero_remaining_is_exhausted", 0, true},
		{"negative_remaining_is_exhausted", -1, true},
		{"one_remaining_is_not_exhausted", 1, false},
		{"many_remaining_is_not_exhausted", 50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPass(t)
			for i := 0; i < 10-tt.remaining; i++ {
				_, _ = p.ConsumeSession("consume", time.Now())
			}
			assert.Equal(t, tt.expected, p.IsExhausted())
		})
	}
}

func TestPass_CanConsume(t *testing.T) {
	p := newTestPass(t)
	assert.True(t, p.CanConsume(time.Now()))
	for i := 0; i < 10; i++ {
		_, _ = p.ConsumeSession("consume", time.Now())
	}
	assert.False(t, p.CanConsume(time.Now()))
}

func TestPass_ConsumeSession(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		p := newTestPass(t)
		now := time.Now()
		mov, err := p.ConsumeSession("Booking Route", now)
		assert.NoError(t, err)
		assert.Equal(t, -1, mov.Amount())
		assert.Equal(t, "Booking Route", mov.Reason())
		assert.Equal(t, 9, p.RemainingSessions())
		assert.Len(t, p.Movements(), 1)
	})

	t.Run("empty_reason_returns_error", func(t *testing.T) {
		p := newTestPass(t)
		_, err := p.ConsumeSession("", time.Now())
		assert.Error(t, err)
	})

	t.Run("exhausted_returns_error", func(t *testing.T) {
		p := newTestPass(t)
		for i := 0; i < 10; i++ {
			_, _ = p.ConsumeSession("consume", time.Now())
		}
		_, err := p.ConsumeSession("one more", time.Now())
		assert.Error(t, err)
	})
}

func TestPass_CanRefund_RefundSession(t *testing.T) {
	t.Run("cannot_refund_when_full", func(t *testing.T) {
		p := newTestPass(t)
		assert.False(t, p.CanRefund())
		_, err := p.RefundSession("refund", time.Now())
		assert.Error(t, err)
	})

	t.Run("can_refund_after_consume", func(t *testing.T) {
		p := newTestPass(t)
		_, _ = p.ConsumeSession("consume", time.Now())
		assert.True(t, p.CanRefund())
		mov, err := p.RefundSession("Cancellation in time", time.Now())
		assert.NoError(t, err)
		assert.Equal(t, 1, mov.Amount())
		assert.Equal(t, 10, p.RemainingSessions())
	})

	t.Run("empty_reason_returns_error", func(t *testing.T) {
		p := newTestPass(t)
		_, _ = p.ConsumeSession("consume", time.Now())
		_, err := p.RefundSession("", time.Now())
		assert.Error(t, err)
	})
}

func TestNewPassMovement(t *testing.T) {
	now := time.Now()
	t.Run("happy_path", func(t *testing.T) {
		m, err := domain.NewPassMovement(1, 1, -1, "Booking", now)
		assert.NoError(t, err)
		assert.Equal(t, 1, m.ID())
		assert.Equal(t, 1, m.PassID())
		assert.Equal(t, -1, m.Amount())
		assert.Equal(t, "Booking", m.Reason())
		assert.Equal(t, now, m.CreatedAt())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			passID    int
			amount    int
			reason    string
			now       time.Time
			wantInErr string
		}{
			{"negative_id", -1, 1, -1, "r", now, "id must not be negative"},
			{"zero_passID", 1, 0, -1, "r", now, "passID must be greater than 0"},
			{"zero_amount", 1, 1, 0, "r", now, "amount must not be zero"},
			{"empty_reason", 1, 1, -1, "", now, "reason must not be empty"},
			{"zero_time", 1, 1, -1, "r", time.Time{}, "createdAt must be a valid time"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewPassMovement(tt.id, tt.passID, tt.amount, tt.reason, tt.now)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestPassType_IsValid(t *testing.T) {
	assert.True(t, domain.PassGeneric.IsValid())
	assert.True(t, domain.PassSpecial.IsValid())
	assert.False(t, domain.PassType("").IsValid())
	assert.False(t, domain.PassType("OTHER").IsValid())
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
