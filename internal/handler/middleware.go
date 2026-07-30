package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/crypto"
)

// AuthRequired returns a Gin middleware that validates a Bearer JWT from
// the Authorization header. On success it sets "user_id" (int) and
// "user_role" (string) in the Gin context. On failure it aborts with 401.
//
// Usage: r.Group("/protected").Use(handler.AuthRequired(jwtSecret))
func AuthRequired(secret string) gin.HandlerFunc {
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

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}
