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

type AuthHandler struct {
	registerer UserRegisterer
}

func NewAuthHandler(registerer UserRegisterer) *AuthHandler {
	return &AuthHandler{registerer: registerer}
}

type registerWithInvitationRequest struct {
	Token    string `json:"token"    binding:"required"`
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerWithInvitationResponse struct {
	User userDTO `json:"user"`
}

// RegisterWithInvitation godoc
// @Summary      Register a new user with an invitation token
// @Description  Completes user registration using a valid invitation token. The token must be in PENDING status and not expired (48h lifetime). The password must be at least 60 characters (bcrypt output length). Returns the created user profile without the password hash.
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
