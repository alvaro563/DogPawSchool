package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

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

func newTestAuthHandler(registerer UserRegisterer, logger UserLogger, passwordChanger PasswordChanger) *AuthHandler {
	return NewAuthHandler(registerer, logger, passwordChanger)
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
			return authuc.RegisterWithInvitationOutput{User: u}, nil
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
	assert.NotContains(t, w.Body.String(), "password", "password must never appear in the response")
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
			return authuc.LoginOutput{Token: "jwt-header.payload.signature", User: u}, nil
		},
	}, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/login", validLoginBody())

	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body loginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "jwt-header.payload.signature", body.Token)
	assert.Equal(t, 42, body.User.ID)
	assert.Equal(t, "Alice", body.User.Name)
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
