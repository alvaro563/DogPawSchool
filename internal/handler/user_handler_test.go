package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	useruc "dogpaw/internal/usecase/user"
)

// ---------------------------------------------------------------------------
// Stubs: minimal types that satisfy the 4 use-case interfaces for tests.
// ---------------------------------------------------------------------------

type stubUserGetter struct {
	fn func(ctx context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error)
}

func (s *stubUserGetter) Execute(ctx context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error) {
	return s.fn(ctx, in)
}

type stubUserLister struct {
	fn func(ctx context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error)
}

func (s *stubUserLister) Execute(ctx context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error) {
	return s.fn(ctx, in)
}

type stubUserUpdater struct {
	fn func(ctx context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error)
}

func (s *stubUserUpdater) Execute(ctx context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
	return s.fn(ctx, in)
}

type stubUserDeactivator struct {
	fn func(ctx context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error)
}

func (s *stubUserDeactivator) Execute(ctx context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
	return s.fn(ctx, in)
}

type stubUserEmailLister struct {
	fn func(ctx context.Context) (useruc.ListUserEmailsOutput, error)
}

func (s *stubUserEmailLister) Execute(ctx context.Context) (useruc.ListUserEmailsOutput, error) {
	return s.fn(ctx)
}

// ---------------------------------------------------------------------------
// Handler constructors that inject only the stub for the endpoint under test.
// ---------------------------------------------------------------------------

func newTestUserHandlerGet(get UserGetter) *UserHandler {
	return NewUserHandler(get, nil, nil, nil, nil)
}

func newTestUserHandlerList(list UserLister) *UserHandler {
	return NewUserHandler(nil, list, nil, nil, nil)
}

func newTestUserHandlerUpdate(update UserUpdater) *UserHandler {
	return NewUserHandler(nil, nil, update, nil, nil)
}

func newTestUserHandlerDeactivate(deactivate UserDeactivator) *UserHandler {
	return NewUserHandler(nil, nil, nil, deactivate, nil)
}

func newTestUserHandlerListEmails(emailList UserEmailLister) *UserHandler {
	return NewUserHandler(nil, nil, nil, nil, emailList)
}

// newTestUser builds a valid active user for stub responses.
func newTestUser(id int) *domain.User {
	u, err := domain.NewUser(id, "Test User", "test@example.com", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestUserGetByID_Success(t *testing.T) {
	t.Parallel()
	u := newTestUser(7)
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(_ context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error) {
		assert.Equal(t, 7, in.ID())
		return useruc.GetUserOutput{User: u}, nil
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/7", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "7"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 7, body.ID)
	assert.Equal(t, "Test User", body.Name)
	assert.Equal(t, "test@example.com", body.Email)
	assert.Equal(t, "REGULAR", body.Role)
	assert.True(t, body.IsActive)
	assert.NotContains(t, w.Body.String(), "password", "password must never appear in the response")
}

func TestUserGetByID_InvalidID(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(context.Context, useruc.GetUserInput) (useruc.GetUserOutput, error) {
		t.Fatal("use case should not be called")
		return useruc.GetUserOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/abc", "")
	c.Params = gin.Params{{Key: "user_id", Value: "abc"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"validation"`)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestUserGetByID_UseCaseValidation(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(_ context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error) {
		return useruc.GetUserOutput{}, &useruc.ValidationError{Field: "id"}
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestUserGetByID_NotFound(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(_ context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error) {
		return useruc.GetUserOutput{}, useruc.ErrNotFound
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/999", "", withUserID(999))
	c.Params = gin.Params{{Key: "user_id", Value: "999"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

func TestUserGetByID_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(_ context.Context, in useruc.GetUserInput) (useruc.GetUserOutput, error) {
		return useruc.GetUserOutput{}, errors.New("db down")
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

func TestUserGetByID_Forbidden(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerGet(&stubUserGetter{fn: func(context.Context, useruc.GetUserInput) (useruc.GetUserOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return useruc.GetUserOutput{}, nil
	}})
	// Current user is 7, requesting user_id=99
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/99", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}}
	h.GetByID(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestUserList_Success(t *testing.T) {
	t.Parallel()
	users := []*domain.User{newTestUser(1), newTestUser(2)}
	h := newTestUserHandlerList(&stubUserLister{fn: func(_ context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error) {
		assert.Equal(t, 10, in.Limit())
		assert.Equal(t, 0, in.Offset())
		return useruc.ListUsersOutput{Users: users}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users?limit=10&offset=0", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listUsersResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Users, 2)
	assert.Equal(t, 10, body.Limit)
	assert.Equal(t, 0, body.Offset)
	assert.Equal(t, 2, body.Count)
	assert.NotContains(t, w.Body.String(), "password", "password must never appear in the list response")
}

func TestUserList_PaginationPassesThrough(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerList(&stubUserLister{fn: func(_ context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error) {
		assert.Equal(t, 25, in.Limit(), "factory should normalize and pass through limit")
		assert.Equal(t, 50, in.Offset())
		return useruc.ListUsersOutput{Users: []*domain.User{}}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users?limit=25&offset=50", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserList_Empty(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerList(&stubUserLister{fn: func(_ context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error) {
		return useruc.ListUsersOutput{Users: []*domain.User{}}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"users":[]`)
}

func TestUserList_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerList(&stubUserLister{fn: func(_ context.Context, in useruc.ListUsersInput) (useruc.ListUsersOutput, error) {
		return useruc.ListUsersOutput{}, errors.New("connection lost")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users", "")

	h.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// ListEmails
// ---------------------------------------------------------------------------

func TestUserListEmails_Success(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerListEmails(&stubUserEmailLister{fn: func(_ context.Context) (useruc.ListUserEmailsOutput, error) {
		return useruc.ListUserEmailsOutput{Emails: []string{"a@example.com", "b@example.com"}}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/emails", "")

	h.ListEmails(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listUserEmailsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, []string{"a@example.com", "b@example.com"}, body.Emails)
	assert.Equal(t, 2, body.Count)
	assert.NotContains(t, w.Body.String(), "password", "password must never appear in the response")
}

func TestUserListEmails_Empty(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerListEmails(&stubUserEmailLister{fn: func(_ context.Context) (useruc.ListUserEmailsOutput, error) {
		return useruc.ListUserEmailsOutput{Emails: []string{}}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/emails", "")

	h.ListEmails(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"emails":[]`)
	assert.Contains(t, w.Body.String(), `"count":0`)
}

func TestUserListEmails_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerListEmails(&stubUserEmailLister{fn: func(_ context.Context) (useruc.ListUserEmailsOutput, error) {
		return useruc.ListUserEmailsOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/emails", "")

	h.ListEmails(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserUpdate_Success_Name(t *testing.T) {
	t.Parallel()
	var captured useruc.UpdateUserInput
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		captured = in
		return useruc.UpdateUserOutput{ID: 42}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"name":"Ana Such"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body updateUserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
	require.NotNil(t, captured.Patch().Name)
	assert.Equal(t, "Ana Such", *captured.Patch().Name)
	assert.Nil(t, captured.Patch().Email, "email must remain nil if absent from body")
}

func TestUserUpdate_Success_Email(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		return useruc.UpdateUserOutput{ID: 42}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"email":"new@example.com"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserUpdate_Success_Both(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		require.NotNil(t, in.Patch().Name)
		require.NotNil(t, in.Patch().Email)
		return useruc.UpdateUserOutput{ID: 42}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"name":"Luna","email":"luna@dogpaw.es"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserUpdate_EmptyBody_Noop(t *testing.T) {
	t.Parallel()
	var capturedPatch domain.UserPatch
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		capturedPatch = in.Patch()
		return useruc.UpdateUserOutput{ID: 42}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedPatch.IsEmpty(), "empty body must produce an empty patch; use case then short-circuits (no DB write) internally")
}

func TestUserUpdate_InvalidID(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(context.Context, useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		t.Fatal("use case should not be called")
		return useruc.UpdateUserOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/abc", `{"name":"X"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "abc"}}
	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestUserUpdate_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(context.Context, useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		t.Fatal("use case should not be called")
		return useruc.UpdateUserOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `not json`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"invalid_request"`)
}

func TestUserUpdate_UseCaseValidation(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		return useruc.UpdateUserOutput{}, &useruc.ValidationError{Field: "email"}
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"email":"not-an-email"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"email"`)
}

func TestUserUpdate_NotFound(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		return useruc.UpdateUserOutput{}, useruc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/999", `{"name":"X"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "999"}}
	h.Update(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

func TestUserUpdate_DuplicateEmail(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		return useruc.UpdateUserOutput{}, domain.ErrDuplicateEmail
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"email":"taken@example.com"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"duplicate_email"`)
}

func TestUserUpdate_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerUpdate(&stubUserUpdater{fn: func(_ context.Context, in useruc.UpdateUserInput) (useruc.UpdateUserOutput, error) {
		return useruc.UpdateUserOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/users/42", `{"name":"X"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}
	h.Update(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// Deactivate
// ---------------------------------------------------------------------------

func TestUserDeactivate_Success(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerDeactivate(&stubUserDeactivator{fn: func(_ context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
		assert.Equal(t, 9, in.ID())
		return useruc.DeactivateUserOutput{ID: 9}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/9/deactivate", "")
	c.Params = gin.Params{{Key: "user_id", Value: "9"}}
	h.Deactivate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body deactivateUserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 9, body.ID)
	assert.False(t, body.IsActive, "deactivate must report is_active=false")
}

func TestUserDeactivate_InvalidID(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerDeactivate(&stubUserDeactivator{fn: func(context.Context, useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
		t.Fatal("use case should not be called")
		return useruc.DeactivateUserOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/abc/deactivate", "")
	c.Params = gin.Params{{Key: "user_id", Value: "abc"}}
	h.Deactivate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestUserDeactivate_NotFound(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerDeactivate(&stubUserDeactivator{fn: func(_ context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
		return useruc.DeactivateUserOutput{}, useruc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/999/deactivate", "")
	c.Params = gin.Params{{Key: "user_id", Value: "999"}}
	h.Deactivate(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserDeactivate_AlreadyInactive(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerDeactivate(&stubUserDeactivator{fn: func(_ context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
		return useruc.DeactivateUserOutput{ID: 9}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/9/deactivate", "")
	c.Params = gin.Params{{Key: "user_id", Value: "9"}}
	h.Deactivate(c)

	assert.Equal(t, http.StatusOK, w.Code, "idempotent: deactivating an already-inactive user is 200")
	assert.Contains(t, w.Body.String(), `"is_active":false`)
}

func TestUserDeactivate_InternalError(t *testing.T) {
	t.Parallel()
	h := newTestUserHandlerDeactivate(&stubUserDeactivator{fn: func(_ context.Context, in useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error) {
		return useruc.DeactivateUserOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/9/deactivate", "")
	c.Params = gin.Params{{Key: "user_id", Value: "9"}}
	h.Deactivate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
