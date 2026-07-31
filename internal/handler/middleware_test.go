package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiddlewareCtx(userID int, role string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("user_role", role)
	return c, w
}

func TestAdminRequired_AllowsAdmin(t *testing.T) {
	t.Parallel()
	c, w := setupMiddlewareCtx(1, "ADMIN")
	handler := AdminRequired()
	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, c.IsAborted())
}

func TestAdminRequired_BlocksRegular(t *testing.T) {
	t.Parallel()
	c, w := setupMiddlewareCtx(7, "REGULAR")
	handler := AdminRequired()
	handler(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, c.IsAborted())
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "forbidden", resp.Error)
}

func TestAdminRequired_BlocksMissingRole(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No role set in context
	handler := AdminRequired()
	handler(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, c.IsAborted())
}

func TestCurrentUserID_ReturnsZeroWhenMissing(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, 0, CurrentUserID(c))
}

func TestCurrentUserID_ReturnsSetValue(t *testing.T) {
	t.Parallel()
	c, _ := setupMiddlewareCtx(42, "REGULAR")
	assert.Equal(t, 42, CurrentUserID(c))
}

func TestCurrentUserRole_ReturnsEmptyWhenMissing(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, "", CurrentUserRole(c))
}

func TestCurrentUserRole_ReturnsSetValue(t *testing.T) {
	t.Parallel()
	c, _ := setupMiddlewareCtx(1, "ADMIN")
	assert.Equal(t, "ADMIN", CurrentUserRole(c))
}

func TestIsAdmin_TrueForAdmin(t *testing.T) {
	t.Parallel()
	c, _ := setupMiddlewareCtx(1, "ADMIN")
	assert.True(t, IsAdmin(c))
}

func TestIsAdmin_FalseForRegular(t *testing.T) {
	t.Parallel()
	c, _ := setupMiddlewareCtx(7, "REGULAR")
	assert.False(t, IsAdmin(c))
}

func TestRequireOwnershipOrAdmin_SameUser(t *testing.T) {
	t.Parallel()
	c, w := setupMiddlewareCtx(7, "REGULAR")
	ok := RequireOwnershipOrAdmin(c, 7)
	assert.True(t, ok)
	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireOwnershipOrAdmin_DifferentUser_Blocks(t *testing.T) {
	t.Parallel()
	c, w := setupMiddlewareCtx(7, "REGULAR")
	ok := RequireOwnershipOrAdmin(c, 99)
	assert.False(t, ok)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "forbidden", resp.Error)
}

func TestRequireOwnershipOrAdmin_AdminCanAccessAny(t *testing.T) {
	t.Parallel()
	c, w := setupMiddlewareCtx(1, "ADMIN")
	ok := RequireOwnershipOrAdmin(c, 99)
	assert.True(t, ok)
	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, w.Code)
}
