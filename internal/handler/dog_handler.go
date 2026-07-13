package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	doguc "dogpaw/internal/usecase/dog"
	incompatuc "dogpaw/internal/usecase/incompatibility"
)

type DogRegistrar interface {
	Execute(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error)
}

type DogListerByOwner interface {
	Execute(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

type DogModifier interface {
	Execute(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error)
}

type DogIncompatibilityAdder interface {
	Execute(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error)
}

type DogIncompatibilityRemover interface {
	Execute(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error)
}

type DogHandler struct {
	register       DogRegistrar
	list           DogListerByOwner
	modify         DogModifier
	addIncompat    DogIncompatibilityAdder
	removeIncompat DogIncompatibilityRemover
}

func NewDogHandler(
	register DogRegistrar,
	list DogListerByOwner,
	modify DogModifier,
	addIncompat DogIncompatibilityAdder,
	removeIncompat DogIncompatibilityRemover,
) *DogHandler {
	return &DogHandler{
		register:       register,
		list:           list,
		modify:         modify,
		addIncompat:    addIncompat,
		removeIncompat: removeIncompat,
	}
}

// Register godoc
// @Summary      Register a new dog
// @Description  Creates a new dog record owned by a user. The new resource URL is returned in the Location header.
// @Tags         dogs
// @Accept       json
// @Produce      json
// @Param        dog  body      registerDogRequest  true  "Dog to register"
// @Success      201  {object}  registerDogResponse  "Dog created successfully"
// @Failure      400  {object}  errorResponse         "Invalid request body, missing fields, validation error, or invalid user_id"
// @Failure      409  {object}  errorResponse         "Passport already exists"
// @Failure      500  {object}  errorResponse         "Internal server error"
// @Router       /api/v1/dogs [post]
func (h *DogHandler) Register(c *gin.Context) {
	var req registerDogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	out, err := h.register.Execute(c.Request.Context(), doguc.RegisterDogInput{
		Name:        req.Name,
		Breed:       req.Breed,
		AgeInMonths: req.AgeInMonths,
		Sex:         domain.Sex(req.Sex),
		WeightKg:    req.WeightKg,
		Passport:    req.Passport,
		UserID:      req.UserID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Location", fmt.Sprintf("/api/v1/dogs/%d", out.ID))
	c.JSON(http.StatusCreated, registerDogResponse{ID: out.ID})
}

// List godoc
// @Summary      List dogs owned by a user
// @Description  Returns a paginated list of dogs belonging to the specified owner. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        owner_id  query  int  true   "ID of the owner user"
// @Param        limit     query  int  false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset    query  int  false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse  "List of dogs with pagination metadata"
// @Failure      400  {object}  errorResponse      "Invalid owner_id"
// @Failure      500  {object}  errorResponse      "Internal server error"
// @Router       /api/v1/dogs [get]
func (h *DogHandler) List(c *gin.Context) {
	ownerID, err := strconv.Atoi(c.Query("owner_id"))
	if err != nil || ownerID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "owner_id",
		})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.list.Execute(c.Request.Context(), doguc.ListByOwnerInput{
		OwnerID: ownerID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	dtos := make([]dogDTO, len(out.Dogs))
	for i, d := range out.Dogs {
		dtos[i] = toDogDTO(d)
	}

	normalizedLimit, normalizedOffset := doguc.NormalizePagination(limit, offset)
	c.JSON(http.StatusOK, listDogsResponse{
		Dogs:   dtos,
		Limit:  normalizedLimit,
		Offset: normalizedOffset,
		Count:  len(dtos),
	})
}

// Modify godoc
// @Summary      Patch a dog (partial update)
// @Description  Applies a partial update to an existing dog. Only the fields present in the request body are modified; omitted fields are preserved. An empty body is a no-op and returns 200 without touching the database. Designed for fixing typos (e.g. "Labarador" -> "Labrador") or correcting registration mistakes.
// @Tags         dogs
// @Accept       json
// @Produce      json
// @Param        id   path      int                true  "Dog ID"
// @Param        dog  body      modifyDogRequest   true  "Fields to patch (only the fields you want to change)"
// @Success      200  {object}  modifyDogResponse  "Dog patched (or no-op if body was empty)"
// @Failure      400  {object}  errorResponse      "Invalid id, invalid request body, or validation error (e.g. empty name, negative weight, invalid sex)"
// @Failure      404  {object}  errorResponse      "Dog not found"
// @Failure      409  {object}  errorResponse      "Passport already exists"
// @Failure      500  {object}  errorResponse      "Internal server error"
// @Router       /api/v1/dogs/{id} [patch]
func (h *DogHandler) Modify(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "id",
		})
		return
	}

	var req modifyDogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	patch := domain.DogPatch{
		Name:          req.Name,
		Breed:         req.Breed,
		AgeInMonths:   req.AgeInMonths,
		Passport:      req.Passport,
		WeightKg:      req.WeightKg,
		Neutered:      req.Neutered,
		Heat:          req.Heat,
		PhotoURL:      req.PhotoURL,
		MedicalNotes:  req.MedicalNotes,
		EducatorNotes: req.EducatorNotes,
		IsActive:      req.IsActive,
	}
	if req.Sex != nil {
		sex := domain.Sex(*req.Sex)
		patch.Sex = &sex
	}

	out, err := h.modify.Execute(c.Request.Context(), doguc.ModifyDogInput{
		ID:    id,
		Patch: patch,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, modifyDogResponse{ID: out.ID})
}

// AddIncompatibility godoc
// @Summary      Add an incompatibility to a dog
// @Description  Idempotently attaches an existing incompatibility (looked up by id) to a dog. If the dog already has that incompatibility, returns 200 with `added: false` and the current list (no DB write). Otherwise persists the change and returns 201 with `added: true` and the updated list. Both the dog and the incompatibility must exist.
// @Tags         dogs
// @Accept       json
// @Produce      json
// @Param        id   path      int                            true  "Dog ID"
// @Param        body body      addIncompatibilityRequest      true  "Incompatibility to attach"
// @Success      201  {object}  addIncompatibilityResponse     "Incompatibility newly attached (added=true)"
// @Success      200  {object}  addIncompatibilityResponse     "Incompatibility was already attached (added=false, idempotent no-op)"
// @Failure      400  {object}  errorResponse                  "Invalid id, invalid body, or validation error"
// @Failure      404  {object}  errorResponse                  "Dog or incompatibility not found"
// @Failure      500  {object}  errorResponse                  "Internal server error"
// @Router       /api/v1/dogs/{id}/incompatibilities [post]
func (h *DogHandler) AddIncompatibility(c *gin.Context) {
	dogID, err := strconv.Atoi(c.Param("id"))
	if err != nil || dogID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "id",
		})
		return
	}

	var req addIncompatibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	out, err := h.addIncompat.Execute(c.Request.Context(), doguc.AddDogIncompatibilityInput{
		DogID:             dogID,
		IncompatibilityID: req.IncompatibilityID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	status := http.StatusOK
	if out.Added {
		status = http.StatusCreated
	}
	c.JSON(status, addIncompatibilityResponse{
		DogID:             out.ID,
		Added:             out.Added,
		Incompatibilities: toIncompatibilityDTOs(out.Incompatibilities),
	})
}

// RemoveIncompatibility godoc
// @Summary      Remove an incompatibility from a dog
// @Description  Idempotently detaches an existing incompatibility (looked up by id) from a dog. If the dog does not have that incompatibility, returns 200 with the current list (no DB write). Otherwise persists the change and returns 200 with the updated list. Both dog and incompatibility must exist; 404 otherwise.
// @Tags         dogs
// @Produce      json
// @Param        id             path      int                    true  "Dog ID"
// @Param        incompatibility_id  path  int                       true  "Incompatibility ID"
// @Success      200  {object}  removeIncompatibilityResponse  "Incompatibility removed (or no-op if not present), with the current list"
// @Failure      400  {object}  errorResponse                    "Invalid id or incompatibility_id"
// @Failure      404  {object}  errorResponse                    "Dog not found"
// @Failure      500  {object}  errorResponse                    "Internal server error"
// @Router       /api/v1/dogs/{id}/incompatibilities/{incompatibility_id} [delete]
func (h *DogHandler) RemoveIncompatibility(c *gin.Context) {
	dogID, err := strconv.Atoi(c.Param("id"))
	if err != nil || dogID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "id",
		})
		return
	}

	incompatID, err := strconv.Atoi(c.Param("incompatibility_id"))
	if err != nil || incompatID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "incompatibility_id",
		})
		return
	}

	out, err := h.removeIncompat.Execute(c.Request.Context(), doguc.RemoveDogIncompatibilityInput{
		DogID:             dogID,
		IncompatibilityID: incompatID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, removeIncompatibilityResponse{
		DogID:             out.ID,
		Removed:           out.Removed,
		Incompatibilities: toIncompatibilityDTOs(out.Incompatibilities),
	})
}

func writeError(c *gin.Context, err error) {
	var dogVerr *doguc.ValidationError
	var incompVerr *incompatuc.ValidationError
	if errors.As(err, &dogVerr) || errors.As(err, &incompVerr) {
		var field string
		if dogVerr != nil {
			field = dogVerr.Field
		} else {
			field = incompVerr.Field
		}
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: field,
		})
		return
	}
	if errors.Is(err, doguc.ErrNotFound) || errors.Is(err, incompatuc.ErrNotFound) || errors.Is(err, postgres.ErrNotFound) {
		c.JSON(http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if errors.Is(err, postgres.ErrInvalidUser) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_user_id"})
		return
	}
	if errors.Is(err, postgres.ErrDuplicatePassport) {
		c.JSON(http.StatusConflict, errorResponse{Error: "duplicate_passport"})
		return
	}
	if errors.Is(err, postgres.ErrIncompatibilityInUse) {
		c.JSON(http.StatusConflict, errorResponse{Error: "incompatibility_in_use"})
		return
	}
	if errors.Is(err, incompatuc.ErrDuplicateName) {
		c.JSON(http.StatusConflict, errorResponse{Error: "duplicate_name"})
		return
	}
	slog.Error("internal error",
		"err", err.Error(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal"})
}

type registerDogRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=120" example:"Luna"`
	Breed       string  `json:"breed" binding:"required,min=1,max=120" example:"Labrador"`
	AgeInMonths int     `json:"age_in_months" binding:"required,gt=0" example:"24"`
	Sex         string  `json:"sex" binding:"required,oneof=MALE FEMALE" example:"FEMALE"`
	WeightKg    float64 `json:"weight_kg" binding:"required,gt=0" example:"22.5"`
	Passport    string  `json:"passport" binding:"required,min=1,max=64" example:"ES-12345"`
	UserID      int     `json:"user_id" binding:"required,gt=0" example:"1"`
}

type registerDogResponse struct {
	ID int `json:"id" example:"42"`
}

type dogDTO struct {
	ID            int     `json:"id" example:"1"`
	Name          string  `json:"name" example:"Luna"`
	Breed         string  `json:"breed" example:"Labrador"`
	AgeInMonths   int     `json:"age_in_months" example:"24"`
	Sex           string  `json:"sex" example:"FEMALE"`
	Neutered      bool    `json:"neutered" example:"false"`
	Heat          bool    `json:"heat" example:"false"`
	WeightKg      float64 `json:"weight_kg" example:"22.5"`
	PhotoURL      string  `json:"photo_url" example:""`
	MedicalNotes  string  `json:"medical_notes" example:""`
	EducatorNotes string  `json:"educator_notes" example:""`
	Passport      string  `json:"passport" example:"ES-12345"`
	UserID        int     `json:"user_id" example:"1"`
	IsActive      bool    `json:"is_active" example:"true"`
}

type listDogsResponse struct {
	Dogs   []dogDTO `json:"dogs"`
	Limit  int      `json:"limit" example:"50"`
	Offset int      `json:"offset" example:"0"`
	Count  int      `json:"count" example:"1"`
}

type modifyDogRequest struct {
	Name          *string  `json:"name,omitempty" example:"Buddie"`
	Breed         *string  `json:"breed,omitempty" example:"Labrador"`
	AgeInMonths   *int     `json:"age_in_months,omitempty" example:"24"`
	Sex           *string  `json:"sex,omitempty" example:"FEMALE"`
	Passport      *string  `json:"passport,omitempty" example:"ES-12345"`
	WeightKg      *float64 `json:"weight_kg,omitempty" example:"22.5"`
	Neutered      *bool    `json:"neutered,omitempty" example:"true"`
	Heat          *bool    `json:"heat,omitempty" example:"false"`
	PhotoURL      *string  `json:"photo_url,omitempty" example:""`
	MedicalNotes  *string  `json:"medical_notes,omitempty" example:""`
	EducatorNotes *string  `json:"educator_notes,omitempty" example:""`
	IsActive      *bool    `json:"is_active,omitempty" example:"true"`
}

type modifyDogResponse struct {
	ID int `json:"id" example:"42"`
}

type addIncompatibilityRequest struct {
	IncompatibilityID int `json:"incompatibility_id" binding:"required,gt=0" example:"3"`
}

type addIncompatibilityResponse struct {
	DogID             int                  `json:"dog_id" example:"42"`
	Added             bool                 `json:"added" example:"true"`
	Incompatibilities []incompatibilityDTO `json:"incompatibilities"`
}

type removeIncompatibilityResponse struct {
	DogID             int                  `json:"dog_id" example:"42"`
	Removed           bool                 `json:"removed" example:"true"`
	Incompatibilities []incompatibilityDTO `json:"incompatibilities"`
}

type incompatibilityDTO struct {
	ID    int    `json:"id" example:"3"`
	Name  string `json:"name" example:"Reactivo a machos enteros"`
	Level string `json:"level" example:"ABSOLUTA"`
}

type errorResponse struct {
	Error   string `json:"error" example:"validation"`
	Field   string `json:"field,omitempty" example:"breed"`
	Details string `json:"details,omitempty" example:"Field 'breed' is required"`
}

func toDogDTO(d *domain.Dog) dogDTO {
	return dogDTO{
		ID:            d.ID(),
		Name:          d.Name(),
		Breed:         d.Breed(),
		AgeInMonths:   d.AgeInMonths(),
		Sex:           string(d.Sex()),
		Neutered:      d.Neutered(),
		Heat:          d.Heat(),
		WeightKg:      d.WeightKg(),
		PhotoURL:      d.PhotoURL(),
		MedicalNotes:  d.MedicalNotes(),
		EducatorNotes: d.EducatorNotes(),
		Passport:      d.Passport(),
		UserID:        d.UserID(),
		IsActive:      d.IsActive(),
	}
}

func toIncompatibilityDTO(in *domain.Incompatibility) incompatibilityDTO {
	return incompatibilityDTO{
		ID:    in.ID(),
		Name:  in.Name(),
		Level: string(in.Type()),
	}
}

func toIncompatibilityDTOs(incompats []domain.Incompatibility) []incompatibilityDTO {
	out := make([]incompatibilityDTO, len(incompats))
	for i, in := range incompats {
		out[i] = toIncompatibilityDTO(&in)
	}
	return out
}
