package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestPass_IsExpired(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

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
			p := &domain.Pass{ExpiresAt: tt.expiresAt}
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
			p := &domain.Pass{RemainingSessions: tt.remaining}
			assert.Equal(t, tt.expected, p.IsExhausted())
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
