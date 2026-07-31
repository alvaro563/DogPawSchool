package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	activityuc "dogpaw/internal/usecase/activity"
)

type ActivityRegisterer interface {
	Execute(ctx context.Context, input activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error)
}

type ActivityGetter interface {
	Execute(ctx context.Context, input activityuc.GetActivityInput) (activityuc.GetActivityOutput, error)
}

type ActivityModifier interface {
	Execute(ctx context.Context, input activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error)
}

type ActivityLister interface {
	Execute(ctx context.Context, input activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error)
}

type ActivityUpcomingLister interface {
	Execute(ctx context.Context, input activityuc.ListUpcomingActivitiesInput) (activityuc.ListUpcomingActivitiesOutput, error)
}

type ActivityCloser interface {
	Execute(ctx context.Context, input activityuc.CloseActivityInput) (activityuc.CloseActivityOutput, error)
}

type ActivityHandler struct {
	register ActivityRegisterer
	get      ActivityGetter
	modify   ActivityModifier
	list     ActivityLister
	upcoming ActivityUpcomingLister
	close    ActivityCloser
}

func NewActivityHandler(
	register ActivityRegisterer,
	get ActivityGetter,
	modify ActivityModifier,
	list ActivityLister,
	upcoming ActivityUpcomingLister,
	close ActivityCloser,
) *ActivityHandler {
	return &ActivityHandler{
		register: register,
		get:      get,
		modify:   modify,
		list:     list,
		upcoming: upcoming,
		close:    close,
	}
}

// Register godoc
// @Summary      Register a new activity
// @Description  Creates a new school activity (class, route, individual session, or extra). Returns the new resource URL in the Location header.
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        activity  body      registerActivityRequest  true  "Activity to create"
// @Success      201       {object}  registerActivityResponse  "Activity created"
// @Failure      400       {object}  errorResponse             "Invalid request body or missing fields"
// @Failure      500       {object}  errorResponse             "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities [post]
func (h *ActivityHandler) Register(c *gin.Context) {
	var request registerActivityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}
	in, err := activityuc.NewRegisterActivityInput(
		request.Name, request.Location,
		domain.ActivityType(request.ActivityType),
		request.MaxCapacity, request.DurationInHours, request.Date,
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
	c.Header("Location", "/api/v1/activities/"+strconv.Itoa(output.ID))
	c.JSON(http.StatusCreated, registerActivityResponse{ID: output.ID})
}

// List godoc
// @Summary      List all activities
// @Description  Returns a paginated list of all activities in the system, most recent first. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         activities
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of activities to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of activities to skip for pagination (default 0)"
// @Success      200  {object}  listActivitiesResponse  "List of activities"
// @Failure      500  {object}  errorResponse           "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities [get]
func (h *ActivityHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, _ := activityuc.NewListAllActivitiesInput(limit, offset)
	output, err := h.list.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListActivitiesResponse(output.Activities, in))
}

// ListUpcoming godoc
// @Summary      List upcoming activities
// @Description  Returns a paginated list of activities scheduled at or after the current time, soonest first.
// @Tags         activities
// @Produce      json
// @Param        limit   query  int  false  "Maximum number of activities to return (default 50, max 100)"
// @Param        offset  query  int  false  "Number of activities to skip for pagination (default 0)"
// @Success      200  {object}  listActivitiesResponse  "List of upcoming activities"
// @Failure      500  {object}  errorResponse           "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities/upcoming [get]
func (h *ActivityHandler) ListUpcoming(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, _ := activityuc.NewListUpcomingActivitiesInput(limit, offset)
	output, err := h.upcoming.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListActivitiesResponse(output.Activities, in))
}

// GetByID godoc
// @Summary      Get activity by ID
// @Description  Returns a single activity by its id.
// @Tags         activities
// @Produce      json
// @Param        id   path      int                 true  "Activity ID"
// @Success      200  {object}  activityResponse   "Activity found"
// @Failure      400  {object}  errorResponse       "Invalid id"
// @Failure      404  {object}  errorResponse       "Activity not found"
// @Failure      500  {object}  errorResponse       "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities/{id} [get]
func (h *ActivityHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	in, err := activityuc.NewGetActivityInput(id)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.get.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(output.Activity))
}

// Modify godoc
// @Summary      Patch an activity
// @Description  Partially updates an activity. Only the supplied fields are mutated; an empty body is a no-op.
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id        path      int                       true  "Activity ID"
// @Param        activity  body      modifyActivityRequest    true  "Fields to patch"
// @Success      200       {object}  activityResponse         "Updated activity"
// @Failure      400       {object}  errorResponse            "Invalid id, body, or validation error"
// @Failure      404       {object}  errorResponse            "Activity not found"
// @Failure      500       {object}  errorResponse            "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities/{id} [patch]
func (h *ActivityHandler) Modify(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request modifyActivityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	patch := domain.ActivityPatch{
		Name:            request.Name,
		Location:        request.Location,
		MaxCapacity:     request.MaxCapacity,
		DurationInHours: request.DurationInHours,
		Date:            request.Date,
	}
	if request.ActivityType != nil {
		activityType := domain.ActivityType(*request.ActivityType)
		patch.ActivityType = &activityType
	}
	in, err := activityuc.NewModifyActivityInput(id, patch)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.modify.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(output.Activity))
}

type registerActivityRequest struct {
	Name            string    `json:"name" example:"Paseo Río"`
	Location        string    `json:"location" example:"Parking Central"`
	ActivityType    string    `json:"activity_type" example:"ROUTE"`
	MaxCapacity     int       `json:"max_capacity" example:"8"`
	DurationInHours int       `json:"duration_in_hours" example:"2"`
	Date            time.Time `json:"date" example:"2026-08-01T10:00:00Z"`
}

type registerActivityResponse struct {
	ID int `json:"id" example:"42"`
}

type listActivitiesResponse struct {
	Activities []activityDTO `json:"activities"`
	Limit      int           `json:"limit" example:"50"`
	Offset     int           `json:"offset" example:"0"`
	Count      int           `json:"count" example:"1"`
}

type modifyActivityRequest struct {
	Name            *string    `json:"name,omitempty" example:"Paseo Largo"`
	Location        *string    `json:"location,omitempty" example:"Río"`
	ActivityType    *string    `json:"activity_type,omitempty" example:"SOCIALIZATION_GROUP"`
	MaxCapacity     *int       `json:"max_capacity,omitempty" example:"12"`
	DurationInHours *int       `json:"duration_in_hours,omitempty" example:"3"`
	Date            *time.Time `json:"date,omitempty" example:"2026-09-01T10:00:00Z"`
}

type closeActivityRequest struct {
	NoShowReservationIDs []int `json:"no_show_reservation_ids" example:"[1,2]"`
}

type closeActivityResponse struct {
	ID     int  `json:"id"     example:"42"`
	Closed bool `json:"closed" example:"true"`
}

// Close godoc
// @Summary      Close an activity
// @Description  Closes an activity after it has finished. Batch-
// @Description  processes every CONFIRMED reservation: those whose
// @Description  id appears in no_show_reservation_ids are marked as
// @Description  NO_SHOW, the rest are marked as COMPLETED. Then
// @Description  the activity is marked as closed and persisted.
// @Description  The entire flow runs in a single transaction.
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id        path      int                   true   "Activity ID"
// @Param        activity  body      closeActivityRequest  false  "Optional no-show reservation IDs"
// @Success      200       {object}  closeActivityResponse "Activity closed"
// @Failure      400       {object}  errorResponse         "Invalid id, activity not finished, or invalid reservation IDs"
// @Failure      404       {object}  errorResponse         "Activity not found"
// @Failure      409       {object}  errorResponse         "Activity already closed or reservation not confirmed"
// @Failure      500       {object}  errorResponse         "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/close [post]
func (h *ActivityHandler) Close(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	var request closeActivityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		// Empty body is valid (no no-show IDs).
		request = closeActivityRequest{}
	}
	in, err := activityuc.NewCloseActivityInput(id, request.NoShowReservationIDs, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.close.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, closeActivityResponse{
		ID:     output.Activity.ID(),
		Closed: output.Activity.IsClosed(),
	})
}

type activityResponse struct {
	ID              int       `json:"id" example:"42"`
	Name            string    `json:"name" example:"Paseo Río"`
	ActivityType    string    `json:"activity_type" example:"ROUTE"`
	MaxCapacity     int       `json:"max_capacity" example:"8"`
	Location        string    `json:"location" example:"Parking Central"`
	DurationInHours int       `json:"duration_in_hours" example:"2"`
	Date            time.Time `json:"date" example:"2026-08-01T10:00:00Z"`
	Closed          bool      `json:"closed" example:"false"`
}

// activityDTO is the wire format of an activity. It mirrors activityResponse
// but is the canonical type used in list responses and embeds.
type activityDTO = activityResponse

// activityPagination is the contract every activity list input
// satisfies: it exposes the (already normalized) limit and offset.
type activityPagination interface {
	Limit() int
	Offset() int
}

// toListActivitiesResponse builds the list response envelope with
// the normalized limit/offset from the validated input.
func toListActivitiesResponse(activities []*domain.Activity, in activityPagination) listActivitiesResponse {
	dtos := make([]activityDTO, len(activities))
	for i, activity := range activities {
		dtos[i] = toActivityDTO(activity)
	}
	return listActivitiesResponse{
		Activities: dtos,
		Limit:      in.Limit(),
		Offset:     in.Offset(),
		Count:      len(dtos),
	}
}

// toActivityDTO converts a domain.Activity into the HTTP wire format.
func toActivityDTO(activity *domain.Activity) activityDTO {
	return activityDTO{
		ID:              activity.ID(),
		Name:            activity.Name(),
		ActivityType:    string(activity.Type()),
		MaxCapacity:     activity.MaxCapacity(),
		Location:        activity.Location(),
		DurationInHours: activity.DurationInHours(),
		Date:            activity.Date(),
		Closed:          activity.IsClosed(),
	}
}
