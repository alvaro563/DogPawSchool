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
	activityuc "dogpaw/internal/usecase/activity"
	doguc "dogpaw/internal/usecase/dog"
	incompatuc "dogpaw/internal/usecase/incompatibility"
	passuc "dogpaw/internal/usecase/pass"
	reservationuc "dogpaw/internal/usecase/reservation"
)

type DogRegistrar interface {
	Execute(ctx context.Context, input doguc.RegisterDogInput) (doguc.RegisterDogOutput, error)
}

type DogLister interface {
	Execute(ctx context.Context, input doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error)
}

type DogListerByOwner interface {
	Execute(ctx context.Context, input doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

type DogActiveLister interface {
	Execute(ctx context.Context, input doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error)
}

type DogByIsActiveLister interface {
	Execute(ctx context.Context, input doguc.ListByIsActiveInput) (doguc.ListByIsActiveOutput, error)
}

type DogByIncompatibilityLister interface {
	Execute(ctx context.Context, input doguc.ListByIncompatibilityInput) (doguc.ListByIncompatibilityOutput, error)
}

type DogByBreedLister interface {
	Execute(ctx context.Context, input doguc.ListByBreedInput) (doguc.ListByBreedOutput, error)
}

type DogBySexLister interface {
	Execute(ctx context.Context, input doguc.ListBySexInput) (doguc.ListBySexOutput, error)
}

type DogByNeuteredLister interface {
	Execute(ctx context.Context, input doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error)
}

type DogByHeatLister interface {
	Execute(ctx context.Context, input doguc.ListByHeatInput) (doguc.ListByHeatOutput, error)
}

type DogByAgeBracketLister interface {
	Execute(ctx context.Context, input doguc.ListByAgeBracketInput) (doguc.ListByAgeBracketOutput, error)
}

type DogBySizeBracketLister interface {
	Execute(ctx context.Context, input doguc.ListBySizeBracketInput) (doguc.ListBySizeBracketOutput, error)
}

type DogModifier interface {
	Execute(ctx context.Context, input doguc.ModifyDogInput) (doguc.ModifyDogOutput, error)
}

type DogIncompatibilityAdder interface {
	Execute(ctx context.Context, input doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error)
}

type DogIncompatibilityRemover interface {
	Execute(ctx context.Context, input doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error)
}

type DogDeleter interface {
	Execute(ctx context.Context, input doguc.DeleteDogInput) (doguc.DeleteDogOutput, error)
}

type DogNeuteredSetter interface {
	Execute(ctx context.Context, input doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error)
}

type DogHeatSetter interface {
	Execute(ctx context.Context, input doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error)
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
	setNeutered           DogNeuteredSetter
	setHeat               DogHeatSetter
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
	setNeutered DogNeuteredSetter,
	setHeat DogHeatSetter,
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
		setNeutered:           setNeutered,
		setHeat:               setHeat,
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
	var request registerDogRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	output, err := h.register.Execute(c.Request.Context(), doguc.RegisterDogInput{
		Name:        request.Name,
		Breed:       request.Breed,
		AgeInMonths: request.AgeInMonths,
		Sex:         domain.Sex(request.Sex),
		WeightKg:    request.WeightKg,
		Passport:    request.Passport,
		UserID:      request.UserID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Location", fmt.Sprintf("/api/v1/dogs/%d", output.ID))
	c.JSON(http.StatusCreated, registerDogResponse{ID: output.ID})
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

	output, err := h.list.Execute(c.Request.Context(), doguc.ListAllDogsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	dtos := make([]dogDTO, len(output.Dogs))
	for i, dog := range output.Dogs {
		dtos[i] = toDogDTO(dog)
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

	output, err := h.listByOwner.Execute(c.Request.Context(), doguc.ListByOwnerInput{
		OwnerID: ownerID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	dtos := make([]dogDTO, len(output.Dogs))
	for i, dog := range output.Dogs {
		dtos[i] = toDogDTO(dog)
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

	var request modifyDogRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	patch := domain.DogPatch{
		Name:          request.Name,
		Breed:         request.Breed,
		AgeInMonths:   request.AgeInMonths,
		Passport:      request.Passport,
		WeightKg:      request.WeightKg,
		Neutered:      request.Neutered,
		Heat:          request.Heat,
		PhotoURL:      request.PhotoURL,
		MedicalNotes:  request.MedicalNotes,
		EducatorNotes: request.EducatorNotes,
		IsActive:      request.IsActive,
	}
	if request.Sex != nil {
		sexValue := domain.Sex(*request.Sex)
		patch.Sex = &sexValue
	}

	output, err := h.modify.Execute(c.Request.Context(), doguc.ModifyDogInput{
		ID:    id,
		Patch: patch,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, modifyDogResponse{ID: output.ID})
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

	var request addIncompatibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	output, err := h.addIncompat.Execute(c.Request.Context(), doguc.AddDogIncompatibilityInput{
		DogID:             dogID,
		IncompatibilityID: request.IncompatibilityID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	status := http.StatusOK
	if output.Added {
		status = http.StatusCreated
	}
	c.JSON(status, addIncompatibilityResponse{
		DogID:             output.ID,
		Added:             output.Added,
		Incompatibilities: toIncompatibilityDTOs(output.Incompatibilities),
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

	output, err := h.removeIncompat.Execute(c.Request.Context(), doguc.RemoveDogIncompatibilityInput{
		DogID:             dogID,
		IncompatibilityID: incompatID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, removeIncompatibilityResponse{
		DogID:             output.ID,
		Removed:           output.Removed,
		Incompatibilities: toIncompatibilityDTOs(output.Incompatibilities),
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

	output, err := h.listActive.Execute(c.Request.Context(), doguc.ListActiveDogsInput{
		Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByIsActive.Execute(c.Request.Context(), doguc.ListByIsActiveInput{
		IsActive: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByIncompatibility.Execute(c.Request.Context(), doguc.ListByIncompatibilityInput{
		IncompatibilityID: incompatID, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByBreed.Execute(c.Request.Context(), doguc.ListByBreedInput{
		Breed: breed, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listBySex.Execute(c.Request.Context(), doguc.ListBySexInput{
		Sex: sex, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByNeutered.Execute(c.Request.Context(), doguc.ListByNeuteredInput{
		Neutered: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByHeat.Execute(c.Request.Context(), doguc.ListByHeatInput{
		Heat: value, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listByAgeBracket.Execute(c.Request.Context(), doguc.ListByAgeBracketInput{
		AgeBracket: bracket, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

	output, err := h.listBySizeBracket.Execute(c.Request.Context(), doguc.ListBySizeBracketInput{
		SizeBracket: bracket, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeList(c, output.Dogs, limit, offset)
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

// SetNeutered godoc
// @Summary      Set the neutered flag of a dog
// @Description  Fast-path endpoint to flip the neutered flag. Body is {"neutered": true|false}. Returns the new state plus the dog's sex. Returns 404 if the dog does not exist.
// @Tags         dogs
// @Accept       json
// @Produce      json
// @Param        id        path   int                true  "Dog ID"
// @Param        body      body   setNeuteredRequest  true  "New value for neutered"
// @Success      200  {object}  setNeuteredResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/{id}/neutered [patch]
func (h *DogHandler) SetNeutered(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request setNeuteredRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	output, err := h.setNeutered.Execute(c.Request.Context(), doguc.SetDogNeuteredInput{
		ID:       id,
		Neutered: request.Neutered,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, setNeuteredResponse{
		ID:       output.ID,
		Neutered: output.Neutered,
		Sex:      string(output.Sex),
	})
}

// SetHeat godoc
// @Summary      Set the heat flag of a dog
// @Description  Fast-path endpoint to flip the heat flag. Body is {"heat": true|false}. Returns 400 with error "invalid_heat_for_sex" if heat=true is attempted on a non-female dog.
// @Tags         dogs
// @Accept       json
// @Produce      json
// @Param        id        path   int            true  "Dog ID"
// @Param        body      body   setHeatRequest  true  "New value for heat"
// @Success      200  {object}  setHeatResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/dogs/{id}/heat [patch]
func (h *DogHandler) SetHeat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request setHeatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	output, err := h.setHeat.Execute(c.Request.Context(), doguc.SetDogHeatInput{
		ID:   id,
		Heat: request.Heat,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, setHeatResponse{
		ID:   output.ID,
		Heat: output.Heat,
		Sex:  string(output.Sex),
	})
}

// writeList serializes a slice of dogs to the standard listDogsResponse.
// Shared by every list method to keep the response shape and the
// NormalizePagination call consistent.
func (h *DogHandler) writeList(c *gin.Context, dogs []*domain.Dog, limit, offset int) {
	dtos := make([]dogDTO, len(dogs))
	for i, dog := range dogs {
		dtos[i] = toDogDTO(dog)
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
	var dogValidationErr *doguc.ValidationError
	var incompValidationErr *incompatuc.ValidationError
	var activityValidationErr *activityuc.ValidationError
	var passValidationErr *passuc.ValidationError
	var reservationValidationErr *reservationuc.ValidationError
	if errors.As(err, &dogValidationErr) || errors.As(err, &incompValidationErr) || errors.As(err, &activityValidationErr) || errors.As(err, &passValidationErr) || errors.As(err, &reservationValidationErr) {
		var field string
		switch {
		case dogValidationErr != nil:
			field = dogValidationErr.Field
		case incompValidationErr != nil:
			field = incompValidationErr.Field
		case activityValidationErr != nil:
			field = activityValidationErr.Field
		case reservationValidationErr != nil:
			field = reservationValidationErr.Field
		default:
			field = passValidationErr.Field
		}
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: field,
		})
		return
	}
	if errors.Is(err, doguc.ErrNotFound) || errors.Is(err, incompatuc.ErrNotFound) || errors.Is(err, activityuc.ErrNotFound) || errors.Is(err, passuc.ErrNotFound) || errors.Is(err, reservationuc.ErrNotFound) || errors.Is(err, reservationuc.ErrInvalidReservation) || errors.Is(err, reservationuc.ErrReservationNotOwned) || errors.Is(err, postgres.ErrNotFound) || errors.Is(err, postgres.ErrActivityNotFound) || errors.Is(err, postgres.ErrPassNotFound) || errors.Is(err, postgres.ErrReservationNotFound) {
		c.JSON(http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if errors.Is(err, postgres.ErrInvalidUser) || errors.Is(err, postgres.ErrInvalidPassUser) || errors.Is(err, passuc.ErrInvalidUserID) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_user_id"})
		return
	}
	if errors.Is(err, reservationuc.ErrInvalidActivity) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_activity_id"})
		return
	}
	if errors.Is(err, reservationuc.ErrActivityInPast) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "activity_in_past"})
		return
	}
	if errors.Is(err, reservationuc.ErrActivityFull) || errors.Is(err, reservationuc.ErrDuplicateReservationForDog) || errors.Is(err, reservationuc.ErrAlreadyCancelled) {
		c.JSON(http.StatusConflict, errorResponse{Error: mapReservationConflictError(err)})
		return
	}
	if errors.Is(err, reservationuc.ErrInvalidDog) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_dog_id"})
		return
	}
	if errors.Is(err, reservationuc.ErrInvalidPass) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_pass_id"})
		return
	}
	if errors.Is(err, reservationuc.ErrPassExhausted) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "pass_exhausted"})
		return
	}
	if errors.Is(err, reservationuc.ErrPassExpired) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "pass_expired"})
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
	if errors.Is(err, doguc.ErrInvalidHeatForSex) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_heat_for_sex"})
		return
	}
	slog.Error("internal error",
		"err", err.Error(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal"})
}

// mapReservationConflictError returns the wire-level "error" key
// for reservation conflicts. Three sentinels map to 409: capacity
// exceeded, duplicate booking for the same dog + activity, and
// already-cancelled reservations. The caller has already checked
// the sentinel so we know one of them matches.
func mapReservationConflictError(err error) string {
	switch {
	case errors.Is(err, reservationuc.ErrActivityFull):
		return "activity_full"
	case errors.Is(err, reservationuc.ErrAlreadyCancelled):
		return "already_cancelled"
	default:
		return "duplicate_reservation"
	}
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

type setNeuteredRequest struct {
	Neutered bool `json:"neutered" example:"true"`
}

type setNeuteredResponse struct {
	ID       int    `json:"id" example:"42"`
	Neutered bool   `json:"neutered" example:"true"`
	Sex      string `json:"sex" example:"FEMALE"`
}

type setHeatRequest struct {
	Heat bool `json:"heat" example:"false"`
}

type setHeatResponse struct {
	ID   int    `json:"id" example:"42"`
	Heat bool   `json:"heat" example:"false"`
	Sex  string `json:"sex" example:"FEMALE"`
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

// toDogDTO converts a domain.Dog into the HTTP wire-format dogDTO. The
// incompatibilities slice is always emitted (never null) so clients can
// iterate unconditionally.
func toDogDTO(dog *domain.Dog) dogDTO {
	dto := dogDTO{
		ID:            dog.ID(),
		Name:          dog.Name(),
		Breed:         dog.Breed(),
		AgeInMonths:   dog.AgeInMonths(),
		Sex:           string(dog.Sex()),
		Neutered:      dog.Neutered(),
		Heat:          dog.Heat(),
		WeightKg:      dog.WeightKg(),
		PhotoURL:      dog.PhotoURL(),
		MedicalNotes:  dog.MedicalNotes(),
		EducatorNotes: dog.EducatorNotes(),
		Passport:      dog.Passport(),
		UserID:        dog.UserID(),
		IsActive:      dog.IsActive(),
	}
	incompatibilities := dog.Incompatibilities()
	dto.Incompatibilities = make([]incompatibilityDTO, 0, len(incompatibilities))
	for index := range incompatibilities {
		dto.Incompatibilities = append(dto.Incompatibilities, toIncompatibilityDTO(&incompatibilities[index]))
	}
	return dto
}

func toIncompatibilityDTO(incompat *domain.Incompatibility) incompatibilityDTO {
	return incompatibilityDTO{
		ID:    incompat.ID(),
		Name:  incompat.Name(),
		Level: string(incompat.Type()),
	}
}

func toIncompatibilityDTOs(incompatibilities []domain.Incompatibility) []incompatibilityDTO {
	out := make([]incompatibilityDTO, len(incompatibilities))
	for index, incompat := range incompatibilities {
		out[index] = toIncompatibilityDTO(&incompat)
	}
	return out
}
