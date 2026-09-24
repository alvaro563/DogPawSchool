package handler

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
)

// AuthRequired returns a Gin middleware that validates an access
// token from the access_token HttpOnly cookie. On success it sets
// "user_id" (int), "user_role" (string), and "token_version" (int)
// in the Gin context. Additionally, it confirms the user still
// exists in the database, is active, and that the token's version
// matches the user's current version (revoked tokens have a stale
// version after a password change). On any failure it aborts with
// 401.
//
// Cookie-based (instead of the previous Authorization: Bearer
// header) so the JWT never sits in JavaScript-accessible storage.
// The SPA's http-client uses credentials: 'include' so the cookie
// travels automatically; the SPA itself never reads the token.
//
// The "access" kind check prevents an attacker who somehow obtains
// a refresh token from presenting it as an access token (the kind
// claim differentiates them).
func AuthRequired(secret string, userRepo domain.UserRepository) gin.HandlerFunc {
	secretBytes := []byte(secret)
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(CookieAccess)
		if err != nil || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}

		claims, err := crypto.ParseToken(tokenString, secretBytes, crypto.KindAccess)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}
		if !user.IsActive() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}
		if user.TokenVersion() != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// AdminRequired returns a Gin middleware that aborts with 403 Forbidden
// if the authenticated user's role is not "ADMIN". Must be used after
// AuthRequired so that "user_role" is already set in the context.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != string(domain.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
			return
		}
		c.Next()
	}
}

// CurrentUserID returns the authenticated user's ID from the Gin
// context, previously set by AuthRequired. Returns 0 if not set.
func CurrentUserID(c *gin.Context) int {
	id, _ := c.Get("user_id")
	userID, _ := id.(int)
	return userID
}

// CurrentUserRole returns the authenticated user's role from the Gin
// context, previously set by AuthRequired. Returns "" if not set.
func CurrentUserRole(c *gin.Context) string {
	role, _ := c.Get("user_role")
	r, _ := role.(string)
	return r
}

// IsAdmin returns true when the authenticated user has the "ADMIN" role.
func IsAdmin(c *gin.Context) bool {
	return CurrentUserRole(c) == string(domain.RoleAdmin)
}

// ClientIP returns the key for per-account rate limits (login
// lockout). It delegates to gin's c.ClientIP(), which honours
// TRUSTED_PROXIES:
//
//   - TRUSTED_PROXIES unset (default): the TCP peer is never
//     treated as a proxy, X-Forwarded-For is ignored entirely, and
//     the raw RemoteAddr host is returned — an untrusted client
//     cannot spoof its rate-limit key.
//   - TRUSTED_PROXIES set (production behind the SPA host's /api
//     proxy): gin walks X-Forwarded-For from right to left and
//     returns the first untrusted hop — the real client — so two
//     different users behind the same proxy IP do NOT share a
//     lockout bucket.
//
// Falls back to the raw RemoteAddr string when gin cannot parse it
// (test fixtures sometimes use non-IP peers); the caller keys on
// whatever string comes back.
func ClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	if c.Request.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil && host != "" {
			return host
		}
		return c.Request.RemoteAddr
	}
	return ""
}

// forbidden writes a 403 Forbidden response and aborts the request.
func forbidden(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
}

// RequireOwnershipOrAdmin aborts with 403 and returns false when the
// authenticated user is not an admin AND their user_id does not match
// resourceUserID. This is the standard guard for endpoints in the
// "any authenticated user" group where regular users may only access
// their own resources.
func RequireOwnershipOrAdmin(c *gin.Context, resourceUserID int) bool {
	if IsAdmin(c) {
		return true
	}
	if CurrentUserID(c) != resourceUserID {
		forbidden(c)
		return false
	}
	return true
}
