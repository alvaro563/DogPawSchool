package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	passuc "dogpaw/internal/usecase/pass"
)

type PassRegisterer interface {
	Execute(ctx context.Context, input passuc.RegisterPassInput) (passuc.RegisterPassOutput, error)
}

type PassModifier interface {
	Execute(ctx context.Context, input passuc.ModifyPassInput) (passuc.ModifyPassOutput, error)
}

type PassGetter interface {
	Execute(ctx context.Context, input passuc.GetPassInput) (passuc.GetPassOutput, error)
}

type PassLister interface {
	Execute(ctx context.Context, input passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error)
}

type PassByUserLister interface {
	Execute(ctx context.Context, input passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error)
}

type PassHandler struct {
	register     PassRegisterer
	modify       PassModifier
	getter       PassGetter
	lister       PassLister
	byUserLister PassByUserLister
}

func NewPassHandler(
	register PassRegisterer,
	modify PassModifier,
	getter PassGetter,
	lister PassLister,
	byUserLister PassByUserLister,
) *PassHandler {
	return &PassHandler{
		register:     register,
		modify:       modify,
		getter:       getter,
		lister:       lister,
		byUserLister: byUserLister,
	}
}

// Register godoc
// @Summary      Register a new pass (bono) for a user
// @Description  Creates a new prepaid session pack for the user identified by the user_id path param. Price is stored in cents.
// @Tags         passes
// @Accept       json
// @Produce      json
// @Param        user_id  path      int                     true   "Owner user ID"
// @Param        pass     body      registerPassRequest    true   "Pass to create"
// @Success      201      {object}  registerPassResponse  "Pass created"
// @Failure      400      {object}  errorResponse          "Invalid user_id, request body, or missing fields"
// @Failure      500      {object}  errorResponse          "Internal server error"
// @Router       /api/v1/users/{user_id}/passes [post]
func (h *PassHandler) Register(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	var request registerPassRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}
	output, err := h.register.Execute(c.Request.Context(), passuc.RegisterPassInput{
		NumOfSessions: request.NumOfSessions,
		Price:         request.Price,
		PassType:      domain.PassType(request.PassType),
		UserID:        userID,
		ExpiresAt:     request.ExpiresAt,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Location", "/api/v1/passes/"+strconv.Itoa(output.ID))
	c.JSON(http.StatusCreated, registerPassResponse{ID: output.ID})
}

// Modify godoc
// @Summary      Patch a pass
// @Description  Partially updates a pass. Only price, pass_type, and
// @Description  expires_at are editable. num_of_sessions,
// @Description  remaining_sessions, user_id, and created_at are
// @Description  immutable to preserve the audit-log invariant. An
// @Description  empty body is a no-op.
// @Tags         passes
// @Accept       json
// @Produce      json
// @Param        id    path      int                  true  "Pass ID"
// @Param        pass  body      modifyPassRequest   true  "Fields to patch"
// @Success      200   {object}  passResponse        "Updated pass"
// @Failure      400   {object}  errorResponse       "Invalid id, body, or field value"
// @Failure      404   {object}  errorResponse       "Pass not found"
// @Failure      500   {object}  errorResponse       "Internal server error"
// @Router       /api/v1/passes/{id} [patch]
func (h *PassHandler) Modify(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request modifyPassRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	output, err := h.modify.Execute(c.Request.Context(), passuc.ModifyPassInput{
		ID: id,
		Patch: domain.PassPatch{
			Price:     request.Price,
			PassType:  passTypePtrFromString(request.PassType),
			ExpiresAt: request.ExpiresAt,
		},
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPassDTO(output.Pass))
}

// GetByID godoc
// @Summary      Get a pass by ID
// @Description  Returns a single pass by its id.
// @Tags         passes
// @Produce      json
// @Param        id   path      int           true  "Pass ID"
// @Success      200  {object}  passResponse  "Pass found"
// @Failure      400  {object}  errorResponse "Invalid id"
// @Failure      404  {object}  errorResponse "Pass not found"
// @Failure      500  {object}  errorResponse "Internal server error"
// @Router       /api/v1/passes/{id} [get]
func (h *PassHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	output, err := h.getter.Execute(c.Request.Context(), passuc.GetPassInput{ID: id})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPassDTO(output.Pass))
}

// List godoc
// @Summary      List all passes (admin only in production)
// @Description  Returns a paginated list of all passes in the system.
// @Description  TODO: gate this route behind an admin-role middleware
// @Description  before production. Currently any client can read all
// @Description  passes, which exposes other users' data.
// @Tags         passes
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of passes to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of passes to skip for pagination (default 0)"
// @Success      200  {object}  listPassesResponse  "List of passes"
// @Failure      500  {object}  errorResponse        "Internal server error"
// @Router       /api/v1/passes [get]
func (h *PassHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	output, err := h.lister.Execute(c.Request.Context(), passuc.ListAllPassesInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListPassesResponse(output.Passes, limit, offset))
}

// ListByUser godoc
// @Summary      List passes owned by a user
// @Description  Returns a paginated list of passes owned by the user identified by the user_id path param.
// @Tags         passes
// @Produce      json
// @Param        user_id  path      int  true   "Owner user ID"
// @Param        limit    query     int  false  "Maximum number of passes to return (default 50, max 100)"
// @Param        offset   query     int  false  "Number of passes to skip for pagination (default 0)"
// @Success      200  {object}  listPassesResponse  "List of passes"
// @Failure      400  {object}  errorResponse       "Invalid user_id"
// @Failure      500  {object}  errorResponse       "Internal server error"
// @Router       /api/v1/users/{user_id}/passes [get]
func (h *PassHandler) ListByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	output, err := h.byUserLister.Execute(c.Request.Context(), passuc.ListByUserPassesInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListPassesResponse(output.Passes, limit, offset))
}

type registerPassRequest struct {
	NumOfSessions int        `json:"num_of_sessions" binding:"required,gt=0" example:"10"`
	Price         int        `json:"price"           binding:"gte=0" example:"12000"`
	PassType      string     `json:"pass_type"       binding:"required,oneof=GENERICO ESPECIFICO" example:"GENERICO"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" example:"2026-12-31T23:59:59Z"`
}

type registerPassResponse struct {
	ID int `json:"id" example:"42"`
}

type modifyPassRequest struct {
	Price     *int       `json:"price,omitempty"                binding:"omitempty,gte=0"      example:"15000"`
	PassType  *string    `json:"pass_type,omitempty"            binding:"omitempty,oneof=GENERICO ESPECIFICO" example:"ESPECIFICO"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"           example:"2027-06-30T23:59:59Z"`
}

// listPassesResponse is the wire format shared by List and
// ListByUser.
type listPassesResponse struct {
	Passes []passDTO `json:"passes"`
	Limit  int       `json:"limit"  example:"50"`
	Offset int       `json:"offset" example:"0"`
	Count  int       `json:"count"  example:"3"`
}

// passDTO is the wire format of a pass. The same shape is used for
// both PATCH and (future) GET responses, so the type is exported via
// passResponse below.
type passDTO struct {
	ID                int        `json:"id"                 example:"42"`
	NumOfSessions     int        `json:"num_of_sessions"    example:"10"`
	RemainingSessions int        `json:"remaining_sessions" example:"7"`
	Price             int        `json:"price"              example:"12000"`
	PassType          string     `json:"pass_type"          example:"GENERICO"`
	UserID            int        `json:"user_id"            example:"1"`
	CreatedAt         time.Time  `json:"created_at"         example:"2026-07-01T10:00:00Z"`
	UpdatedAt         time.Time  `json:"updated_at"         example:"2026-07-15T14:30:00Z"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty" example:"2026-12-31T23:59:59Z"`
}

// passResponse is the alias used in Swagger annotations.
type passResponse = passDTO

// toPassDTO converts a domain.Pass into the HTTP wire format.
func toPassDTO(pass *domain.Pass) passDTO {
	return passDTO{
		ID:                pass.ID(),
		NumOfSessions:     pass.NumOfSessions(),
		RemainingSessions: pass.RemainingSessions(),
		Price:             pass.Price(),
		PassType:          string(pass.Type()),
		UserID:            pass.UserID(),
		CreatedAt:         pass.CreatedAt(),
		UpdatedAt:         pass.UpdatedAt(),
		ExpiresAt:         pass.ExpiresAt(),
	}
}

// passTypePtrFromString converts an optional *string to a
// *domain.PassType. The domain.PassType type does not implement
// encoding.TextUnmarshaler with a case-insensitive match for
// Gin's "oneof" tag, so the handler validates the string before
// calling the use case.
func passTypePtrFromString(s *string) *domain.PassType {
	if s == nil {
		return nil
	}
	passType := domain.PassType(*s)
	return &passType
}

// toListPassesResponse builds the list response envelope with
// normalized limit/offset and a non-nil passes slice (so the JSON
// is always "passes":[] and never "passes":null).
func toListPassesResponse(passes []*domain.Pass, limit, offset int) listPassesResponse {
	normalizedLimit, normalizedOffset := passuc.NormalizePagination(limit, offset)
	dtos := make([]passDTO, len(passes))
	for i, pass := range passes {
		dtos[i] = toPassDTO(pass)
	}
	return listPassesResponse{
		Passes: dtos,
		Limit:  normalizedLimit,
		Offset: normalizedOffset,
		Count:  len(dtos),
	}
}
