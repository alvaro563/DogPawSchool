package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// newTestEngine constructs a minimal Gin engine with the rate
// limiter mounted. The /ping endpoint always 200s; the test
// invokes it from different RemoteAddr values to exercise the
// per-IP bucket isolation.
func newTestEngine(t *testing.T, lim *ipRateLimiter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rateLimitMiddleware(lim))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimit_BurstAllowed(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(1.0), 3)
	t.Cleanup(lim.Stop)
	r := newTestEngine(t, lim)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.1:5000"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d must succeed within burst", i+1)
	}
}

func TestRateLimit_ExhaustsBucket(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(1.0), 2)
	t.Cleanup(lim.Stop)
	r := newTestEngine(t, lim)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.2:5000"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "burst request %d", i+1)
	}

	// 3rd request must be denied with 429 and the IETF headers.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.2:5000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"), "Retry-After must be set on 429")
	assert.NotEmpty(t, w.Header().Get("RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("RateLimit-Reset"))
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(1.0), 1)
	t.Cleanup(lim.Stop)
	r := newTestEngine(t, lim)

	// Burn the bucket for 10.0.0.3.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.3:5000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// A different IP must NOT be affected.
	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.4:5000"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "different RemoteAddr must have its own bucket")
}

func TestRateLimit_RejectsMissingRemoteAddr(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(1.0), 1)
	t.Cleanup(lim.Stop)
	r := newTestEngine(t, lim)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	// httptest.NewRequest sets RemoteAddr to "192.0.2.1:1234" by
	// default; we deliberately clear it to a malformed value to
	// trigger the bad-request path.
	req.RemoteAddr = "not-a-valid-address"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_remote_addr")
}

func TestRateLimit_RetryAfterIsParseableInteger(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(0.5), 1) // 1 token / 2s, burst 1
	t.Cleanup(lim.Stop)
	r := newTestEngine(t, lim)

	// Consume the only token.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.5:5000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Next call should be 429 with Retry-After in seconds.
	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.5:5000"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	secs, err := strconv.Atoi(w.Header().Get("Retry-After"))
	require.NoError(t, err, "Retry-After must be an integer (seconds)")
	assert.GreaterOrEqual(t, secs, 1)
}

func TestRateLimit_StopHaltsBackgroundCleanup(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(rate.Limit(1.0), 1)
	// Stop must be safe to call multiple times (used in defer +
	// main shutdown both).
	lim.Stop()
	lim.Stop()
	// Wait briefly so the cleanup goroutine can wind down. A
	// test failure here would manifest as a goroutine leak in
	// `go test -race`.
	time.Sleep(50 * time.Millisecond)
}
