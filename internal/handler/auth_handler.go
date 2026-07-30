package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	authuc "dogpaw/internal/usecase/auth"
)

type UserRegisterer interface {
	Execute(ctx context.Context, input authuc.RegisterWithInvitationInput) (authuc.RegisterWithInvitationOutput, error)
}

type UserLogger interface {
	Execute(ctx context.Context, input authuc.LoginInput) (authuc.LoginOutput, error)
}

type PasswordChanger interface {
	Execute(ctx context.Context, input authuc.ChangePasswordInput) (authuc.ChangePasswordOutput, error)
}

type AuthHandler struct {
	registerer      UserRegisterer
	logger          UserLogger
	passwordChanger PasswordChanger
}

func NewAuthHandler(registerer UserRegisterer, logger UserLogger, passwordChanger PasswordChanger) *AuthHandler {
	return &AuthHandler{registerer: registerer, logger: logger, passwordChanger: passwordChanger}
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

type loginResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

// RegisterWithInvitation godoc
// @Summary      Register a new user with an invitation token
// @Description  Completes user registration using a valid invitation token. The token must be in PENDING status and not expired (48h lifetime). The password must be at least 8 characters. Returns the created user profile without the password hash.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerWithInvitationRequest  true  "Registration data"
// @Success      201   {object}  registerWithInvitationResponse  "User registered"
// @Failure      400   {object}  errorResponse                   "Invalid request body, missing fields, or validation error (e.g. short password)"
// @Failure      404   {object}  errorResponse                   "Token not found"
// @Failure      409   {object}  errorResponse                   "Token already used, expired, or revoked"
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

	c.JSON(http.StatusCreated, registerWithInvitationResponse{
		User: toUserDTO(output.User),
	})
}

// Login godoc
// @Summary      Authenticate user and return a JWT token
// @Description  Authenticates a user with email and password. On success it returns a signed JWT (HS256) and the user profile. The token expires after 24 hours and carries the user ID (sub) and role (role) claims.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login credentials"
// @Success      200   {object}  loginResponse  "Login successful"
// @Failure      400   {object}  errorResponse  "Invalid request body (malformed JSON, missing email, missing password)"
// @Failure      401   {object}  errorResponse  "Invalid credentials or inactive user"
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

	in, err := authuc.NewLoginInput(req.Email, req.Password, nil)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.logger.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		Token: output.Token,
		User:  toUserDTO(output.User),
	})
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
// @Description  Verifies the current password and replaces it with a new one. Requires a valid Bearer JWT in the Authorization header. The new password must be at least 8 characters and different from the current one.
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

	in, err := authuc.NewChangePasswordInput(userID.(int), req.OldPassword, req.NewPassword, nil)
	if err != nil {
		writeError(c, err)
		return
	}

	_, err = h.passwordChanger.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, changePasswordResponse{Message: "password_updated"})
}
