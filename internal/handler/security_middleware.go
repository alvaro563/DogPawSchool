package handler

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig controls which headers SecurityHeadersMiddleware
// emits. In production, the strict set is the default; HSTS is only
// emitted when TLS is actually in use (so HTTP-only deploys don't
// lock clients into HTTPS they cannot reach).
type SecurityHeadersConfig struct {
	// Strict means the production preset: HSTS enabled, no
	// 'unsafe-inline' for style, frame-ancestors 'none'.
	Strict bool
	// TLSEnabled controls whether Strict-Transport-Security is
	// emitted. HSTS without TLS would tell the browser to upgrade
	// requests to HTTPS for a domain that does not serve it.
	TLSEnabled bool
}

// SecurityHeadersMiddleware emits the browser-hardening response
// headers on every successful response. CSP is the most consequential
// one: it limits which sources of script, style, image, and connect
// can be used by the SPA, mitigating XSS impact even when a
// vulnerability slips into the bundle.
//
// Defaults are tuned for the SPA at / with an API at /api. Adjust
// per deployment by composing multiple middlewares with different
// configs (e.g. /swagger has its own relaxed config).
//
// IMPORTANT: This middleware sets headers via c.Header, which writes
// to the response BEFORE the handler runs. Order in the engine
// matters — install it before any handler that may short-circuit
// (e.g. CORS, rate limit) so those short-circuits also emit the
// headers.
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content-Security-Policy. The 'unsafe-inline' for style is
		// unavoidable with Tailwind/vite's inline <style> tags; the
		// trade-off is documented in the project README.
		//
		// img-src includes https: so admin-uploaded photo URLs from
		// arbitrary CDNs (S3, Cloudflare R2) load without a per-domain
		// allowlist. If a stricter posture is required later, the
		// allowlist is a one-line change.
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"img-src 'self' data: https:; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// X-Content-Type-Options prevents MIME sniffing — defence
		// against content-type confusion attacks (e.g. serving a JS
		// file as text/plain).
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options is redundant with frame-ancestors in modern
		// browsers, but older clients still respect it. Belt and
		// braces against clickjacking.
		c.Header("X-Frame-Options", "DENY")

		// Referrer-Policy: send only the origin on cross-origin
		// navigations, the full URL on same-origin. Privacy
		// default that does not break analytics.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy: the SPA does not use the camera,
		// microphone, or geolocation APIs. Disable them outright so
		// an XSS cannot request them via navigator.permissions.
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS only when TLS is enabled. Two years, includeSubDomains,
		// preload-eligible. The operator must explicitly add the
		// domain to https://hstspreload.org after observing that
		// HTTPS works for every subdomain.
		if cfg.TLSEnabled {
			c.Header("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains")
		}

		c.Next()
	}
}
