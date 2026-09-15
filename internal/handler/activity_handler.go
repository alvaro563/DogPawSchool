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

type ActivityBulkCompleter interface {
	Execute(ctx context.Context, input activityuc.BulkCompleteReservationsInput) (activityuc.BulkCompleteReservationsOutput, error)
}

type ActivitySlotCounter interface {
	CountHeldSlotsBatch(ctx context.Context, activityIDs []int) (map[int]int, error)
}

type ActivityHandler struct {
	register     ActivityRegisterer
	get          ActivityGetter
	modify       ActivityModifier
	list         ActivityLister
	upcoming     ActivityUpcomingLister
	close        ActivityCloser
	bulkComplete ActivityBulkCompleter
	slotCounter  ActivitySlotCounter
}

func NewActivityHandler(
	register ActivityRegisterer,
	get ActivityGetter,
	modify ActivityModifier,
	list ActivityLister,
	upcoming ActivityUpcomingLister,
	close ActivityCloser,
	bulkComplete ActivityBulkCompleter,
	slotCounter ActivitySlotCounter,
) *ActivityHandler {
	return &ActivityHandler{
		register:     register,
		get:          get,
		modify:       modify,
		list:         list,
		upcoming:     upcoming,
		close:        close,
		bulkComplete: bulkComplete,
		slotCounter:  slotCounter,
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
	var sizeTarget *domain.SizeBracket
	if request.SizeTarget != nil {
		st := domain.SizeBracket(*request.SizeTarget)
		sizeTarget = &st
	}
	in, err := activityuc.NewRegisterActivityInput(
		request.Name, request.Description, request.Location,
		domain.ActivityType(request.ActivityType),
		request.MaxCapacity, request.DurationInHours, request.Date,
		request.DogID, sizeTarget,
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
// @Summary      List activities (with filters)
// @Description  Returns a paginated list of activities in the system. Optionally filter by date range (from/to, RFC3339, inclusive on both ends) and by closed state (?closed=true|false). Most recent first. Limit defaults to 50 and is capped at 100. Offset defaults to 0.
// @Tags         activities
// @Produce      json
// @Param        limit   query  int    false  "Maximum number of activities to return (default 50, max 100)"
// @Param        offset  query  int    false  "Number of activities to skip for pagination (default 0)"
// @Param        from    query  string false  "Filter activities from this date (RFC3339, inclusive)"
// @Param        to      query  string false  "Filter activities up to this date (RFC3339, inclusive)"
// @Param        closed  query  bool   false  "true = only closed activities, false = only open activities. Omit for both."
// @Success      200  {object}  listActivitiesResponse  "List of activities"
// @Failure      400  {object}  errorResponse           "Invalid date range or closed value"
// @Failure      500  {object}  errorResponse           "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/activities [get]
func (h *ActivityHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	// from/to are pointers so the repo's nullableTime helper can
	// convert nil into SQL NULL and the `$N::timestamptz IS NULL`
	// predicate short-circuits the comparison. A zero time.Time
	// would serialise to `0001-01-01 00:00:00 UTC` and silently
	// exclude every row.
	var from, to *time.Time
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "from", Details: "invalid RFC3339 date"})
			return
		}
		from = &t
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "to", Details: "invalid RFC3339 date"})
			return
		}
		to = &t
	}

	// closed: nil = both, *true = closed only, *false = open only.
	var closedFilter *bool
	if raw := c.Query("closed"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "closed", Details: "must be true or false"})
			return
		}
		closedFilter = &v
	}

	// Read the viewer's session so the SQL can apply the visibility
	// filter at the database level (group + own individual for
	// non-admin, everything for admin).
	userID := CurrentUserID(c)
	isAdmin := IsAdmin(c)

	in, err := activityuc.NewListAllActivitiesInput(limit, offset, from, to, closedFilter, userID, isAdmin)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.list.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	heldSlots, err := h.computeHeldSlots(c.Request.Context(), output.Activities)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListActivitiesResponse(output.Activities, in, heldSlots))
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
	userID := CurrentUserID(c)
	isAdmin := IsAdmin(c)
	in, _ := activityuc.NewListUpcomingActivitiesInput(limit, offset, userID, isAdmin)
	output, err := h.upcoming.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	heldSlots, err := h.computeHeldSlots(c.Request.Context(), output.Activities)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListActivitiesResponse(output.Activities, in, heldSlots))
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
	userID := CurrentUserID(c)
	isAdmin := IsAdmin(c)
	in, err := activityuc.NewGetActivityInput(id, userID, isAdmin)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.get.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	heldSlots, err := h.computeHeldSlots(c.Request.Context(), []*domain.Activity{output.Activity})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(output.Activity, heldSlots[output.Activity.ID()]))
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
		Description:     request.Description,
		Location:        request.Location,
		MaxCapacity:     request.MaxCapacity,
		DurationInHours: request.DurationInHours,
		Date:            request.Date,
	}
	if request.ActivityType != nil {
		activityType := domain.ActivityType(*request.ActivityType)
		patch.ActivityType = &activityType
	}
	if request.SizeTarget != nil {
		st := domain.SizeBracket(*request.SizeTarget)
		patch.SizeTarget = &st
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
	heldSlots, err := h.computeHeldSlots(c.Request.Context(), []*domain.Activity{output.Activity})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(output.Activity, heldSlots[output.Activity.ID()]))
}

type registerActivityRequest struct {
	Name            string    `json:"name" example:"Paseo Río"`
	Description     string    `json:"description" example:"Paseo grupal por la ribera del río"`
	Location        string    `json:"location" example:"Parking Central"`
	ActivityType    string    `json:"activity_type" example:"ROUTE"`
	MaxCapacity     int       `json:"max_capacity" example:"8"`
	DurationInHours int       `json:"duration_in_hours" example:"2"`
	Date            time.Time `json:"date" example:"2026-08-01T10:00:00Z"`
	// DogID is the target dog for INDIVIDUAL_CLASS; required when
	// activity_type="INDIVIDUAL_CLASS", NULL for group classes and
	// extra events. Omitted / null in the JSON for non-individual.
	DogID *int `json:"dog_id,omitempty" example:"7"`
	// SizeTarget restricts the booking to a single size bracket
	// (MINI / MEDIUM / LARGE). Only valid for SOCIALIZATION_GROUP and
	// ROUTE. NULL / omitted means "all sizes welcome".
	SizeTarget *string `json:"size_target,omitempty" example:"MINI" enum:"MINI,MEDIUM,LARGE"`
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
	Description     *string    `json:"description,omitempty" example:"Descripción actualizada"`
	Location        *string    `json:"location,omitempty" example:"Río"`
	ActivityType    *string    `json:"activity_type,omitempty" example:"SOCIALIZATION_GROUP"`
	MaxCapacity     *int       `json:"max_capacity,omitempty" example:"12"`
	DurationInHours *int       `json:"duration_in_hours,omitempty" example:"3"`
	Date            *time.Time `json:"date,omitempty" example:"2026-09-01T10:00:00Z"`
	// SizeTarget uses a triple-state via *string:
	//   nil       = no change
	//   pointer to ""  = clear the target (back to all sizes)
	//   pointer to "MINI"/"MEDIUM"/"LARGE" = set the target
	// Only valid for SOCIALIZATION_GROUP / ROUTE.
	SizeTarget *string `json:"size_target,omitempty" example:"MEDIUM" enum:"MINI,MEDIUM,LARGE,"`
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

type bulkCompleteReservationsResponse struct {
	ID        int  `json:"id"        example:"42"`
	Completed int  `json:"completed" example:"5"`
	Closed    bool `json:"closed"    example:"true"`
}

// BulkCompleteReservations godoc
// @Summary      Complete activity and mark all reservations COMPLETED
// @Description  Bulk-completes every CONFIRMED reservation of the
// @Description  activity AND closes the activity in a single
// @Description  transaction. The activity must have finished
// @Description  (date + duration < now). Rejects with 409 if any
// @Description  PENDING_TO_CONFIRM reservation is present (admin
// @Description  must confirm/reject individually first). Idempotent
// @Description  on already-closed activities (returns Closed=true,
// @Description  Completed=0).
// @Tags         activities
// @Produce      json
// @Param        id   path int true "Activity ID"
// @Success      200 {object} bulkCompleteReservationsResponse
// @Failure      400 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      409 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/complete-all [post]
func (h *ActivityHandler) BulkCompleteReservations(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "id"})
		return
	}
	in, err := activityuc.NewBulkCompleteReservationsInput(id, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	out, err := h.bulkComplete.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, bulkCompleteReservationsResponse{
		ID:        out.ActivityID,
		Completed: out.CompletedCount,
		Closed:    out.Closed,
	})
}

type activityResponse struct {
	ID              int       `json:"id" example:"42"`
	Name            string    `json:"name" example:"Paseo Río"`
	Description     string    `json:"description" example:"Paseo grupal por la ribera del río"`
	ActivityType    string    `json:"activity_type" example:"ROUTE"`
	MaxCapacity     int       `json:"max_capacity" example:"8"`
	AvailableSpots  int       `json:"available_spots" example:"5"`
	Location        string    `json:"location" example:"Parking Central"`
	DurationInHours int       `json:"duration_in_hours" example:"2"`
	Date            time.Time `json:"date" example:"2026-08-01T10:00:00Z"`
	Closed          bool      `json:"closed" example:"false"`
	// DogID is the target dog for INDIVIDUAL_CLASS; omitted from the
	// response when the activity is group / extra (NULL in the DB).
	DogID *int `json:"dog_id,omitempty" example:"7"`
	// SizeTarget is the optional size restriction for
	// SOCIALIZATION_GROUP / ROUTE. NULL / omitted = all sizes.
	SizeTarget *string `json:"size_target,omitempty" example:"MINI" enum:"MINI,MEDIUM,LARGE"`
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
func toListActivitiesResponse(activities []*domain.Activity, in activityPagination, heldSlots map[int]int) listActivitiesResponse {
	dtos := make([]activityDTO, len(activities))
	for i, activity := range activities {
		dtos[i] = toActivityDTO(activity, heldSlots[activity.ID()])
	}
	return listActivitiesResponse{
		Activities: dtos,
		Limit:      in.Limit(),
		Offset:     in.Offset(),
		Count:      len(dtos),
	}
}

// toActivityDTO converts a domain.Activity into the HTTP wire format.
func toActivityDTO(activity *domain.Activity, heldSlots int) activityDTO {
	var sizePtr *string
	if st := activity.SizeTarget(); st != nil {
		v := string(*st)
		sizePtr = &v
	}
	return activityDTO{
		ID:              activity.ID(),
		Name:            activity.Name(),
		Description:     activity.Description(),
		ActivityType:    string(activity.Type()),
		MaxCapacity:     activity.MaxCapacity(),
		AvailableSpots:  activity.MaxCapacity() - heldSlots,
		Location:        activity.Location(),
		DurationInHours: activity.DurationInHours(),
		Date:            activity.Date(),
		Closed:          activity.IsClosed(),
		DogID:           activity.DogID(),
		SizeTarget:      sizePtr,
	}
}

// computeHeldSlots returns a map of activityID → slot-holding
// reservation count. Returns an empty map for degrated mode (when
// the slot counter is not available).
func (h *ActivityHandler) computeHeldSlots(ctx context.Context, activities []*domain.Activity) (map[int]int, error) {
	if h.slotCounter == nil || len(activities) == 0 {
		return map[int]int{}, nil
	}
	ids := make([]int, len(activities))
	for i, a := range activities {
		ids[i] = a.ID()
	}
	return h.slotCounter.CountHeldSlotsBatch(ctx, ids)
}
