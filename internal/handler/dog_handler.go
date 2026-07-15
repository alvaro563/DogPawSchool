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

type DogLister interface {
	Execute(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error)
}

type DogListerByOwner interface {
	Execute(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

type DogActiveLister interface {
	Execute(ctx context.Context, in doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error)
}

type DogByIsActiveLister interface {
	Execute(ctx context.Context, in doguc.ListByIsActiveInput) (doguc.ListByIsActiveOutput, error)
}

type DogByIncompatibilityLister interface {
	Execute(ctx context.Context, in doguc.ListByIncompatibilityInput) (doguc.ListByIncompatibilityOutput, error)
}

type DogByBreedLister interface {
	Execute(ctx context.Context, in doguc.ListByBreedInput) (doguc.ListByBreedOutput, error)
}

type DogBySexLister interface {
	Execute(ctx context.Context, in doguc.ListBySexInput) (doguc.ListBySexOutput, error)
}

type DogByNeuteredLister interface {
	Execute(ctx context.Context, in doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error)
}

type DogByHeatLister interface {
	Execute(ctx context.Context, in doguc.ListByHeatInput) (doguc.ListByHeatOutput, error)
}

type DogByAgeBracketLister interface {
	Execute(ctx context.Context, in doguc.ListByAgeBracketInput) (doguc.ListByAgeBracketOutput, error)
}

type DogBySizeBracketLister interface {
	Execute(ctx context.Context, in doguc.ListBySizeBracketInput) (doguc.ListBySizeBracketOutput, error)
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

type DogDeleter interface {
	Execute(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error)
}

type DogHandler struct {
	register              DogRegistrar
	list                  DogLister
	listByOwner           DogListerByOwner
	listActive            DogActiveLister
	listByIsActive        DogByIsActiveLister
	listByIncompatibility DogByIncompatibilityLister
	listByBreed           DogByBreedLister
	listBySex             DogBySexLister
	listByNeutered        DogByNeuteredLister
	listByHeat            DogByHeatLister
	listByAgeBracket      DogByAgeBracketLister
	listBySizeBracket     DogBySizeBracketLister
	modify                DogModifier
	addIncompat           DogIncompatibilityAdder
	removeIncompat        DogIncompatibilityRemover
	delete                DogDeleter
}

func NewDogHandler(
	register DogRegistrar,
	list DogLister,
	listByOwner DogListerByOwner,
	listActive DogActiveLister,
	listByIsActive DogByIsActiveLister,
	listByIncompatibility DogByIncompatibilityLister,
	listByBreed DogByBreedLister,
	listBySex DogBySexLister,
	listByNeutered DogByNeuteredLister,
	listByHeat DogByHeatLister,
	listByAgeBracket DogByAgeBracketLister,
	listBySizeBracket DogBySizeBracketLister,
	modify DogModifier,
	addIncompat DogIncompatibilityAdder,
	removeIncompat DogIncompatibilityRemover,
	deleteDog DogDeleter,
) *DogHandler {
	return &DogHandler{
		register:              register,
		list:                  list,
		listByOwner:           listByOwner,
		listActive:            listActive,
		listByIsActive:        listByIsActive,
		listByIncompatibility: listByIncompatibility,
		listByBreed:           listByBreed,
		listBySex:             listBySex,
		listByNeutered:        listByNeutered,
		listByHeat:            listByHeat,
		listByAgeBracket:      listByAgeBracket,
		listBySizeBracket:     listBySizeBracket,
		modify:                modify,
		addIncompat:           addIncompat,
		removeIncompat:        removeIncompat,
		delete:                deleteDog,
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
// @Summary      List all dogs in the system
// @Description  Returns a paginated list of all dogs across all owners. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse  "List of dogs with pagination metadata"
// @Failure      500  {object}  errorResponse     "Internal server error"
// @Router       /api/v1/dogs [get]
func (h *DogHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.list.Execute(c.Request.Context(), doguc.ListAllDogsInput{
		Limit:  limit,
		Offset: offset,
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

// ListByOwner godoc
// @Summary      List dogs owned by a specific user
// @Description  Returns a paginated list of dogs belonging to the given owner. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        owner_id  path   int  true   "ID of the owner user"
// @Param        limit     query  int  false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset    query  int  false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse  "List of dogs with pagination metadata"
// @Failure      400  {object}  errorResponse     "Invalid owner_id"
// @Failure      500  {object}  errorResponse     "Internal server error"
// @Router       /api/v1/dogs/owner/{owner_id} [get]
func (h *DogHandler) ListByOwner(c *gin.Context) {
	ownerID, err := strconv.Atoi(c.Param("owner_id"))
	if err != nil || ownerID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "owner_id",
		})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByOwner.Execute(c.Request.Context(), doguc.ListByOwnerInput{
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

// ListActive godoc
// @Summary      List active dogs
// @Description  Returns a paginated list of dogs whose is_active is true. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/active [get]
func (h *DogHandler) ListActive(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listActive.Execute(c.Request.Context(), doguc.ListActiveDogsInput{
		Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByIsActive godoc
// @Summary      List dogs filtered by is_active
// @Description  Returns a paginated list of dogs whose is_active matches :value (true or false). Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        value   path   string  true   "Boolean value: true or false"
// @Param        limit   query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/is_active/{value} [get]
func (h *DogHandler) ListByIsActive(c *gin.Context) {
	value, err := strconv.ParseBool(c.Param("value"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "value"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByIsActive.Execute(c.Request.Context(), doguc.ListByIsActiveInput{
		IsActive: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByIncompatibility godoc
// @Summary      List dogs that have a specific incompatibility
// @Description  Returns a paginated list of dogs attached to the given incompatibility id. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        incompat_id  path   int  true   "ID of the incompatibility"
// @Param        limit        query  int  false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset       query  int  false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/incompatibility/{incompat_id} [get]
func (h *DogHandler) ListByIncompatibility(c *gin.Context) {
	incompatID, err := strconv.Atoi(c.Param("incompat_id"))
	if err != nil || incompatID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "incompatibility_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByIncompatibility.Execute(c.Request.Context(), doguc.ListByIncompatibilityInput{
		IncompatibilityID: incompatID, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByBreed godoc
// @Summary      List dogs filtered by breed
// @Description  Returns a paginated list of dogs whose breed matches :breed. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        breed  path   string  true   "Breed name (exact match)"
// @Param        limit  query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/breed/{breed} [get]
func (h *DogHandler) ListByBreed(c *gin.Context) {
	breed := c.Param("breed")
	if breed == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "breed"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByBreed.Execute(c.Request.Context(), doguc.ListByBreedInput{
		Breed: breed, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListBySex godoc
// @Summary      List dogs filtered by sex
// @Description  Returns a paginated list of dogs whose sex matches :sex. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        sex     path   string  true   "Sex: MALE or FEMALE"
// @Param        limit   query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/sex/{sex} [get]
func (h *DogHandler) ListBySex(c *gin.Context) {
	sex := domain.Sex(c.Param("sex"))
	if !sex.IsValid() {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "sex"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listBySex.Execute(c.Request.Context(), doguc.ListBySexInput{
		Sex: sex, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByNeutered godoc
// @Summary      List dogs filtered by neutered status
// @Description  Returns a paginated list of dogs whose neutered flag matches :value. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        value   path   string  true   "Boolean value: true or false"
// @Param        limit   query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/neutered/{value} [get]
func (h *DogHandler) ListByNeutered(c *gin.Context) {
	value, err := strconv.ParseBool(c.Param("value"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "value"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByNeutered.Execute(c.Request.Context(), doguc.ListByNeuteredInput{
		Neutered: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByHeat godoc
// @Summary      List dogs filtered by heat status
// @Description  Returns a paginated list of dogs whose heat flag matches :value. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        value   path   string  true   "Boolean value: true or false"
// @Param        limit   query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset  query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/heat/{value} [get]
func (h *DogHandler) ListByHeat(c *gin.Context) {
	value, err := strconv.ParseBool(c.Param("value"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "value"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByHeat.Execute(c.Request.Context(), doguc.ListByHeatInput{
		Heat: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListByAgeBracket godoc
// @Summary      List dogs filtered by age bracket
// @Description  Returns a paginated list of dogs whose age_bracket matches :bracket. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        bracket  path   string  true   "Age bracket: CHILDREN, TEENAGER, SEMI_ADULT, ADULT, UNKNOWN"
// @Param        limit    query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset   query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/age/{bracket} [get]
func (h *DogHandler) ListByAgeBracket(c *gin.Context) {
	bracket := domain.AgeBracket(c.Param("bracket"))
	if !bracket.IsValid() {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "age_bracket"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listByAgeBracket.Execute(c.Request.Context(), doguc.ListByAgeBracketInput{
		AgeBracket: bracket, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// ListBySizeBracket godoc
// @Summary      List dogs filtered by size bracket
// @Description  Returns a paginated list of dogs whose size_bracket matches :bracket. Limit defaults to 50, capped at 100. Offset defaults to 0.
// @Tags         dogs
// @Produce      json
// @Param        bracket  path   string  true   "Size bracket: MINI, MEDIUM, LARGE, UNKNOWN"
// @Param        limit    query  int     false  "Maximum number of dogs to return (default 50, max 100)"
// @Param        offset   query  int     false  "Number of dogs to skip for pagination (default 0)"
// @Success      200  {object}  listDogsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/size/{bracket} [get]
func (h *DogHandler) ListBySizeBracket(c *gin.Context) {
	bracket := domain.SizeBracket(c.Param("bracket"))
	if !bracket.IsValid() {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "size_bracket"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	out, err := h.listBySizeBracket.Execute(c.Request.Context(), doguc.ListBySizeBracketInput{
		SizeBracket: bracket, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, out.Dogs, limit, offset)
}

// Delete godoc
// @Summary      Delete a dog
// @Description  Removes a dog by id. Associated dog_incompatibilities and reservations rows are deleted automatically by the database (ON DELETE CASCADE). Returns 404 if the dog does not exist.
// @Tags         dogs
// @Produce      json
// @Param        id   path  int  true  "Dog ID"
// @Success      204
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/{id} [delete]
func (h *DogHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	if _, err := h.delete.Execute(c.Request.Context(), doguc.DeleteDogInput{ID: id}); err != nil {
		writeError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// writeList serializes a slice of dogs to the standard listDogsResponse.
// Shared by every list method to keep the response shape and the
// NormalizePagination call consistent.
func (h *DogHandler) writeList(c *gin.Context, dogs []*domain.Dog, limit, offset int) {
	dtos := make([]dogDTO, len(dogs))
	for i, d := range dogs {
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
	ID                int                  `json:"id" example:"1"`
	Name              string               `json:"name" example:"Luna"`
	Breed             string               `json:"breed" example:"Labrador"`
	AgeInMonths       int                  `json:"age_in_months" example:"24"`
	Sex               string               `json:"sex" example:"FEMALE"`
	Neutered          bool                 `json:"neutered" example:"false"`
	Heat              bool                 `json:"heat" example:"false"`
	WeightKg          float64              `json:"weight_kg" example:"22.5"`
	PhotoURL          string               `json:"photo_url" example:""`
	MedicalNotes      string               `json:"medical_notes" example:""`
	EducatorNotes     string               `json:"educator_notes" example:""`
	Passport          string               `json:"passport" example:"ES-12345"`
	UserID            int                  `json:"user_id" example:"1"`
	IsActive          bool                 `json:"is_active" example:"true"`
	Incompatibilities []incompatibilityDTO `json:"incompatibilities"`
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
	dto := dogDTO{
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
	// Always emit an array (never null) so clients can iterate unconditionally.
	incompats := d.Incompatibilities()
	dto.Incompatibilities = make([]incompatibilityDTO, 0, len(incompats))
	for i := range incompats {
		dto.Incompatibilities = append(dto.Incompatibilities, toIncompatibilityDTO(&incompats[i]))
	}
	return dto
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
