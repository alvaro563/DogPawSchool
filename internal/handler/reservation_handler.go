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

type AdminReservationRegisterer interface {
	Execute(ctx context.Context, input reservationuc.RegisterAdminReservationInput) (reservationuc.RegisterAdminReservationOutput, error)
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

type ReservationListerAll interface {
	Execute(ctx context.Context, input reservationuc.ListAllReservationsInput) (reservationuc.ListAllReservationsOutput, error)
}

type ReservationUpcomingAllLister interface {
	Execute(ctx context.Context, input reservationuc.ListUpcomingAllInput) (reservationuc.ListUpcomingAllOutput, error)
}

type ReservationNoShower interface {
	Execute(ctx context.Context, input reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error)
}

type ReservationCompleter interface {
	Execute(ctx context.Context, input reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error)
}

type ReservationConfirmer interface {
	Execute(ctx context.Context, input reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error)
}

type ReservationRejecter interface {
	Execute(ctx context.Context, input reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error)
}

type ReservationForgiver interface {
	Execute(ctx context.Context, input reservationuc.ForgiveReservationInput) (reservationuc.ForgiveReservationOutput, error)
}

type ActivityRosterGetter interface {
	Execute(ctx context.Context, input reservationuc.ListActivityRosterInput) (reservationuc.ListActivityRosterOutput, error)
}

type PendingReservationsGetter interface {
	Execute(ctx context.Context, input reservationuc.ListPendingReservationsInput) (reservationuc.ListPendingReservationsOutput, error)
}

// ReservationHandler owns the HTTP entry points for reservation
// use cases. It exposes 12 use cases (Register, Cancel, Get,
// ListByUser, ListUpcomingByUser, ListByDog, ListByPass,
// ListByActivity, MarkNoShow, CompleteReservation, ConfirmPending,
// RejectPending) plus the admin-only RegisterAdmin and the
// class-day roster read (ListActivityRoster).
type ReservationHandler struct {
	register       ReservationRegisterer
	adminRegister  AdminReservationRegisterer
	cancel         ReservationCanceler
	get            ReservationGetter
	listByUser     ReservationListerByUser
	listUpcoming   ReservationListerUpcomingByUser
	listByDog      ReservationListerByDog
	listByPass     ReservationListerByPass
	listByActivity ReservationListerByActivity
	noShow         ReservationNoShower
	complete       ReservationCompleter
	confirm        ReservationConfirmer
	reject         ReservationRejecter
	forgive        ReservationForgiver
	listAll        ReservationListerAll
	listUpcomingAll ReservationUpcomingAllLister
	activityRoster ActivityRosterGetter
	listPending    PendingReservationsGetter
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
	confirm ReservationConfirmer,
	reject ReservationRejecter,
	forgive ReservationForgiver,
	listAll ReservationListerAll,
	listUpcomingAll ReservationUpcomingAllLister,
	adminRegister AdminReservationRegisterer,
	activityRoster ActivityRosterGetter,
	listPending PendingReservationsGetter,
) *ReservationHandler {
	return &ReservationHandler{
		register:       register,
		adminRegister:  adminRegister,
		cancel:         cancel,
		get:            get,
		listByUser:     listByUser,
		listUpcoming:   listUpcoming,
		listByDog:      listByDog,
		listByPass:     listByPass,
		listByActivity: listByActivity,
		noShow:         noShow,
		complete:       complete,
		confirm:        confirm,
		reject:         reject,
		forgive:        forgive,
		listAll:        listAll,
		listUpcomingAll: listUpcomingAll,
		activityRoster: activityRoster,
		listPending:    listPending,
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
// @Security     BearerAuth
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

	if !RequireOwnershipOrAdmin(c, userID) {
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
	c.JSON(http.StatusCreated, registerReservationResponse{
		ID:             output.ID,
		Status:         string(output.Status),
		PendingReasons: formatPendingReasons(output.PendingReasons, nil),
	})
}

// RegisterAdmin godoc
// @Summary      Create an admin reservation
// @Description  Books any dog with any usable school pass. Admin authority
// @Description  bypasses ownership and compatibility conflicts and confirms
// @Description  the reservation immediately.
// @Tags         reservations
// @Accept       json
// @Produce      json
// @Param        reservation body registerReservationRequest true "Reservation to create"
// @Success      201 {object} registerReservationResponse
// @Failure      400 {object} errorResponse
// @Failure      409 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Security     BearerAuth
// @Router       /api/v1/reservations [post]
func (h *ReservationHandler) RegisterAdmin(c *gin.Context) {
	var request registerReservationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Details: err.Error()})
		return
	}
	in, err := reservationuc.NewRegisterAdminReservationInput(request.ActivityID, request.DogID, request.PassID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.adminRegister.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Location", "/api/v1/reservations/"+strconv.Itoa(output.ID))
	c.JSON(http.StatusCreated, registerReservationResponse{
		ID:             output.ID,
		Status:         string(output.Status),
		PendingReasons: formatPendingReasons(output.PendingReasons, nil),
	})
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
	ID             int      `json:"id"             example:"99"`
	Status         string   `json:"status"         example:"CONFIRMED"`
	// PendingReasons carries the user-facing Spanish explanations for
	// each reason the reservation was held in StatusPendingToConfirm.
	// Populated only when Status is "PENDING_TO_CONFIRM" and non-empty.
	PendingReasons []string `json:"pending_reasons,omitempty"`
}

// confirmPendingReservationResponse is the wire format for a
// successful admin confirm/reject. Same shape as cancelReservationResponse
// (id + status); the status field will be CONFIRMED after a confirm and
// CANCELLED_IN_TIME after a reject.
type confirmPendingReservationResponse struct {
	ID     int    `json:"id"     example:"99"`
	Status string `json:"status" example:"CONFIRMED"`
}

// ConfirmPending godoc
// @Summary      Confirm a pending reservation
// @Description  Admin-only. Promotes a PENDING_TO_CONFIRM reservation
// @Description  (created with only MEDIA/BAJA compatibility conflicts)
// @Description  to CONFIRMED. The slot is already held, so no capacity
// @Description  re-check is performed.
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int                              true   "Owner user ID"
// @Param        id       path      int                              true   "Reservation ID"
// @Success      200      {object}  confirmPendingReservationResponse "Reservation confirmed"
// @Failure      400      {object}  errorResponse                     "Invalid id"
// @Failure      404      {object}  errorResponse                     "Reservation not found"
// @Failure      409      {object}  errorResponse                     "Reservation is not pending (not_pending)"
// @Failure      500      {object}  errorResponse                     "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations/{id}/confirm [post]
func (h *ReservationHandler) ConfirmPending(c *gin.Context) {
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewConfirmPendingReservationInput(reservationID)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.confirm.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, confirmPendingReservationResponse{
		ID:     output.Reservation.ID(),
		Status: string(output.Reservation.Status()),
	})
}

// RejectPending godoc
// @Summary      Reject a pending reservation
// @Description  Admin-only. Demotes a PENDING_TO_CONFIRM reservation to
// @Description  CANCELLED_IN_TIME (freeing the slot) and refunds the
// @Description  pass session consumed at booking.
// @Tags         reservations
// @Produce      json
// @Param        user_id  path      int                              true   "Owner user ID"
// @Param        id       path      int                              true   "Reservation ID"
// @Success      200      {object}  confirmPendingReservationResponse "Reservation rejected"
// @Failure      400      {object}  errorResponse                     "Invalid id"
// @Failure      404      {object}  errorResponse                     "Reservation not found"
// @Failure      409      {object}  errorResponse                     "Reservation is not pending (not_pending)"
// @Failure      500      {object}  errorResponse                     "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations/{id}/reject [post]
func (h *ReservationHandler) RejectPending(c *gin.Context) {
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewRejectPendingReservationInput(reservationID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.reject.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, confirmPendingReservationResponse{
		ID:     output.Reservation.ID(),
		Status: string(output.Reservation.Status()),
	})
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
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations/{id}/cancel [post]
func (h *ReservationHandler) Cancel(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}

	if !RequireOwnershipOrAdmin(c, userID) {
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

// CancelAdmin godoc
// @Summary      Admin-cancel a reservation
// @Description  Admin-only. Cancels any CONFIRMED reservation
// @Description  regardless of the owner. The owner user_id is not
// @Description  required in the path because the admin does not need
// @Description  to know who owns the reservation to cancel it.
// @Description  In-time vs late window policy and refund rules are
// @Description  identical to the user-cancel endpoint above.
// @Tags         reservations
// @Produce      json
// @Param        id       path      int                          true   "Reservation ID"
// @Success      200      {object}  cancelReservationResponse    "Reservation cancelled"
// @Failure      400      {object}  errorResponse                "Invalid reservation_id"
// @Failure      404      {object}  errorResponse                "Reservation not found"
// @Failure      409      {object}  errorResponse                "Already cancelled / activity in past"
// @Failure      500      {object}  errorResponse                "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/reservations/{id}/cancel [post]
func (h *ReservationHandler) CancelAdmin(c *gin.Context) {
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewCancelReservationAdminInput(reservationID, time.Now)
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

// forgiveReservationResponse is the wire format for a successful
// forgive. The pass_session_refunded field mirrors the use case's
// output: when false (fresh pass with nothing to refund), the client
// should show an honest "forgiven but session was not refundable"
// message.
type forgiveReservationResponse struct {
	ID                  int    `json:"id"                  example:"99"`
	Status              string `json:"status"              example:"FORGIVEN"`
	PassSessionRefunded bool   `json:"pass_session_refunded" example:"true"`
}

// Forgive godoc
// @Summary      Forgive a late-cancelled reservation
// @Description  Admin-only. Transitions a CANCELLED_LATE reservation
// @Description  to FORGIVEN and, when the pass has available balance,
// @Description  refunds the consumed session. The only path that
// @Description  converts a late cancellation into a pass refund.
// @Tags         reservations
// @Produce      json
// @Param        id   path      int     true  "Reservation ID"
// @Success      200  {object}  forgiveReservationResponse "Reservation forgiven"
// @Failure      400  {object}  errorResponse              "Invalid reservation_id"
// @Failure      404  {object}  errorResponse              "Reservation not found"
// @Failure      409  {object}  errorResponse              "Reservation is not in a state that can be forgiven"
// @Failure      500  {object}  errorResponse              "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/reservations/{id}/forgive [post]
func (h *ReservationHandler) Forgive(c *gin.Context) {
	reservationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || reservationID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "reservation_id"})
		return
	}
	in, err := reservationuc.NewForgiveReservationInput(reservationID, time.Now)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.forgive.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, forgiveReservationResponse{
		ID:                  output.Reservation.ID(),
		Status:              string(output.Reservation.Status()),
		PassSessionRefunded: output.PassSessionRefunded,
	})
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
// @Security     BearerAuth
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
// @Security     BearerAuth
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
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations [get]
func (h *ReservationHandler) ListByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}

	if !RequireOwnershipOrAdmin(c, userID) {
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
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations/upcoming [get]
func (h *ReservationHandler) ListUpcomingByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}

	if !RequireOwnershipOrAdmin(c, userID) {
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
// @Security     BearerAuth
// @Router       /api/v1/users/{user_id}/reservations/{id} [get]
func (h *ReservationHandler) GetByID(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "user_id"})
		return
	}

	if !RequireOwnershipOrAdmin(c, userID) {
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
// @Security     BearerAuth
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
// @Security     BearerAuth
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
// @Security     BearerAuth
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

// ListAll godoc
// @Summary      List all reservations
// @Description  Returns a paginated list of every reservation in the
// @Description  system, most recent first. Admin only.
// @Tags         reservations
// @Produce      json
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Param        status   query     string  false  "Filter by status (CONFIRMED, PENDING_TO_CONFIRM, COMPLETED, CANCELLED_IN_TIME, CANCELLED_LATE, FORGIVEN, NO_SHOW). Empty = no filter."
// @Success      200      {object}  listReservationsResponse
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Security     BearerAuth
// @Router       /api/v1/reservations [get]
func (h *ReservationHandler) ListAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	status, err := parseStatusFilter(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "validation",
			Field:   "status",
			Details: err.Error(),
		})
		return
	}
	in, err := reservationuc.NewListAllReservationsInput(limit, offset, status)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listAll.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ListUpcomingAll godoc
// @Summary      List all upcoming reservations
// @Description  Returns a paginated list of every CONFIRMED reservation
// @Description  whose activity date is at or after now, soonest first.
// @Description  Admin only.
// @Tags         reservations
// @Produce      json
// @Param        limit    query     int     false  "Maximum number of reservations to return (default 50, max 100)"
// @Param        offset   query     int     false  "Number of reservations to skip for pagination (default 0)"
// @Success      200      {object}  listReservationsResponse
// @Failure      500      {object}  errorResponse
// @Security     BearerAuth
// @Router       /api/v1/reservations/upcoming [get]
func (h *ReservationHandler) ListUpcomingAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, _ := reservationuc.NewListUpcomingAllInput(limit, offset)
	output, err := h.listUpcomingAll.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toListReservationsResponse(output.Views, in))
}

// ListPending godoc
// @Summary      List pending reservations awaiting admin approval
// @Description  Returns every reservation in StatusPendingToConfirm,
// @Description  each annotated with the dog owner's id and name.
// @Description  Server-side partitioning and batched owner lookup
// @Description  keep the response shape stable: the admin can
// @Description  render the triage page with a single fetch and
// @Description  zero filtering. Admin only.
// @Tags         reservations
// @Produce      json
// @Param        limit   query     int                          false  "Maximum number of pending reservations to return (default 50, max 100)"
// @Param        offset  query     int                          false  "Number of pending reservations to skip for pagination (default 0)"
// @Success      200     {object}  pendingReservationsResponse  "Pending reservations"
// @Failure      500     {object}  errorResponse                "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/reservations/pending [get]
func (h *ReservationHandler) ListPending(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	in, err := reservationuc.NewListPendingReservationsInput(limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.listPending.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPendingReservationsResponse(output, in))
}

// pendingReservationsResponse is the wire format for the admin
// "pending to approve" list page. It deliberately exposes owner_id
// and owner_name per entry so the client can render each row
// without an extra user lookup. Pagination fields mirror the
// envelope used by the other list endpoints.
type pendingReservationsResponse struct {
	Pending []pendingReservationEntryDTO `json:"pending"`
	Limit   int                          `json:"limit"`
	Offset  int                          `json:"offset"`
	Count   int                          `json:"count"`
}

// pendingReservationEntryDTO is the wire shape for a single pending
// reservation. It mirrors ActivityRosterEntry for the activity-
// scoped view but adds activity_date and activity_location because
// the "pending" page needs to render the class context, not just
// the attendance.
type pendingReservationEntryDTO struct {
	ReservationID    int       `json:"reservation_id"    example:"42"`
	DogID            int       `json:"dog_id"            example:"5"`
	DogName          string    `json:"dog_name"          example:"Luna"`
	OwnerID          int       `json:"owner_id"          example:"7"`
	OwnerName        string    `json:"owner_name"        example:"Carlos García"`
	ActivityID       int       `json:"activity_id"       example:"10"`
	ActivityName     string    `json:"activity_name"     example:"Paseo Río"`
	ActivityDate     time.Time `json:"activity_date"     example:"2026-08-01T10:00:00Z"`
	ActivityLocation string    `json:"activity_location" example:"Parking Central"`
}

// toPendingReservationsResponse converts the use case output into
// the wire format. Pagination fields are read from the validated
// input so the client sees the values that the use case actually
// applied (mirrors the convention used by every other list
// endpoint in this handler).
func toPendingReservationsResponse(
	out reservationuc.ListPendingReservationsOutput,
	in reservationuc.ListPendingReservationsInput,
) pendingReservationsResponse {
	entries := make([]pendingReservationEntryDTO, len(out.Pending))
	for i, e := range out.Pending {
		entries[i] = pendingReservationEntryDTO{
			ReservationID:    e.ReservationID(),
			DogID:            e.DogID(),
			DogName:          e.DogName(),
			OwnerID:          e.OwnerID(),
			OwnerName:        e.OwnerName(),
			ActivityID:       e.ActivityID(),
			ActivityName:     e.ActivityName(),
			ActivityDate:     e.ActivityDate(),
			ActivityLocation: e.ActivityLocation(),
		}
	}
	return pendingReservationsResponse{
		Pending: entries,
		Limit:   in.Limit(),
		Offset:  in.Offset(),
		Count:   len(entries),
	}
}

// ListActivityRoster godoc
// @Summary      List an activity's roster (admin class-day view)
// @Description  Returns the activity together with its confirmed
// @Description  and pending-to-confirm attendees, each annotated
// @Description  with the dog owner's id and name. Server-side
// @Description  partitioning keeps the response shape stable for
// @Description  the admin dashboard: the client can render the
// @Description  page with a single fetch. Admin only.
// @Tags         reservations
// @Produce      json
// @Param        id   path      int                       true  "Activity ID"
// @Success      200  {object}  activityRosterResponse   "Activity roster"
// @Failure      400  {object}  errorResponse            "Invalid id"
// @Failure      404  {object}  errorResponse            "Activity not found"
// @Failure      500  {object}  errorResponse            "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/admin/activities/{id}/roster [get]
func (h *ReservationHandler) ListActivityRoster(c *gin.Context) {
	activityID, err := strconv.Atoi(c.Param("id"))
	if err != nil || activityID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "validation", Field: "activity_id"})
		return
	}
	in, err := reservationuc.NewListActivityRosterInput(activityID)
	if err != nil {
		writeError(c, err)
		return
	}
	output, err := h.activityRoster.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityRosterResponse(output))
}

// activityRosterResponse is the wire format for the admin
// class-day roster. It deliberately nests activity + confirmed +
// pending under named keys so the client can render three sections
// without any grouping logic.
type activityRosterResponse struct {
	Activity  activityDTO             `json:"activity"`
	Confirmed []activityRosterEntryDTO `json:"confirmed"`
	Pending   []activityRosterEntryDTO `json:"pending"`
}

// activityRosterEntryDTO is the wire shape for a single attendee
// on the roster. It is intentionally flat (no nested aggregates) so
// the client can render the row directly without unwrapping.
type activityRosterEntryDTO struct {
	ReservationID int    `json:"reservation_id" example:"42"`
	DogID         int    `json:"dog_id"         example:"5"`
	DogName       string `json:"dog_name"       example:"Luna"`
	OwnerID       int    `json:"owner_id"       example:"7"`
	OwnerName     string `json:"owner_name"     example:"Carlos García"`
}

// toActivityRosterResponse converts the use case output into the
// wire format. The activity is reused from the same DTO shape used
// by the read-activity endpoints (computed spots are intentionally
// omitted: the roster is a class-day view, the capacity is read
// directly from the activity aggregate via available_spots in a
// separate call when the client needs it).
func toActivityRosterResponse(out reservationuc.ListActivityRosterOutput) activityRosterResponse {
	return activityRosterResponse{
		Activity: activityDTO{
			ID:               out.Activity.ID(),
			Name:             out.Activity.Name(),
			ActivityType:     string(out.Activity.Type()),
			Location:         out.Activity.Location(),
			MaxCapacity:      out.Activity.MaxCapacity(),
			DurationInHours:  out.Activity.DurationInHours(),
			Date:             out.Activity.Date(),
			Closed:           out.Activity.IsClosed(),
		},
		Confirmed: toActivityRosterEntryDTOs(out.Confirmed),
		Pending:   toActivityRosterEntryDTOs(out.Pending),
	}
}

func toActivityRosterEntryDTOs(entries []reservationuc.ActivityRosterEntry) []activityRosterEntryDTO {
	dtos := make([]activityRosterEntryDTO, len(entries))
	for i, e := range entries {
		dtos[i] = activityRosterEntryDTO{
			ReservationID: e.ReservationID(),
			DogID:         e.DogID(),
			DogName:       e.DogName(),
			OwnerID:       e.OwnerID(),
			OwnerName:     e.OwnerName(),
		}
	}
	return dtos
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

	OwnerID int `json:"owner_id"             example:"7"`

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
		OwnerID:          view.DogUserID(),
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
