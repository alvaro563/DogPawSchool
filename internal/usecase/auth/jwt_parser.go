package auth

import (
	"errors"

	"dogpaw/internal/crypto"
)

// JWTTokenParser is the production implementation of TokenParser. It
// adapts the JWT-parsing concerns (signature verification, expiry,
// kind claim) to the use case layer. The expected kind is fixed at
// construction because RefreshUseCase only ever parses refresh
// tokens; kind mismatches collapse into ErrRefreshTokenInvalid so
// the handler can map them to 401 without exposing which check
// failed.
type JWTTokenParser struct {
	secret       []byte
	expectedKind crypto.TokenKind
}

// NewJWTTokenParser builds a parser bound to the given secret and
// expected kind. Use crypto.KindRefresh for the /auth/refresh flow;
// KindAccess would be a misuse (refresh tokens are validated only
// by /auth/refresh).
func NewJWTTokenParser(secret string, expectedKind crypto.TokenKind) *JWTTokenParser {
	return &JWTTokenParser{secret: []byte(secret), expectedKind: expectedKind}
}

// Parse validates and decodes a JWT. Errors collapse into a single
// domain-level sentinel so the use case never imports crypto.
func (p *JWTTokenParser) Parse(tokenString string) (*TokenClaims, error) {
	claims, err := crypto.ParseToken(tokenString, p.secret, p.expectedKind)
	if err != nil {
		// crypto.ErrWrongTokenKind, jwt.ErrTokenExpired,
		// jwt.ErrTokenSignatureInvalid, ... all collapse here.
		return nil, ErrRefreshTokenInvalid
	}
	if errors.Is(err, crypto.ErrWrongTokenKind) {
		return nil, ErrRefreshTokenInvalid
	}
	return &TokenClaims{
		UserID:       claims.UserID,
		Role:         claims.Role,
		TokenVersion: claims.TokenVersion,
		Kind:         string(claims.Kind),
	}, nil
}
