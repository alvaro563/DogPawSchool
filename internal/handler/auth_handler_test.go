package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	authuc "dogpaw/internal/usecase/auth"
)

type stubUserRegisterer struct {
	fn func(ctx context.Context, in authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error)
}

func (s *stubUserRegisterer) Execute(ctx context.Context, in authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
	return s.fn(ctx, in)
}

type stubUserLogger struct {
	fn func(ctx context.Context, in authuc.LoginInput) (authuc.LoginOutput, error)
}

func (s *stubUserLogger) Execute(ctx context.Context, in authuc.LoginInput) (authuc.LoginOutput, error) {
	return s.fn(ctx, in)
}

type stubPasswordChanger struct {
	fn func(ctx context.Context, in authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error)
}

func (s *stubPasswordChanger) Execute(ctx context.Context, in authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
	return s.fn(ctx, in)
}

func newTestAuthHandler(registerer UserRegisterer, logger UserLogger, passwordChanger PasswordChanger, refreshers ...*stubTokenRefresher) *AuthHandler {
	refresher := &stubTokenRefresher{}
	if len(refreshers) > 0 {
		refresher = refreshers[0]
	}
	return NewAuthHandler(
		registerer, logger, passwordChanger,
		refresher,
		CookieConfig{Secure: false, AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour},
	)
}

// stubTokenRefresher returns a valid output on any input. Used only
// in tests that exercise Login/Register/Logout flows — those do not
// exercise the refresh path.
type stubTokenRefresher struct {
	fn func(ctx context.Context, refreshToken string) (authuc.RefreshOutput, error)
}

func (s *stubTokenRefresher) Execute(ctx context.Context, refreshToken string) (authuc.RefreshOutput, error) {
	if s.fn != nil {
		return s.fn(ctx, refreshToken)
	}
	return authuc.RefreshOutput{}, errors.New("stubTokenRefresher: not used in this test")
}

func validRegisterWithInvitationBody() string {
	return `{"token":"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2","name":"Alice","password":"securepassword123"}`
}

func newRegisteredUser() *domain.User {
	u, err := domain.NewUser(1, "Alice", "alice@example.com", "hashed_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

func validLoginBody() string {
	return `{"email":"alice@dogpaw.com","password":"securepassword123"}`
}

func newLoggedInUser() *domain.User {
	u, err := domain.NewUser(42, "Alice", "alice@dogpaw.com", "hashed-pw-60chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

func TestRegisterWithInvitation_Success(t *testing.T) {
	t.Parallel()
	u := newRegisteredUser()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(_ context.Context, in authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			assert.Equal(t, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", in.Token())
			assert.Equal(t, "Alice", in.Name())
			return authuc.RegisterWithInvitationOutput{User: u, AccessToken: "jwt-token", RefreshToken: "jwt-refresh"}, nil
		},
	}, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", validRegisterWithInvitationBody())

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body registerWithInvitationResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.User.ID)
	assert.Equal(t, "Alice", body.User.Name)
	assert.Equal(t, "alice@example.com", body.User.Email)
	assert.Equal(t, "REGULAR", body.User.Role)
	assert.True(t, body.User.IsActive)
	// Tokens travel in Set-Cookie headers, never in the body.
	var regCookies []string
	for _, ck := range w.Result().Cookies() {
		regCookies = append(regCookies, ck.Name+"="+ck.Value)
	}
	assert.Contains(t, regCookies, "access_token=jwt-token")
	assert.Contains(t, regCookies, "refresh_token=jwt-refresh")
	assert.NotContains(t, w.Body.String(), "password", "password must never appear in the response")
}

// The SPA and the API share one origin in production (the SPA host
// proxies /api/* — see render.yaml), so both session cookies must be
// SameSite=Lax: first-party on every fetch, absent on cross-site
// POST (CSRF vector). SameSite=None would additionally require
// Secure and exposes the session to third-party-cookie blocking.
func TestSetAuthCookies_SameSiteLaxAndHttpOnly(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", "")

	h.setAuthCookies(c, "access-jwt", "refresh-jwt")

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, ck := range cookies {
		assert.Equal(t, http.SameSiteLaxMode, ck.SameSite,
			"cookie %s must be SameSite=Lax", ck.Name)
		assert.True(t, ck.HttpOnly, "cookie %s must be HttpOnly", ck.Name)
		assert.Equal(t, "/", ck.Path, "cookie %s must be Path=/", ck.Name)
	}
}

func TestRegisterWithInvitation_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.RegisterWithInvitationOutput{}, nil
		},
	}, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", `not-json`)

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterWithInvitation_EmptyToken(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.RegisterWithInvitationOutput{}, nil
		},
	}, nil, nil)
	body := `{"token":"","name":"Alice","password":"securepassword123"}`
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", body)

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterWithInvitation_EmptyName(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.RegisterWithInvitationOutput{}, nil
		},
	}, nil, nil)
	body := `{"token":"valid-token","name":"","password":"securepassword123"}`
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", body)

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogout_ClearsBothCookies(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/logout", "")

	h.Logout(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// Both cookies must be expired (MaxAge=-1).
	for _, ck := range w.Result().Cookies() {
		if ck.Name == CookieAccess || ck.Name == CookieRefresh {
			assert.Equal(t, -1, ck.MaxAge,
				"cookie %s must be expired on logout", ck.Name)
		}
	}
}

func TestRefresh_NoCookie(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, nil, &stubTokenRefresher{})
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/refresh", "")
	h.Refresh(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_HappyPath_IssuesNewCookies(t *testing.T) {
	t.Parallel()
	user := newLoggedInUser()
	h := newTestAuthHandler(nil, nil, nil, &stubTokenRefresher{
		fn: func(_ context.Context, refreshToken string) (authuc.RefreshOutput, error) {
			assert.NotEmpty(t, refreshToken, "handler must forward the cookie value")
			return authuc.RefreshOutput{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				User:         user,
			}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/refresh", "")
	c.Request.AddCookie(&http.Cookie{Name: CookieRefresh, Value: "old-refresh"})

	h.Refresh(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var cookies []string
	for _, ck := range w.Result().Cookies() {
		cookies = append(cookies, ck.Name+"="+ck.Value)
	}
	assert.Contains(t, cookies, "access_token=new-access")
	assert.Contains(t, cookies, "refresh_token=new-refresh")
}

func TestRegisterWithInvitation_ShortPassword(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.RegisterWithInvitationOutput{}, nil
		},
	}, nil, nil)
	body := `{"token":"valid-token","name":"Alice","password":"abc"}`
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", body)

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterWithInvitation_TokenNotFound(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			return authuc.RegisterWithInvitationOutput{}, authuc.ErrNotFound
		},
	}, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", validRegisterWithInvitationBody())

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterWithInvitation_InvitationInvalid(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			return authuc.RegisterWithInvitationOutput{}, domain.ErrInvitationInvalid
		},
	}, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", validRegisterWithInvitationBody())

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegisterWithInvitation_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(context.Context, authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			return authuc.RegisterWithInvitationOutput{}, errors.New("db connection failed")
		},
	}, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", validRegisterWithInvitationBody())

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Login handler tests ---

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	u := newLoggedInUser()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(_ context.Context, in authuc.LoginInput) (authuc.LoginOutput, error) {
			assert.Equal(t, "alice@dogpaw.com", in.Email())
			return authuc.LoginOutput{AccessToken: "jwt-header.payload.signature", RefreshToken: "jwt-refresh", User: u}, nil
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body loginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.User.ID)
	assert.Equal(t, "Alice", body.User.Name)
	// Tokens travel in Set-Cookie headers, never in the body.
	var loginCookies []string
	for _, ck := range w.Result().Cookies() {
		loginCookies = append(loginCookies, ck.Name+"="+ck.Value)
	}
	assert.Contains(t, loginCookies, "access_token=jwt-header.payload.signature")
	assert.Contains(t, loginCookies, "refresh_token=jwt-refresh")
	assert.NotContains(t, w.Body.String(), "password", "password hash must never appear in the response")
}

func TestLogin_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.LoginOutput{}, nil
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", `not-json`)

	h.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_EmptyEmail(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.LoginOutput{}, nil
		},
	}, nil)
	body := `{"email":"","password":"secret"}`
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", body)

	h.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_EmptyPassword(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.LoginOutput{}, nil
		},
	}, nil)
	body := `{"email":"alice@dogpaw.com","password":""}`
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", body)

	h.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			return authuc.LoginOutput{}, authuc.ErrInvalidCredentials
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_InactiveUser(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			return authuc.LoginOutput{}, authuc.ErrUserInactive
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(context.Context, authuc.LoginInput) (authuc.LoginOutput, error) {
			return authuc.LoginOutput{}, errors.New("db connection failed")
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- ChangePassword handler tests ---

func validChangePasswordBody() string {
	return `{"old_password":"current-password","new_password":"new-secure-password"}`
}

func TestChangePassword_Success(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(_ context.Context, in authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			assert.Equal(t, 42, in.UserID())
			assert.Equal(t, "current-password", in.OldPassword())
			return authuc.ChangePasswordOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", validChangePasswordBody())
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body changePasswordResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "password_updated", body.Message)
}

func TestChangePassword_MissingUserInContext(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.ChangePasswordOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", validChangePasswordBody())
	// Do NOT set user_id

	h.ChangePassword(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePassword_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.ChangePasswordOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", `not-json`)
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_EmptyOldPassword(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.ChangePasswordOutput{}, nil
		},
	})
	body := `{"old_password":"","new_password":"new-secure-password"}`
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", body)
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_ShortNewPassword(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			t.Fatal("use case should not be called")
			return authuc.ChangePasswordOutput{}, nil
		},
	})
	body := `{"old_password":"current","new_password":"short"}`
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", body)
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_InvalidCredentials(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			return authuc.ChangePasswordOutput{}, authuc.ErrInvalidCredentials
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", validChangePasswordBody())
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePassword_SamePassword(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			return authuc.ChangePasswordOutput{}, authuc.ErrSamePassword
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", validChangePasswordBody())
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestChangePassword_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestAuthHandler(nil, nil, &stubPasswordChanger{
		fn: func(context.Context, authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error) {
			return authuc.ChangePasswordOutput{}, errors.New("db connection failed")
		},
	})
	c, w := setupCtx(http.MethodPatch, "/api/v1/auth/password", validChangePasswordBody())
	c.Set("user_id", 42)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogin_LockoutReturns429WithRetryAfter(t *testing.T) {
	t.Parallel()

	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(_ context.Context, _ authuc.LoginInput) (authuc.LoginOutput, error) {
			return authuc.LoginOutput{}, &domain.ErrAccountLockout{
				RetryAfter: 7 * time.Minute,
				Reason:     domain.LockoutReasonEmail,
			}
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())
	c.Request.RemoteAddr = "203.0.113.7:51234"

	h.Login(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "420", w.Header().Get("Retry-After"),
		"Retry-After must be ceil(7m) = 420s")
	assert.Equal(t, "420", w.Header().Get("RateLimit-Reset"))

	var body errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "account_locked", body.Error)
	assert.Contains(t, body.Details, "email")
}

func TestLogin_LockoutByIPUsesRetryAfterFloor(t *testing.T) {
	t.Parallel()

	// Sub-second retry-after must be rounded up to 1 (the
	// minimum integer the IETF draft allows on Retry-After).
	h := newTestAuthHandler(nil, &stubUserLogger{
		fn: func(_ context.Context, _ authuc.LoginInput) (authuc.LoginOutput, error) {
			return authuc.LoginOutput{}, &domain.ErrAccountLockout{
				RetryAfter: 100 * time.Millisecond,
				Reason:     domain.LockoutReasonIP,
			}
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "1", w.Header().Get("Retry-After"))
}
