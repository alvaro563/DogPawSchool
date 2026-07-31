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
// @Description  Creates a new incompatibility category (a TRAIT with a
// @Description  stable code, or a TRIGGER pointing at the code of the
// @Description  trait it reacts to). The name must be unique
// @Description  (case-insensitive); trait codes must be unique too.
// @Tags         incompatibilities
// @Accept       json
// @Produce      json
// @Param        body  body      registerIncompatibilityRequest   true  "Incompatibility to create"
// @Success      201   {object}  registerIncompatibilityResponse  "Incompatibility created"
// @Failure      400   {object}  errorResponse                    "Validation error"
// @Failure      409   {object}  errorResponse                    "Name or trait code already exists"
// @Failure      500   {object}  errorResponse                    "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/incompatibilities [post]
func (h *IncompatibilityHandler) Register(c *gin.Context) {
	var request registerIncompatibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	kind := domain.IncompatibilityKindTrigger
	if request.Kind != "" {
		kind = domain.IncompatibilityKind(request.Kind)
	}
	in, err := incompatuc.NewRegisterIncompatibilityInput(
		request.Name, domain.IncompatibilityLevel(request.Level), kind, request.Code, request.TargetTraitCode,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.register.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, registerIncompatibilityResponse{ID: output.ID})
}

// List godoc
// @Summary      List incompatibilities
// @Description  Returns all incompatibilities, optionally filtered by
// @Description  level and/or kind.
// @Tags         incompatibilities
// @Produce      json
// @Param        level  query  string  false  "Filter by level (ABSOLUTA, MEDIA, BAJA)"
// @Param        kind   query  string  false  "Filter by kind (TRAIT, TRIGGER)"
// @Success      200    {object}  listIncompatibilitiesResponse  "List of incompatibilities"
// @Failure      400    {object}  errorResponse                  "Invalid level or kind filter"
// @Failure      500    {object}  errorResponse                  "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/incompatibilities [get]
func (h *IncompatibilityHandler) List(c *gin.Context) {
	var levelPtr *domain.IncompatibilityLevel
	if levelString := c.Query("level"); levelString != "" {
		parsedLevel := domain.IncompatibilityLevel(levelString)
		levelPtr = &parsedLevel
	}
	var kindPtr *domain.IncompatibilityKind
	if kindString := c.Query("kind"); kindString != "" {
		parsedKind := domain.IncompatibilityKind(kindString)
		kindPtr = &parsedKind
	}
	in, err := incompatuc.NewListIncompatibilitiesInput(levelPtr, kindPtr)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.list.Execute(c.Request.Context(), in)
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
// @Security     BearerAuth
// @Router       /api/v1/incompatibilities/{id} [get]
func (h *IncompatibilityHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	in, err := incompatuc.NewGetIncompatibilityInput(id)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.get.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toIncompatibilityDTO(output.Incompatibility))
}

// Modify godoc
// @Summary      Patch an incompatibility
// @Description  Partially updates an incompatibility (name, level, kind,
// @Description  code and/or target_trait_code). An empty body is a no-op.
// @Tags         incompatibilities
// @Accept       json
// @Produce      json
// @Param        id    path      int                              true  "Incompatibility ID"
// @Param        body  body      modifyIncompatibilityRequest    true  "Fields to patch"
// @Success      200   {object}  incompatibilityResponse        "Updated incompatibility"
// @Failure      400   {object}  errorResponse                   "Invalid id, body, or validation error"
// @Failure      404   {object}  errorResponse                   "Incompatibility not found"
// @Failure      409   {object}  errorResponse                   "Name or trait code already exists"
// @Failure      500   {object}  errorResponse                   "Internal server error"
// @Security     BearerAuth
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
	patch := domain.IncompatibilityPatch{Name: request.Name, Code: request.Code, TargetTraitCode: request.TargetTraitCode}
	if request.Level != nil {
		levelValue := domain.IncompatibilityLevel(*request.Level)
		patch.Level = &levelValue
	}
	if request.Kind != nil {
		kindValue := domain.IncompatibilityKind(*request.Kind)
		patch.Kind = &kindValue
	}
	in, err := incompatuc.NewModifyIncompatibilityInput(id, patch)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.modify.Execute(c.Request.Context(), in)
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
// @Security     BearerAuth
// @Router       /api/v1/incompatibilities/{id} [delete]
func (h *IncompatibilityHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	in, err := incompatuc.NewDeleteIncompatibilityInput(id)
	if err != nil {
		writeError(c, err)
		return
	}
	_, err = h.delete.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

type registerIncompatibilityRequest struct {
	Name            string `json:"name" example:"Reacciona mal al transportin"`
	Level           string `json:"level" example:"MEDIA"`
	Kind            string `json:"kind,omitempty" example:"TRIGGER"`
	Code            string `json:"code,omitempty" example:"MIEDOSO"`
	TargetTraitCode string `json:"target_trait_code,omitempty" example:"MACHO_ENTERO"`
}

type registerIncompatibilityResponse struct {
	ID int `json:"id" example:"5"`
}

type listIncompatibilitiesResponse struct {
	Incompatibilities []incompatibilityDTO `json:"incompatibilities"`
	Count             int                  `json:"count" example:"3"`
}

type modifyIncompatibilityRequest struct {
	Name            *string `json:"name,omitempty" example:"Miedo a petardos y cohetes"`
	Level           *string `json:"level,omitempty" example:"ABSOLUTA"`
	Kind            *string `json:"kind,omitempty" example:"TRAIT"`
	Code            *string `json:"code,omitempty" example:"MIEDOSO"`
	TargetTraitCode *string `json:"target_trait_code,omitempty" example:"MACHO_ENTERO"`
}

// incompatibilityResponse is the wire format for a single
// incompatibility on the incompatibilities endpoints.
type incompatibilityResponse struct {
	ID              int    `json:"id" example:"3"`
	Name            string `json:"name" example:"Miedo a petardos"`
	Level           string `json:"level" example:"BAJA"`
	Kind            string `json:"kind" example:"TRIGGER"`
	Code            string `json:"code,omitempty" example:"MIEDOSO"`
	TargetTraitCode string `json:"target_trait_code,omitempty" example:"MACHO_ENTERO"`
}
