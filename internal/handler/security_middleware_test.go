package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSecurityHeaders_AppliedToAllResponses verifies every
// response (including 404 short-circuits) carries the
// browser-hardening headers.
func TestSecurityHeaders_AppliedToAllResponses(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	cfg := SecurityHeadersConfig{Strict: true, TLSEnabled: true}
	r := gin.New()
	r.Use(SecurityHeadersMiddleware(cfg))
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/missing", func(c *gin.Context) {
		c.String(http.StatusNotFound, "nope")
	})

	for _, path := range []string{"/x", "/missing"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		h := w.Header()
		assert.NotEmpty(t, h.Get("Content-Security-Policy"),
			"%s: CSP must be present", path)
		assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
		assert.Contains(t, h.Get("Permissions-Policy"), "camera=()")
		assert.NotEmpty(t, h.Get("Strict-Transport-Security"))
	}
}

// TestSecurityHeaders_NoHSTSWithoutTLS confirms the HSTS header is
// only emitted when TLS is actually in use. HSTS without TLS would
// tell the browser to upgrade to HTTPS for a domain that doesn't
// serve it, leaving the deploy unreachable.
func TestSecurityHeaders_NoHSTSWithoutTLS(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	cfg := SecurityHeadersConfig{Strict: true, TLSEnabled: false}
	r := gin.New()
	r.Use(SecurityHeadersMiddleware(cfg))
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"),
		"HSTS must NOT be emitted when TLSEnabled is false")
}

// TestBodyLimit_DoesNotCapGET verifies the body limit applies only
// to mutating methods.
func TestBodyLimit_DoesNotCapGET(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimitMiddleware())
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestBodyLimit_RejectsOversizedPOST simulates a request whose
// Content-Length header exceeds the cap. MaxBytesReader also trips
// if the client streams without Content-Length, but Content-Length
// is the cheap pre-read path.
func TestBodyLimit_RejectsOversizedPOST(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimitMiddleware())
	r.POST("/x", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	w := httptest.NewRecorder()
	// 2 MiB > 1 MiB cap.
	bigBody := bytes.Repeat([]byte("a"), 2<<20)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(bigBody))
	req.ContentLength = int64(len(bigBody))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

// TestBodyLimit_AcceptsSmallPOST is the positive counterpart.
func TestBodyLimit_AcceptsSmallPOST(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimitMiddleware())
	r.POST("/x", func(c *gin.Context) {
		// Drain the body so MaxBytesReader doesn't linger.
		_, _ = io.Copy(io.Discard, c.Request.Body)
		c.String(http.StatusOK, "ok")
	})
	w := httptest.NewRecorder()
	body := bytes.Repeat([]byte("a"), 1024)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
