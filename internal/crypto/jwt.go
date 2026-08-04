package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"dogpaw/internal/domain"
)

// JWTTokenGenerator creates signed JWT tokens using HMAC-SHA256.
// It satisfies authuc.TokenGenerator structurally (the interface is
// verified at the composition root, not here, so authuc is never
// imported to avoid a test cycle: usecase/auth test files import
// crypto, and crypto must not import usecase/auth back).
type JWTTokenGenerator struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTTokenGenerator returns a generator that signs with the given
// secret and sets exp = iat + ttl.
func NewJWTTokenGenerator(secret string, ttl time.Duration) *JWTTokenGenerator {
	return &JWTTokenGenerator{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate creates a signed JWT for user u. The token carries:
//   - sub:           user ID
//   - role:          user role (ADMIN | REGULAR)
//   - token_version: current token version (for revocation on password change)
//   - iat:           issued-at timestamp
//   - exp:           expiration timestamp (now + ttl)
//
// It satisfies authuc.TokenGenerator.
func (g *JWTTokenGenerator) Generate(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":           user.ID(),
		"role":          string(user.Role()),
		"token_version": user.TokenVersion(),
		"iat":           now.Unix(),
		"exp":           now.Add(g.ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

// TokenClaims holds the data extracted from a validated JWT.
type TokenClaims struct {
	UserID       int
	Role         string
	TokenVersion int
}

// ParseToken verifies a signed JWT and extracts the user ID, role, and
// token version. It returns the claims on success or an error if the
// token is expired, malformed, or signed with a different secret.
func ParseToken(tokenString string, secret []byte) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt parse: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid claims")
	}
	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, fmt.Errorf("jwt: missing or invalid sub claim")
	}
	role, ok := claims["role"].(string)
	if !ok {
		return nil, fmt.Errorf("jwt: missing or invalid role claim")
	}
	tokenVersion := 0
	if tv, ok := claims["token_version"]; ok {
		if tvFloat, ok := tv.(float64); ok {
			tokenVersion = int(tvFloat)
		}
	}
	return &TokenClaims{UserID: int(subFloat), Role: role, TokenVersion: tokenVersion}, nil
}
