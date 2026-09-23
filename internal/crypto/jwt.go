package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"dogpaw/internal/domain"
)

// TokenKind discriminates access and refresh tokens. The two share a
// secret and the same signing algorithm, but the refresh token has
// a longer TTL and is rejected by the access-token-only AuthRequired
// middleware.
type TokenKind string

const (
	KindAccess  TokenKind = "access"
	KindRefresh TokenKind = "refresh"
)

// JWTTokenGenerator creates signed JWT tokens using HMAC-SHA256.
// It satisfies authuc.TokenGenerator structurally (the interface is
// verified at the composition root, not here, so authuc is never
// imported to avoid a test cycle: usecase/auth test files import
// crypto, and crypto must not import usecase/auth back).
type JWTTokenGenerator struct {
	secret []byte
	ttl    time.Duration
	kind   TokenKind
}

// NewJWTTokenGenerator returns a generator that signs with the given
// secret and sets exp = iat + ttl. Kind is the token kind emitted
// into the "kind" claim; the same secret is shared by both kinds.
func NewJWTTokenGenerator(secret string, ttl time.Duration, kind TokenKind) *JWTTokenGenerator {
	return &JWTTokenGenerator{
		secret: []byte(secret),
		ttl:    ttl,
		kind:   kind,
	}
}

// Generate creates a signed JWT for user u. The token carries:
//   - sub:           user ID
//   - role:          user role (ADMIN | REGULAR)
//   - token_version: current token version (for revocation on password change)
//   - kind:          "access" or "refresh"
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
		"kind":          string(g.kind),
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
	Kind         TokenKind
}

// ErrWrongTokenKind is returned by ParseToken when the token's "kind"
// claim does not match the kind the caller expected. Used by the
// AuthRequired middleware (rejects refresh tokens) and the
// /auth/refresh handler (rejects access tokens).
var ErrWrongTokenKind = errors.New("jwt: wrong token kind")

// ParseToken verifies a signed JWT and extracts the user ID, role,
// token version, and kind. It returns the claims on success or an
// error if the token is expired, malformed, signed with a different
// secret, or has a different kind than expected.
//
// expectedKind: pass KindAccess or KindRefresh to enforce the kind
// at parse time. Pass empty string to skip the check.
func ParseToken(tokenString string, secret []byte, expectedKind TokenKind) (*TokenClaims, error) {
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
	kind := TokenKind("")
	if k, ok := claims["kind"].(string); ok {
		kind = TokenKind(k)
	}
	if expectedKind != "" && kind != expectedKind {
		return nil, ErrWrongTokenKind
	}
	return &TokenClaims{UserID: int(subFloat), Role: role, TokenVersion: tokenVersion, Kind: kind}, nil
}
