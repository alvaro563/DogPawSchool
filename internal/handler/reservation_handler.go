package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"dogpaw/internal/domain"
	reservationuc "dogpaw/internal/usecase/reservation"
)

type ReservationRegisterer interface {
	Execute(ctx context.Context, input reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error)
}

type ReservationCanceler interface {
	Execute(ctx context.Context, input reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error)
}

type ReservationGetter interface {
	Execute(ctx context.Context, input reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error)
}

type ReservationListerByUser interface {
	Execute(ctx context.Context, input reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error)
}

type ReservationListerUpcomingByUser interface {
	Execute(ctx context.Context, input reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error)
}

type ReservationListerByDog interface {
	Execute(ctx context.Context, input reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error)
}

type ReservationListerByPass interface {
	Execute(ctx context.Context, input reservationuc.ListByPassReservationsInput) (reservationuc.ListByPassReservationsOutput, error)
}

type ReservationListerByActivity interface {
	Execute(ctx context.Context, input reservationuc.ListByActivityReservationsInput) (reservationuc.ListByActivityReservationsOutput, error)
}

type ReservationNoShower interface {
	Execute(ctx context.Context, input reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error)
}

type ReservationCompleter interface {
	Execute(ctx context.Context, input reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error)
}

// ReservationHandler owns the HTTP entry points for reservation
// use cases. It exposes 10 use cases (Register, Cancel, Get,
// ListByUser, ListUpcomingByUser, ListByDog, ListByPass,
// ListByActivity, MarkNoShow, CompleteReservation).
type ReservationHandler struct {
	register       ReservationRegisterer
	cancel         ReservationCanceler
	get            ReservationGetter
	listByUser     ReservationListerByUser
	listUpcoming   ReservationListerUpcomingByUser
	listByDog      ReservationListerByDog
	listByPass     ReservationListerByPass
	listByActivity ReservationListerByActivity
	noShow         ReservationNoShower
	complete       ReservationCompleter
}

func NewReservationHandler(
	register ReservationRegisterer,
	cancel ReservationCanceler,
	get ReservationGetter,
	listByUser ReservationListerByUser,
	listUpcoming ReservationListerUpcomingByUser,
	listByDog ReservationListerByDog,
	listByPass ReservationListerByPass,
	listByActivity ReservationListerByActivity,
	noShow ReservationNoShower,
	complete ReservationCompleter,
) *ReservationHandler {
	return &ReservationHandler{
		register:       register,
		cancel:         cancel,
		get:            get,
		listByUser:     listByUser,
		listUpcoming:   listUpcoming,
		listByDog:      listByDog,
		listByPass:     listByPass,
		listByActivity: listByActivity,
		noShow:         noShow,
		complete:       complete,
	}
}

// Register godoc
// @Summary      Book a reservation for a dog
// @Description  Books a dog into an activity, paid from one of the
// @Description  owner's passes. Atomically: validates the activity
// @Description  is in the future and not full, the dog is owned by
// @Description  the user in the path, and the pass is owned by the
// @Description  user and has at least one session. Consumes one pass
// @Description  session and creates the reservation in StatusConfirmed.
// @Tags         reservations
// @Accept       json
// @Produce      json
// @Param        user_id      path      int                          true   "Owner user ID"
// @Param        reservation  body      registerReservationRequest   true   "Reservation to create"
// @Success      201          {object}  registerReservationResponse  "Reservation created"
// @Failure      400          {object}  errorResponse                "Invalid user_id, request body, or missing fields"
// @Failure      404          {object}  errorResponse                "Not found"
// @Failure      409          {object}  errorResponse                "Activity full or duplicate reservation"
// @Failure      500          {object}  errorResponse                "Internal server error"
// @Router       /api/v1/users/{user_id}/reservations [post]
func (h *ReservationHandler) Register(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "validation",
			Field: "user_id",
		})
		return
	}
	var request registerReservationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Details: err.Error(),
		})
		return
	}

	in, err := reservationuc.NewRegisterReservationInput(userID, request.ActivityID, request.DogID, request.PassID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.register.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Location", "/api/v1/reservations/"+strconv.Itoa(output.ID))
	c.JSON(http.StatusCreated, registerReservationResponse{ID: output.ID})
}

// registerReservationRequest is the wire format for creating a
// reservation. The owner user_id is taken from the URL path; only
// the cross-aggregate ids (activity, dog, pass) are in the body.
type registerReservationRequest struct {
	ActivityID int `json:"activity_id" example:"42"`
	DogID      int `json:"dog_id"      example:"7"`
	PassID     int `json:"pass_id"     example:"3"`
}

type registerReservationResponse struct {
	ID int `json:"id" example:"99"`
}

// Cancel godoc
// @Summary      Cancel a reservation
// @Description  Cancels a CONFIRMED reservation owned by the user in
// @Description  the path. The activity must still be in the future
// @Description  and the reservation must be in StatusConfirmed. If
// @Description  the cancel happens more than 2h before the activity
// @Description  date, the reservation transitions to
// @Description  StatusCancelledInTime AND the pass session is
// @Description  refunded (remaining_sessions + 1, audit movement +1
// @Description  appended). If the cancel happens within the late
// @Description  window, the reservation transitions to
// @Description  StatusCancelledLate and no refund is applied (an
// @Description  admin can later call Forgive to refund it).
// @Tags         reservations
// @Produce      json
// @Param        user_id        path      int                          true   "Owner user ID"
// @Param        id             path      int                          true   "Reservation ID"
// @Success      200            {object}  cancelReservationResponse    "Reservation cancelled"
// @Failure      400            {object}  errorResponse                "Invalid user_id or reservation_id"
// @Failure      404            {object}  errorResponse                "Reservation not found"
// @Failure      409            {object}  errorResponse                "Already cancelled / not in a cancellable state"
// @Failure      500            {object}  errorResponse                "Internal server error"
// @Router       /api/v1/users/{user_id}/reservations/{id}/cancel [post]
func (h *ReservationHandler) Cancel(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewCancelReservationInput(userID, reservationID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.cancel.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, cancelReservationResponse{
		ID:     output.Reservation.ID(),
		Status: string(output.Reservation.Status()),
	})
}

// cancelReservationResponse is the wire format for a successful
// cancel. The status field exposes the new state (CANCELLED_IN_TIME
// or CANCELLED_LATE) so the client can decide whether to surface
// a "session refunded" message to the user.
type cancelReservationResponse struct {
	ID     int    `json:"id"     example:"99"`
	Status string `json:"status" example:"CANCELLED_IN_TIME"`
}

// markNoShowResponse is the wire format for a successful
// no-show mark. Same shape as cancelReservationResponse (id +
// status); the status field will be NO_SHOW.
type markNoShowResponse struct {
	ID     int    `json:"id"     example:"99"`
	Status string `json:"status" example:"NO_SHOW"`
}

// MarkNoShow godoc
// @Summary      Mark a reservation as no-show
// @Description  Transitions a CONFIRMED reservation to NO_SHOW.
// @Description  The activity must have already started
// @Description  (date < now). Does not refund the pass session
// @Description  because the slot is past. Owner-only: the
// @Description  user_id in the path must own the dog. Returns 404
// @Description  if the reservation does not exist OR belongs to
// @Description  a different user (no leak).
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int                         true   "Owner user ID"
// @Param        id       path      int                         true   "Reservation ID"
// @Success      200       {object}  markNoShowResponse         "Reservation marked no-show"
// @Failure      400       {object}  errorResponse              "Invalid user_id / reservation_id, or activity has not started yet"
// @Failure      404       {object}  errorResponse              "Reservation or activity not found"
// @Failure      409       {object}  errorResponse              "Reservation not in CONFIRMED state (not_cancellable)"
// @Failure      500       {object}  errorResponse              "Internal server error"
// @Router       /api/v1/users/{user_id}/reservations/{id}/no-show [post]
func (h *ReservationHandler) MarkNoShow(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewMarkReservationNoShowInput(userID, reservationID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.noShow.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, markNoShowResponse{
		ID:     output.Reservation.ID(),
		Status: string(output.Reservation.Status()),
	})
}

// completeReservationResponse is the wire format for a successful
// reservation completion. Same shape as cancelReservationResponse
// (id + status); the status field will be COMPLETED.
type completeReservationResponse struct {
	ID     int    `json:"id"     example:"99"`
	Status string `json:"status" example:"COMPLETED"`
}

// CompleteReservation godoc
// @Summary      Mark a reservation as completed
// @Description  Transitions a CONFIRMED reservation to COMPLETED.
// @Description  The activity must have already finished
// @Description  (date + duration < now). Does not refund the pass
// @Description  session because the session was consumed at
// @Description  registration and the activity has been delivered.
// @Description  Owner-only: the user_id in the path must own the
// @Description  dog. Returns 404 if the reservation does not exist
// @Description  OR belongs to a different user (no leak).
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int                         true   "Owner user ID"
// @Param        id       path      int                         true   "Reservation ID"
// @Success      200       {object}  completeReservationResponse "Reservation marked completed"
// @Failure      400       {object}  errorResponse              "Invalid user_id / reservation_id, or activity has not finished yet"
// @Failure      404       {object}  errorResponse              "Reservation or activity not found"
// @Failure      409       {object}  errorResponse              "Reservation not in CONFIRMED state (not_completable)"
// @Failure      500       {object}  errorResponse              "Internal server error"
// @Router       /api/v1/users/{user_id}/reservations/{id}/complete [post]
func (h *ReservationHandler) CompleteReservation(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewCompleteReservationInput(userID, reservationID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.complete.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, completeReservationResponse{
		ID:     output.Reservation.ID(),
		Status: string(output.Reservation.Status()),
	})
}

// ============================================================================
// Read endpoints
// ============================================================================

// ListByUser godoc
// @Summary      List a user's reservations
// @Description  Returns the denormalized ReservationView for every
// @Description  reservation whose dog is owned by the user in the
// @Description  path, ordered by created_at DESC. Supports
// @Description  optional filters: status, from, to.
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int     true   "Owner user ID"
// @Param        status   query     string  false  "Filter by status (CONFIRMED, COMPLETED, CANCELLED_IN_TIME, CANCELLED_LATE, FORGIVEN, NO_SHOW)"
// @Param        from     query     string  false  "Filter by created_at >= from (RFC3339)"
// @Param        to       query     string  false  "Filter by created_at <  to (RFC3339)"
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/users/{user_id}/reservations [get]
func (h *ReservationHandler) ListByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	status, err := parseStatusFilter(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "status", Details: err.Error()})
		return
	}
	from, err := parseTimeFilter(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "from", Details: err.Error()})
		return
	}
	to, err := parseTimeFilter(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "to", Details: err.Error()})
		return
	}
	in, err := reservationuc.NewListByUserReservationsInput(userID, status, from, to, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listByUser.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ListUpcomingByUser godoc
// @Summary      List a user's upcoming reservations
// @Description  Returns the views of every CONFIRMED reservation
// @Description  whose activity date is at or after the current
// @Description  time, ordered by activity date ASC.
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int     true   "Owner user ID"
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/users/{user_id}/reservations/upcoming [get]
func (h *ReservationHandler) ListUpcomingByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, err := reservationuc.NewListUpcomingByUserInput(userID, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listUpcoming.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// GetByID godoc
// @Summary      Get a reservation by id
// @Description  Returns the denormalized ReservationView for the
// @Description  given reservation id, owned by the user in the
// @Description  path. Returns 404 if the id does not exist OR if
// @Description  the reservation belongs to a different user (no
// @Description  leak).
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int     true   "Owner user ID"
// @Param        id       path      int     true   "Reservation ID"
// @Success      200      {object}  reservationViewResponse
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/users/{user_id}/reservations/{id} [get]
func (h *ReservationHandler) GetByID(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewGetReservationInput(userID, reservationID)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.get.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, reservationViewResponse{Reservation: toReservationViewDTO(output.View)})
}

// ListByDog godoc
// @Summary      List a dog's reservations
// @Description  Returns the views of every reservation for the
// @Description  given dog, ordered by created_at DESC.
// @Tags         reservations
// @Produce      json
// @Param        id       path      int     true   "Dog ID"
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/dogs/{id}/reservations [get]
func (h *ReservationHandler) ListByDog(c *gin.Context) {
	dogID, err := strconv.Atoi(c.Param("id"))
	if err != nil || dogID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "dog_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, err := reservationuc.NewListByDogReservationsInput(dogID, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listByDog.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ListByPass godoc
// @Summary      List a pass's reservations
// @Description  Returns the views of every reservation paid from
// @Description  the given pass, ordered by created_at DESC.
// @Description  Pass audit view.
// @Tags         reservations
// @Produce      json
// @Param        id       path      int     true   "Pass ID"
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/passes/{id}/reservations [get]
func (h *ReservationHandler) ListByPass(c *gin.Context) {
	passID, err := strconv.Atoi(c.Param("id"))
	if err != nil || passID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "pass_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, err := reservationuc.NewListByPassReservationsInput(passID, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listByPass.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ListByActivity godoc
// @Summary      List an activity's reservations
// @Description  Returns the views of every reservation for the
// @Description  given activity, ordered by created_at ASC. Class
// @Description  roster view.
// @Tags         reservations
// @Produce      json
// @Param        id       path      int     true   "Activity ID"
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /api/v1/activities/{id}/reservations [get]
func (h *ReservationHandler) ListByActivity(c *gin.Context) {
	activityID, err := strconv.Atoi(c.Param("id"))
	if err != nil || activityID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "activity_id"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, err := reservationuc.NewListByActivityReservationsInput(activityID, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listByActivity.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ============================================================================
// Shared DTOs (enriched reservation view)
// ============================================================================

// reservationViewDTO is the wire format shared by every read
// endpoint (list + detail). The single shape is intentional: the
// client does not have to learn multiple envelopes to render
// different views of a reservation.
type reservationViewDTO struct {
	ID        int       `json:"id"                example:"42"`
	Status    string    `json:"status"            example:"CONFIRMED"`
	CreatedAt time.Time `json:"created_at"         example:"2026-07-20T10:00:00Z"`

	ActivityID       int       `json:"activity_id"        example:"10"`
	ActivityName     string    `json:"activity_name"      example:"Paseo Río"`
	ActivityDate     time.Time `json:"activity_date"      example:"2026-08-01T10:00:00Z"`
	ActivityLocation string    `json:"activity_location"  example:"Parking Central"`
	ActivityClosed   bool      `json:"activity_closed"    example:"false"`

	DogID   int    `json:"dog_id"             example:"5"`
	DogName string `json:"dog_name"           example:"Luna"`

	PassID        int    `json:"pass_id"            example:"3"`
	PassType      string `json:"pass_type"          example:"GENERICO"`
	PassRemaining int    `json:"pass_remaining"     example:"7"`
}

// reservationViewResponse wraps a single reservationViewDTO for the
// detail endpoint. Keeping a single-element envelope keeps the wire
// format consistent with the list endpoint (both are objects with a
// stable top-level key, never a bare array).
type reservationViewResponse struct {
	Reservation reservationViewDTO `json:"reservation"`
}

// listReservationsResponse is the wire format for every list
// endpoint. The pagination fields (limit, offset, count) are
// always present and reflect the normalized values that the use
// case actually used (read from the validated input).
type listReservationsResponse struct {
	Reservations []reservationViewDTO `json:"reservations"`
	Limit        int                  `json:"limit"`
	Offset       int                  `json:"offset"`
	Count        int                  `json:"count"`
}

// reservationPagination is the contract every reservation list
// input satisfies: it exposes the (already normalized) limit and
// offset. Defined here as a tiny private interface so we can
// build the response envelope from any list input uniformly.
type reservationPagination interface {
	Limit() int
	Offset() int
}

// toListReservationsResponse serializes the views and reads the
// (already normalized) limit/offset from the validated input.
func toListReservationsResponse(views []*domain.ReservationView, in reservationPagination) listReservationsResponse {
	dtos := make([]reservationViewDTO, len(views))
	for i, view := range views {
		dtos[i] = toReservationViewDTO(view)
	}
	return listReservationsResponse{
		Reservations: dtos,
		Limit:        in.Limit(),
		Offset:       in.Offset(),
		Count:        len(dtos),
	}
}

// toReservationViewDTO converts a domain.ReservationView into the
// wire format. Pure function: no business logic, no I/O.
func toReservationViewDTO(view *domain.ReservationView) reservationViewDTO {
	return reservationViewDTO{
		ID:               view.ID(),
		Status:           string(view.Status()),
		CreatedAt:        view.CreatedAt(),
		ActivityID:       view.ActivityID(),
		ActivityName:     view.ActivityName(),
		ActivityDate:     view.ActivityDate(),
		ActivityLocation: view.ActivityLocation(),
		ActivityClosed:   view.Activity().IsClosed(),
		DogID:            view.DogID(),
		DogName:          view.DogName(),
		PassID:           view.PassID(),
		PassType:         string(view.PassType()),
		PassRemaining:    view.PassRemaining(),
	}
}

// parseStatusFilter converts an optional ?status= query param into
// a *domain.ReservationStatus. Returns nil for empty (no filter).
// The factory validates the enum.
func parseStatusFilter(raw string) (*domain.ReservationStatus, error) {
	if raw == "" {
		return nil, nil
	}
	status := domain.ReservationStatus(raw)
	return &status, nil
}

// parseTimeFilter converts an optional ?from=/?to= query param
// (RFC3339) into a *time.Time. Returns nil for empty (no filter).
// Returns an error for non-empty but malformed values. The
// factory validates the from<=to range.
func parseTimeFilter(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
