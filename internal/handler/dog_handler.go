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
)

type DogRegistrar interface {
	Execute(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error)
}

type DogListerByOwner interface {
	Execute(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

type DogHandler struct {
	register DogRegistrar
	list     DogListerByOwner
}

func NewDogHandler(register DogRegistrar, list DogListerByOwner) *DogHandler {
	return &DogHandler{register: register, list: list}
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

func writeError(c *gin.Context, err error) {
	var verr *doguc.ValidationError
	if errors.As(err, &verr) {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: verr.Field,
		})
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
