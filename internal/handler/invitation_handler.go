package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	invitationuc "dogpaw/internal/usecase/invitation"
)

type InvitationCreator interface {
	Execute(ctx context.Context, input invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error)
}

type InvitationHandler struct {
	creator InvitationCreator
}

func NewInvitationHandler(creator InvitationCreator) *InvitationHandler {
	return &InvitationHandler{creator: creator}
}

type createInvitationRequest struct {
	CreatedBy int    `json:"created_by" binding:"required"`
	Email     string `json:"email"      binding:"required"`
	Role      string `json:"role"       binding:"required"`
}

type createInvitationResponse struct {
	ID    int    `json:"id"`
	Token string `json:"token"`
}

// Create godoc
// @Summary      Create an invitation
// @Description  Creates a new PENDING invitation for a client to register. The returned 64-hex token must be delivered to the client's email. The invitation expires in 48 hours. The created_by field must reference an existing admin user id.
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        body  body      createInvitationRequest  true  "Invitation to create"
// @Success      201   {object}  createInvitationResponse  "Invitation created with token"
// @Failure      400   {object}  errorResponse             "Invalid request body, missing fields, or validation error"
// @Failure      409   {object}  errorResponse             "Duplicate token (unlikely — 256-bit random)"
// @Failure      500   {object}  errorResponse             "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/invitations [post]
func (h *InvitationHandler) Create(c *gin.Context) {
	var req createInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	role := domain.UserRole(req.Role)

	in, err := invitationuc.NewCreateInvitationInput(req.CreatedBy, req.Email, role, nil)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.creator.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createInvitationResponse{
		ID:    output.ID,
		Token: output.Token,
	})
}
