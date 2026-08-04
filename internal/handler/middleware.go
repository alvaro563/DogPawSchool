package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
)

// AuthRequired returns a Gin middleware that validates a Bearer JWT from
// the Authorization header. On success it sets "user_id" (int),
// "user_role" (string), and "token_version" (int) in the Gin context.
// Additionally, it confirms the user still exists in the database, is
// active, and that the token's version matches the user's current
// version (revoked tokens have a stale version after a password change).
// On any failure it aborts with 401.
func AuthRequired(secret string, userRepo domain.UserRepository) gin.HandlerFunc {
	secretBytes := []byte(secret)
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}

		tokenString, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
			return
		}

		claims, err := crypto.ParseToken(tokenString, secretBytes)
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
