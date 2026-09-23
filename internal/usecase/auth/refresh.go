package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RefreshOutput mirrors LoginOutput: the caller (handler) maps the
// two tokens to Set-Cookie headers.
type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

// RefreshUseCase validates a refresh JWT, loads the underlying user,
// re-checks token_version (so password changes still invalidate all
// outstanding sessions), and emits a fresh access+refresh pair.
//
// Errors:
//   - domain.ErrRefreshTokenInvalid (or any wrapped form) on a
//     malformed or wrong-kind token.
//   - domain.ErrInvalidCredentials (existing sentinel) when the user
//     does not exist, is inactive, or has bumped token_version.
//
// The composition root wires the JWT secret + the access/refresh
// generators. Only the refresh TokenGenerator is needed to parse +
// re-emit — but we need a parser, so the use case depends on a
// TokenParser interface rather than the concrete crypto package.
type RefreshUseCase struct {
	userRepo   domain.UserRepository
	refreshGen TokenGenerator
	accessGen  TokenGenerator
	parser     TokenParser
}

// TokenParser validates a signed token and returns its claims.
type TokenParser interface {
	Parse(tokenString string) (*TokenClaims, error)
}

// TokenClaims is what the parser returns. Mirrors
// crypto.TokenClaims structurally so the use case never imports the
// crypto package (and the tests can substitute a fake).
type TokenClaims struct {
	UserID       int
	Role         string
	TokenVersion int
	Kind         string
}

func NewRefreshUseCase(
	userRepo domain.UserRepository,
	refreshGen TokenGenerator,
	accessGen TokenGenerator,
	parser TokenParser,
) *RefreshUseCase {
	return &RefreshUseCase{
		userRepo:   userRepo,
		refreshGen: refreshGen,
		accessGen:  accessGen,
		parser:     parser,
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, refreshToken string) (RefreshOutput, error) {
	claims, err := uc.parser.Parse(refreshToken)
	if err != nil {
		return RefreshOutput{}, fmt.Errorf("%w: %v", ErrRefreshTokenInvalid, err)
	}
	if claims.Kind != "refresh" {
		return RefreshOutput{}, ErrRefreshTokenInvalid
	}

	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RefreshOutput{}, ErrInvalidCredentials
		}
		return RefreshOutput{}, fmt.Errorf("refresh: get user: %w", err)
	}
	if user == nil {
		return RefreshOutput{}, ErrInvalidCredentials
	}
	if !user.CanLogin() {
		return RefreshOutput{}, ErrInvalidCredentials
	}
	if user.TokenVersion() != claims.TokenVersion {
		// The user changed their password (or admin deactivated
		// the user via /auth/logout-all — future). All
		// outstanding sessions are dead.
		return RefreshOutput{}, ErrInvalidCredentials
	}

	now := time.Now()
	_ = now // reserved for future "issued at" audits

	access, err := uc.accessGen.Generate(user)
	if err != nil {
		return RefreshOutput{}, fmt.Errorf("refresh: generate access: %w", err)
	}
	refresh, err := uc.refreshGen.Generate(user)
	if err != nil {
		return RefreshOutput{}, fmt.Errorf("refresh: generate refresh: %w", err)
	}

	return RefreshOutput{AccessToken: access, RefreshToken: refresh, User: user}, nil
}
