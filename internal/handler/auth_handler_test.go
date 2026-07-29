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

func newTestAuthHandler(registerer UserRegisterer) *AuthHandler {
	return NewAuthHandler(registerer)
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

func TestRegisterWithInvitation_Success(t *testing.T) {
	t.Parallel()
	u := newRegisteredUser()
	h := newTestAuthHandler(&stubUserRegisterer{
		fn: func(_ context.Context, in authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error) {
			assert.Equal(t, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", in.Token())
			assert.Equal(t, "Alice", in.Name())
			return authuc.RegisterWithInvitationOutput{User: u}, nil
		},
	})
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
	})
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
	})
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
	})
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
	})
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
	})
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
	})
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
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/auth/register", validRegisterWithInvitationBody())

	h.RegisterWithInvitation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
