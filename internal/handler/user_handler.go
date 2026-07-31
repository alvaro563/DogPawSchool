package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	useruc "dogpaw/internal/usecase/user"
)

// UserGetter loads a single user by id.
type UserGetter interface {
	Execute(ctx context.Context, input useruc.GetUserInput) (useruc.GetUserOutput, error)
}

// UserLister lists all users (admin view) with pagination.
type UserLister interface {
	Execute(ctx context.Context, input useruc.ListUsersInput) (useruc.ListUsersOutput, error)
}

// UserUpdater applies a partial update (name and/or email) to a user.
type UserUpdater interface {
	Execute(ctx context.Context, input useruc.UpdateUserInput) (useruc.UpdateUserOutput, error)
}

// UserDeactivator flips the is_active flag to false (soft delete).
type UserDeactivator interface {
	Execute(ctx context.Context, input useruc.DeactivateUserInput) (useruc.DeactivateUserOutput, error)
}

// UserEmailLister lists every registered email (admin view).
type UserEmailLister interface {
	Execute(ctx context.Context) (useruc.ListUserEmailsOutput, error)
}

// UserHandler owns the 5 user endpoints. All use cases are injected
// as interfaces so the handler can be unit-tested with stubs and so
// the dependency direction stays correct (handler -> usecase interface,
// never -> usecase concrete, never -> repository).
type UserHandler struct {
	get        UserGetter
	list       UserLister
	update     UserUpdater
	deactivate UserDeactivator
	emailList  UserEmailLister
}

func NewUserHandler(
	get UserGetter,
	list UserLister,
	update UserUpdater,
	deactivate UserDeactivator,
	emailList UserEmailLister,
) *UserHandler {
	return &UserHandler{
		get:        get,
		list:       list,
		update:     update,
		deactivate: deactivate,
		emailList:  emailList,
	}
}

// GetByID godoc
// @Summary      Get a user by ID
// @Description  Returns the public profile of a user. The password hash is never included in the response.
// @Tags         users
// @Produce      json
// @Param        user_id  path  int  true  "User ID"
// @Success      200  {object}  userDTO
// @Failure      400  {object}  errorResponse  "Invalid id"
// @Failure      404  {object}  errorResponse  "User not found"
// @Failure      500  {object}  errorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}

	if !RequireOwnershipOrAdmin(c, id) {
		return
	}

	in, err := useruc.NewGetUserInput(id)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.get.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(output.User))
}

// List godoc
// @Summary      List all users (admin view)
// @Description  Returns a paginated list of every user in the system. Intended for the admin panel. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         users
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of users to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of users to skip for pagination (default 0)"
// @Success      200  {object}  listUsersResponse
// @Failure      500  {object}  errorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	in, _ := useruc.NewListUsersInput(limit, offset)
	output, err := h.list.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListUsersResponse(output.Users, in))
}

// ListEmails godoc
// @Summary      List all user emails (admin view)
// @Description  Returns the email of every registered user, ordered by user id. Intended for the admin panel (e.g. bulk communications). Admin only.
// @Tags         users
// @Produce      json
// @Success      200  {object}  listUserEmailsResponse
// @Failure      500  {object}  errorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/emails [get]
func (h *UserHandler) ListEmails(c *gin.Context) {
	output, err := h.emailList.Execute(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, listUserEmailsResponse{
		Emails: output.Emails,
		Count:  len(output.Emails),
	})
}

// listUserEmailsResponse is the wire format for the email list
// endpoint. The count field mirrors the pagination-aware lists so the
// client can render totals without parsing the array.
type listUserEmailsResponse struct {
	Emails []string `json:"emails"`
	Count  int      `json:"count"`
}

// Update godoc
// @Summary      Patch a user (name and/or email)
// @Description  Applies a partial update to an existing user. Only the fields present in the request body are modified; omitted fields are preserved. An empty body is a no-op and returns 200 without touching the database. The password is never writable through this endpoint.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user_id  path      int                true  "User ID"
// @Param        user     body      updateUserRequest  true  "Fields to patch (only the fields you want to change)"
// @Success      200      {object}  updateUserResponse "User patched (or no-op if body was empty)"
// @Failure      400      {object}  errorResponse      "Invalid id, invalid request body, or validation error (e.g. empty name, malformed email)"
// @Failure      404      {object}  errorResponse      "User not found"
// @Failure      409      {object}  errorResponse      "Email already in use"
// @Failure      500      {object}  errorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}

	var request updateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	patch := domain.UserPatch{
		Name:  request.Name,
		Email: request.Email,
	}

	in, err := useruc.NewUpdateUserInput(id, patch)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.update.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, updateUserResponse{ID: output.ID})
}

// Deactivate godoc
// @Summary      Deactivate a user (soft delete)
// @Description  Flips the user's is_active flag to false. Idempotent: deactivating an already-inactive user is a no-op and returns 200. The user's data is preserved. Use this for off-boarding instead of hard delete.
// @Tags         users
// @Produce      json
// @Param        user_id  path  int  true  "User ID"
// @Success      200      {object}  deactivateUserResponse  "User deactivated (or already inactive)"
// @Failure      400      {object}  errorResponse           "Invalid id"
// @Failure      404      {object}  errorResponse           "User not found"
// @Failure      500      {object}  errorResponse           "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/deactivate [post]
func (h *UserHandler) Deactivate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}

	in, err := useruc.NewDeactivateUserInput(id)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.deactivate.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, deactivateUserResponse{
		ID:       output.ID,
		IsActive: false,
	})
}

// listUsersResponse is the wire format for the list endpoint. Built
// once here so every list handler in the user package uses the same
// envelope: data + pagination metadata + count.
type listUsersResponse struct {
	Users  []userDTO `json:"users"`
	Limit  int       `json:"limit"  example:"50"`
	Offset int       `json:"offset" example:"0"`
	Count  int       `json:"count"  example:"10"`
}

// userPagination is the contract every user list input satisfies:
// exposes the (already normalized) limit and offset.
type userPagination interface {
	Limit() int
	Offset() int
}

// toListUsersResponse serializes a slice of users to the standard
// listUsersResponse envelope.
func toListUsersResponse(users []*domain.User, in userPagination) listUsersResponse {
	dtos := make([]userDTO, len(users))
	for i, user := range users {
		dtos[i] = toUserDTO(user)
	}
	return listUsersResponse{
		Users:  dtos,
		Limit:  in.Limit(),
		Offset: in.Offset(),
		Count:  len(dtos),
	}
}

// userDTO is the wire format of a user. The password hash is
// intentionally NEVER included.
type userDTO struct {
	ID       int    `json:"id"        example:"1"`
	Name     string `json:"name"      example:"Carlos Admin"`
	Email    string `json:"email"     example:"admin@dogpaw.com"`
	Role     string `json:"role"      example:"ADMIN"`
	IsActive bool   `json:"is_active" example:"true"`
}

// toUserDTO converts a domain.User into the HTTP wire format. The
// password hash is dropped at this boundary so it can never leak via
// any endpoint.
func toUserDTO(user *domain.User) userDTO {
	return userDTO{
		ID:       user.ID(),
		Name:     user.Name(),
		Email:    user.Email(),
		Role:     string(user.Role()),
		IsActive: user.IsActive(),
	}
}

type updateUserRequest struct {
	Name  *string `json:"name,omitempty"  example:"Ana Such"`
	Email *string `json:"email,omitempty" example:"ana.such@dogpaw.es"`
}

type updateUserResponse struct {
	ID int `json:"id" example:"1"`
}

type deactivateUserResponse struct {
	ID       int  `json:"id"        example:"9"`
	IsActive bool `json:"is_active" example:"false"`
}
