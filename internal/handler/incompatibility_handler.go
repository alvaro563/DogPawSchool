package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	incompatuc "dogpaw/internal/usecase/incompatibility"
)

type IncompatibilityRegisterer interface {
	Execute(ctx context.Context, input incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error)
}

type IncompatibilityLister interface {
	Execute(ctx context.Context, input incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error)
}

type IncompatibilityGetter interface {
	Execute(ctx context.Context, input incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error)
}

type IncompatibilityModifier interface {
	Execute(ctx context.Context, input incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error)
}

type IncompatibilityDeleter interface {
	Execute(ctx context.Context, input incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error)
}

type IncompatibilityHandler struct {
	register IncompatibilityRegisterer
	list     IncompatibilityLister
	get      IncompatibilityGetter
	modify   IncompatibilityModifier
	delete   IncompatibilityDeleter
}

func NewIncompatibilityHandler(
	register IncompatibilityRegisterer,
	list IncompatibilityLister,
	get IncompatibilityGetter,
	modify IncompatibilityModifier,
	delete IncompatibilityDeleter,
) *IncompatibilityHandler {
	return &IncompatibilityHandler{
		register: register, list: list, get: get, modify: modify, delete: delete,
	}
}

// Register godoc
// @Summary      Register a new incompatibility
// @Description  Creates a new incompatibility category. The name must be unique (case-insensitive).
// @Tags         incompatibilities
// @Accept       json
// @Produce      json
// @Param        body  body      registerIncompatibilityRequest   true  "Incompatibility to create"
// @Success      201   {object}  registerIncompatibilityResponse  "Incompatibility created"
// @Failure      400   {object}  errorResponse                    "Validation error"
// @Failure      409   {object}  errorResponse                    "Name already exists"
// @Failure      500   {object}  errorResponse                    "Internal server error"
// @Router       /api/v1/incompatibilities [post]
func (h *IncompatibilityHandler) Register(c *gin.Context) {
	var request registerIncompatibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	output, err := h.register.Execute(c.Request.Context(), incompatuc.RegisterIncompatibilityInput{
		Name:  request.Name,
		Level: domain.IncompatibilityLevel(request.Level),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, registerIncompatibilityResponse{ID: output.ID})
}

// List godoc
// @Summary      List incompatibilities
// @Description  Returns all incompatibilities, optionally filtered by level.
// @Tags         incompatibilities
// @Produce      json
// @Param        level  query  string  false  "Filter by level (ABSOLUTA, MEDIA, BAJA)"
// @Success      200    {object}  listIncompatibilitiesResponse  "List of incompatibilities"
// @Failure      400    {object}  errorResponse                  "Invalid level filter"
// @Failure      500    {object}  errorResponse                  "Internal server error"
// @Router       /api/v1/incompatibilities [get]
func (h *IncompatibilityHandler) List(c *gin.Context) {
	var levelPtr *domain.IncompatibilityLevel
	if levelString := c.Query("level"); levelString != "" {
		parsedLevel := domain.IncompatibilityLevel(levelString)
		levelPtr = &parsedLevel
	}
	output, err := h.list.Execute(c.Request.Context(), incompatuc.ListIncompatibilitiesInput{Level: levelPtr})
	if err != nil {
		writeError(c, err)
		return
	}
	dtos := make([]incompatibilityDTO, len(output.Incompatibilities))
	for i, incompat := range output.Incompatibilities {
		dtos[i] = toIncompatibilityDTO(incompat)
	}
	c.JSON(http.StatusOK, listIncompatibilitiesResponse{Incompatibilities: dtos, Count: len(dtos)})
}

// GetByID godoc
// @Summary      Get incompatibility by ID
// @Description  Returns a single incompatibility by its ID.
// @Tags         incompatibilities
// @Produce      json
// @Param        id   path      int                       true  "Incompatibility ID"
// @Success      200  {object}  incompatibilityResponse   "Incompatibility found"
// @Failure      400  {object}  errorResponse             "Invalid id"
// @Failure      404  {object}  errorResponse             "Incompatibility not found"
// @Failure      500  {object}  errorResponse             "Internal server error"
// @Router       /api/v1/incompatibilities/{id} [get]
func (h *IncompatibilityHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	output, err := h.get.Execute(c.Request.Context(), incompatuc.GetIncompatibilityInput{ID: id})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toIncompatibilityDTO(output.Incompatibility))
}

// Modify godoc
// @Summary      Patch an incompatibility
// @Description  Partially updates an incompatibility (name and/or level). An empty body is a no-op.
// @Tags         incompatibilities
// @Accept       json
// @Produce      json
// @Param        id    path      int                              true  "Incompatibility ID"
// @Param        body  body      modifyIncompatibilityRequest    true  "Fields to patch"
// @Success      200   {object}  incompatibilityResponse        "Updated incompatibility"
// @Failure      400   {object}  errorResponse                   "Invalid id, body, or validation error"
// @Failure      404   {object}  errorResponse                   "Incompatibility not found"
// @Failure      409   {object}  errorResponse                   "Name already exists"
// @Failure      500   {object}  errorResponse                   "Internal server error"
// @Router       /api/v1/incompatibilities/{id} [patch]
func (h *IncompatibilityHandler) Modify(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request modifyIncompatibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	patch := domain.IncompatibilityPatch{Name: request.Name}
	if request.Level != nil {
		levelValue := domain.IncompatibilityLevel(*request.Level)
		patch.Level = &levelValue
	}
	output, err := h.modify.Execute(c.Request.Context(), incompatuc.ModifyIncompatibilityInput{ID: id, Patch: patch})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toIncompatibilityDTO(output.Incompatibility))
}

// Delete godoc
// @Summary      Delete an incompatibility
// @Description  Deletes an incompatibility. Fails with 409 if it is still referenced by any dog.
// @Tags         incompatibilities
// @Produce      json
// @Param        id   path      int               true  "Incompatibility ID"
// @Success      204  "No content"
// @Failure      400  {object}  errorResponse     "Invalid id"
// @Failure      404  {object}  errorResponse     "Incompatibility not found"
// @Failure      409  {object}  errorResponse     "Incompatibility is in use by at least one dog"
// @Failure      500  {object}  errorResponse     "Internal server error"
// @Router       /api/v1/incompatibilities/{id} [delete]
func (h *IncompatibilityHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	_, err = h.delete.Execute(c.Request.Context(), incompatuc.DeleteIncompatibilityInput{ID: id})
	if err != nil {
		writeError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

type registerIncompatibilityRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=120" example:"Reacciona mal al transportin"`
	Level string `json:"level" binding:"required,oneof=ABSOLUTA MEDIA BAJA" example:"MEDIA"`
}

type registerIncompatibilityResponse struct {
	ID int `json:"id" example:"5"`
}

type listIncompatibilitiesResponse struct {
	Incompatibilities []incompatibilityDTO `json:"incompatibilities"`
	Count             int                  `json:"count" example:"3"`
}

type incompatibilityResponse struct {
	ID    int    `json:"id" example:"3"`
	Name  string `json:"name" example:"Miedo a petardos"`
	Level string `json:"level" example:"BAJA"`
}

type modifyIncompatibilityRequest struct {
	Name  *string `json:"name,omitempty" example:"Miedo a petardos y cohetes"`
	Level *string `json:"level,omitempty" example:"ABSOLUTA"`
}

type incompatibilityDTOAlias = incompatibilityDTO

type _ = incompatibilityDTOAlias
