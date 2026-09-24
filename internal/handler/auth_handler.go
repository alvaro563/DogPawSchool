package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	authuc "dogpaw/internal/usecase/auth"
)

// Cookie names. Defined here so both the handler and any future
// middleware (e.g. server-side logout via a revoked-token list) share
// the same names without a circular dependency.
const (
	CookieAccess  = "access_token"
	CookieRefresh = "refresh_token"
)

// UserRegisterer, UserLogger, PasswordChanger: structural
// interfaces that decouple the handler from the concrete use cases
// (the composition root wires them).
type UserRegisterer interface {
	Execute(ctx context.Context, input authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error)
}

type UserLogger interface {
	Execute(ctx context.Context, input authuc.LoginInput) (authuc.LoginOutput, error)
}

type PasswordChanger interface {
	Execute(ctx context.Context, input authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error)
}

// TokenRefresher validates a refresh token and emits a new pair
// (access + refresh) + the resolved user. Returns
// (user, nil) on success, (nil, error) otherwise.
type TokenRefresher interface {
	Execute(ctx context.Context, refreshToken string) (authuc.RefreshOutput, error)
}

// AuthHandler wires the auth endpoints. Cookies are set on
// successful login/register/refresh/change-password via
// setAuthCookies. The body of those responses deliberately omits the
// token — the SPA must never see the raw JWT (otherwise we might as
// well keep localStorage).
type AuthHandler struct {
	registerer      UserRegisterer
	logger          UserLogger
	passwordChanger PasswordChanger
	refresher       TokenRefresher
	cookieCfg       CookieConfig
}

type CookieConfig struct {
	Secure      bool
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
}

func NewAuthHandler(
	registerer UserRegisterer,
	logger UserLogger,
	passwordChanger PasswordChanger,
	refresher TokenRefresher,
	cookieCfg CookieConfig,
) *AuthHandler {
	return &AuthHandler{
		registerer:      registerer,
		logger:          logger,
		passwordChanger: passwordChanger,
		refresher:       refresher,
		cookieCfg:       cookieCfg,
	}
}

// setAuthCookies emits the access + refresh cookies in the response.
// SameSite=Lax on both: the SPA and the API share one origin (the
// SPA host proxies /api/* — see render.yaml), so every fetch is
// same-site and Lax cookies travel on all of them, while cross-site
// POSTs (CSRF vectors) still get no cookie. HttpOnly + Path=/ so
// the SPA's own routes receive them and JavaScript never sees the
// raw JWT. SameSite=None is deliberately NOT used: it requires
// Secure to be accepted by browsers and only matters for cross-site
// deployments, which the same-origin proxy makes unnecessary.
func (h *AuthHandler) setAuthCookies(c *gin.Context, access, refresh string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieAccess, access, int(h.cookieCfg.AccessTTL.Seconds()), "/", "", h.cookieCfg.Secure, true)
	c.SetCookie(CookieRefresh, refresh, int(h.cookieCfg.RefreshTTL.Seconds()), "/", "", h.cookieCfg.Secure, true)
}

// clearAuthCookies expires both cookies immediately. The browser
// deletes the cookie on the next response regardless of the path /
// domain. Used by Logout.
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieAccess, "", -1, "/", "", h.cookieCfg.Secure, true)
	c.SetCookie(CookieRefresh, "", -1, "/", "", h.cookieCfg.Secure, true)
}

type registerWithInvitationRequest struct {
	Token    string `json:"token"    binding:"required"`
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerWithInvitationResponse struct {
	User userDTO `json:"user"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginResponse returns the user profile + expires_in so the SPA
// can show a "session expires in X" indicator. The token itself is
// NOT in the body — it travels in the Set-Cookie header.
type loginResponse struct {
	User       userDTO `json:"user"`
	ExpiresIn  int     `json:"expires_in"`
}

// RegisterWithInvitation godoc
// @Summary      Register a new user with an invitation token
// @Description  Completes user registration using a valid invitation token. The token must be in PENDING status and not expired (48h lifetime). The password must be at least 8 characters. On success the response sets two HttpOnly cookies (access_token, refresh_token). The SPA never sees the raw JWT. IP rate limited by middleware (separate bucket from login).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerWithInvitationRequest  true  "Registration data"
// @Success      201   {object}  registerWithInvitationResponse  "User registered"
// @Failure      400   {object}  errorResponse                   "Invalid request body, missing fields, or validation error (e.g. short password)"
// @Failure      404   {object}  errorResponse                   "Token not found"
// @Failure      409   {object}  errorResponse                   "Token already used, expired, or revoked"
// @Failure      429   {object}  errorResponse                   "Too many registration attempts from this source. Retry-After header indicates seconds."
// @Failure      500   {object}  errorResponse                   "Internal server error"
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) RegisterWithInvitation(c *gin.Context) {
	var req registerWithInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	in, err := authuc.NewRegisterWithInvitationInput(req.Token, req.Name, req.Password, nil)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.registerer.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setAuthCookies(c, output.AccessToken, output.RefreshToken)
	c.JSON(http.StatusCreated, registerWithInvitationResponse{
		User: toUserDTO(output.User),
	})
}

// Login godoc
// @Summary      Authenticate user with email and password
// @Description  Authenticates a user with email and password. On success it sets two HttpOnly cookies (access_token, refresh_token — both SameSite=Lax) and returns the user profile with the access TTL. After too many failed attempts the account is temporarily locked — a 429 with Retry-After is returned. The SPA never sees the raw JWT.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login credentials"
// @Success      200   {object}  loginResponse  "Login successful"
// @Failure      400   {object}  errorResponse  "Invalid request body (malformed JSON, missing email, missing password)"
// @Failure      401   {object}  errorResponse  "Invalid credentials or inactive user"
// @Failure      429   {object}  errorResponse  "Account temporarily locked — too many failed attempts. Retry-After header indicates seconds until unlock."
// @Failure      500   {object}  errorResponse  "Internal server error"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	in, err := authuc.NewLoginInput(req.Email, req.Password, ClientIP(c), nil)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.logger.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setAuthCookies(c, output.AccessToken, output.RefreshToken)
	c.JSON(http.StatusOK, loginResponse{
		User:      toUserDTO(output.User),
		ExpiresIn: int(h.cookieCfg.AccessTTL.Seconds()),
	})
}

// Refresh godoc
// @Summary      Refresh the session cookies
// @Description  Exchanges a valid refresh_token cookie for a new pair of access_token + refresh_token cookies. Use this when the access_token has expired but the refresh_token is still valid (within 24 hours). Returns 401 if the refresh token is missing, malformed, expired, or revoked (e.g. the user changed their password). Rate limited by IP.
// @Tags         auth
// @Produce      json
// @Success      200   {object}  loginResponse  "Cookies refreshed"
// @Failure      401   {object}  errorResponse  "Refresh token missing, invalid, or expired"
// @Failure      429   {object}  errorResponse  "Too many refresh attempts. Retry-After header indicates seconds."
// @Failure      500   {object}  errorResponse  "Internal server error"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(CookieRefresh)
	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
		return
	}

	output, err := h.refresher.Execute(c.Request.Context(), refreshToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
		return
	}

	h.setAuthCookies(c, output.AccessToken, output.RefreshToken)
	c.JSON(http.StatusOK, loginResponse{
		User:      toUserDTO(output.User),
		ExpiresIn: int(h.cookieCfg.AccessTTL.Seconds()),
	})
}

// Logout godoc
// @Summary      Clear session cookies
// @Description  Clears the access_token and refresh_token cookies. Idempotent — calling it twice returns 204 both times. Does NOT invalidate server-side state (the tokens remain technically valid until they expire); the cookie clearance forces the browser to stop sending them.
// @Tags         auth
// @Produce      json
// @Success      204  "Logged out (cookies cleared)"
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAuthCookies(c)
	c.AbortWithStatus(http.StatusNoContent)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type changePasswordResponse struct {
	Message string `json:"message"`
}

// ChangePassword godoc
// @Summary      Change the authenticated user's password
// @Description  Verifies the current password and replaces it with a new one. Requires a valid access_token cookie. The new password must be at least 8 characters and different from the current one. On success the user's token_version is bumped — all existing tokens (including the current one) are invalidated. Fresh cookies are issued so the SPA stays authenticated.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      changePasswordRequest  true  "Old and new password"
// @Success      200   {object}  changePasswordResponse  "Password changed successfully"
// @Failure      400   {object}  errorResponse            "Invalid request body or validation error (e.g. short new password)"
// @Failure      401   {object}  errorResponse            "Missing or invalid token, wrong old password, or inactive user"
// @Failure      409   {object}  errorResponse            "New password matches the old one"
// @Failure      500   {object}  errorResponse            "Internal server error"
// @Router       /api/v1/auth/password [patch]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	parsedID, ok := userID.(int)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid_credentials"})
		return
	}

	in, err := authuc.NewChangePasswordInput(parsedID, req.OldPassword, req.NewPassword, nil)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.passwordChanger.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	// Issue fresh cookies so the SPA stays logged in (the old
	// access token is dead because token_version was bumped).
	if output.AccessToken != "" && output.RefreshToken != "" {
		h.setAuthCookies(c, output.AccessToken, output.RefreshToken)
	}
	c.JSON(http.StatusOK, changePasswordResponse{Message: "password_updated"})
}
