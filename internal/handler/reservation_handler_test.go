package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
	reservationuc "dogpaw/internal/usecase/reservation"
)

var fixedNow = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

type stubReservationRegisterer struct {
	fn func(ctx context.Context, in reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error)
}

func (s *stubReservationRegisterer) Execute(ctx context.Context, in reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationCanceler struct {
	fn func(ctx context.Context, in reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error)
}

func (s *stubReservationCanceler) Execute(ctx context.Context, in reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationNoShower struct {
	fn func(ctx context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error)
}

func (s *stubReservationNoShower) Execute(ctx context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationCompleter struct {
	fn func(ctx context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error)
}

func (s *stubReservationCompleter) Execute(ctx context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationConfirmer struct {
	fn func(ctx context.Context, in reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error)
}

func (s *stubReservationConfirmer) Execute(ctx context.Context, in reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationRejecter struct {
	fn func(ctx context.Context, in reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error)
}

func (s *stubReservationRejecter) Execute(ctx context.Context, in reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error) {
	return s.fn(ctx, in)
}

func newReservationHandler(
	reg ReservationRegisterer,
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
) *ReservationHandler {
	return NewReservationHandler(reg, cancel, get, listByUser, listUpcoming, listByDog, listByPass, listByActivity, noShow, complete, confirm, reject)
}

func newReservationHandlerReg(reg ReservationRegisterer) *ReservationHandler {
	return newReservationHandler(reg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerCancel(cancel ReservationCanceler) *ReservationHandler {
	return newReservationHandler(nil, cancel, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

// newCancelledReservation builds a domain.Reservation in the given
// terminal status. Used by handler tests that need a
// *domain.Reservation to wrap in a CancelReservationOutput.
func newCancelledReservation(id int, status domain.ReservationStatus) *domain.Reservation {
	reservation, err := domain.NewReservationWithStatus(id, 10, 20, 30, status, fixedNow)
	if err != nil {
		panic(err)
	}
	return reservation
}

func validRegisterReservationBody() string {
	return `{"activity_id":42,"dog_id":7,"pass_id":3}`
}

// TestReservationRegister_Success verifies the happy-path POST
// creates the resource, returns 201 with the new id, and sets the
// Location header.
func TestReservationRegister_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRegisterer{
		fn: func(_ context.Context, in reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			assert.Equal(t, 1, in.UserID(), "user_id comes from the path, not the body")
			assert.Equal(t, 42, in.ActivityID())
			assert.Equal(t, 7, in.DogID())
			assert.Equal(t, 3, in.PassID())
			return reservationuc.RegisterReservationOutput{ID: 99}, nil
		},
	}
	h := newReservationHandlerReg(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/reservations/99", w.Header().Get("Location"))
	var body registerReservationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 99, body.ID)
}

// TestReservationRegister_InvalidUserID verifies that a non-numeric
// or non-positive path param yields 400 validation.
func TestReservationRegister_InvalidUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerReg(&stubReservationRegisterer{
				fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
					t.Fatal("use case should not be called on bad user_id")
					return reservationuc.RegisterReservationOutput{}, nil
				},
			})
			c, w := setupCtx(http.MethodPost, "/api/v1/users/"+tt.pathID+"/reservations", validRegisterReservationBody())
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathID}}

			h.Register(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

// TestReservationRegister_InvalidBody verifies that a malformed
// JSON body yields 400 invalid_request.
func TestReservationRegister_InvalidBody(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			t.Fatal("use case should not be called on invalid body")
			return reservationuc.RegisterReservationOutput{}, nil
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", `not json`, withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"invalid_request"`)
}

// TestReservationRegister_MissingFields verifies that Gin's binding
// rejects a body with zero-valued ids.
func TestReservationRegister_MissingFields(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			t.Fatal("use case should not be called on missing fields")
			return reservationuc.RegisterReservationOutput{}, nil
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", `{"activity_id":0,"dog_id":0,"pass_id":0}`, withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// The factory is the sole validator (Q2). Zero ids in the body
	// are caught by NewRegisterReservationInput, not by gin binding,
	// so the response is the standard validation envelope with the
	// field name.
	assert.Contains(t, w.Body.String(), `"field":"activity_id"`)
}

// TestReservationRegister_UseCaseValidation verifies that a
// *ValidationError from the use case maps to 400 with the field
// name (here, defending against a future validator that might run
// in the use case rather than the handler).
func TestReservationRegister_UseCaseValidation(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, &reservationuc.ValidationError{Field: "activity_id"}
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"activity_id"`)
}

// TestReservationRegister_ActivityInPast verifies ErrActivityInPast
// maps to 400 activity_in_past.
func TestReservationRegister_ActivityInPast(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrActivityInPast
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_in_past"`)
}

// TestReservationRegister_ActivityFull verifies ErrActivityFull
// maps to 409 activity_full.
func TestReservationRegister_ActivityFull(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrActivityFull
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_full"`)
}

// TestReservationRegister_PassExhausted verifies ErrPassExhausted
// maps to 400 pass_exhausted.
func TestReservationRegister_PassExhausted(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrPassExhausted
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"pass_exhausted"`)
}

// TestReservationRegister_PassExpired verifies ErrPassExpired maps
// to 400 pass_expired.
func TestReservationRegister_PassExpired(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrPassExpired
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"pass_expired"`)
}

// TestReservationRegister_DuplicateReservation verifies
// ErrDuplicateReservationForDog maps to 409.
func TestReservationRegister_DuplicateReservation(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrDuplicateReservationForDog
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"duplicate_reservation"`)
}

// TestReservationRegister_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationRegister_InternalError(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

func TestReservationRegister_Forbidden(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerReg(&stubReservationRegisterer{fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return reservationuc.RegisterReservationOutput{}, nil
	}})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/99/reservations", validRegisterReservationBody(), withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}}
	h.Register(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

// ============================================================================
// Cancel endpoint tests
// ============================================================================

// TestReservationCancel_SuccessInTime verifies the happy-path POST
// returns 200 with the new CANCELLED_IN_TIME status.
func TestReservationCancel_SuccessInTime(t *testing.T) {
	t.Parallel()
	reservation := newCancelledReservation(99, domain.StatusCancelledInTime)
	stub := &stubReservationCanceler{
		fn: func(_ context.Context, in reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			assert.Equal(t, 1, in.UserID(), "user_id comes from the path, not the body")
			assert.Equal(t, 99, in.ReservationID())
			return reservationuc.CancelReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerCancel(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body cancelReservationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 99, body.ID)
	assert.Equal(t, "CANCELLED_IN_TIME", body.Status)
}

// TestReservationCancel_SuccessLate verifies the late-cancel path
// returns 200 with CANCELLED_LATE (no refund is the client's
// problem to surface, not the server's).
func TestReservationCancel_SuccessLate(t *testing.T) {
	t.Parallel()
	reservation := newCancelledReservation(99, domain.StatusCancelledLate)
	stub := &stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerCancel(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body cancelReservationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "CANCELLED_LATE", body.Status)
}

// TestReservationCancel_InvalidUserID verifies that a non-numeric
// or non-positive path user_id yields 400 validation.
func TestReservationCancel_InvalidUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerCancel(&stubReservationCanceler{
				fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
					t.Fatal("use case should not be called on bad user_id")
					return reservationuc.CancelReservationOutput{}, nil
				},
			})
			c, w := setupCtx(http.MethodPost, "/api/v1/users/"+tt.pathID+"/reservations/99/cancel", "")
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathID}, {Key: "id", Value: "99"}}

			h.Cancel(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

// TestReservationCancel_InvalidReservationID verifies that a
// non-numeric or non-positive path reservation id yields 400
// validation.
func TestReservationCancel_InvalidReservationID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "xyz"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerCancel(&stubReservationCanceler{
				fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
					t.Fatal("use case should not be called on bad reservation_id")
					return reservationuc.CancelReservationOutput{}, nil
				},
			})
			c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/"+tt.pathID+"/cancel", "", withUserID(1))
			c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: tt.pathID}}

			h.Cancel(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"reservation_id"`)
		})
	}
}

// TestReservationCancel_AlreadyCancelled verifies ErrAlreadyCancelled
// maps to 409.
func TestReservationCancel_AlreadyCancelled(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrAlreadyCancelled
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"already_cancelled"`)
}

// TestReservationCancel_ActivityInPast verifies ErrActivityInPast
// maps to 400.
func TestReservationCancel_ActivityInPast(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrActivityInPast
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_in_past"`)
}

// TestReservationCancel_InvalidReservationID_NotFound verifies
// ErrInvalidReservation maps to 404.
func TestReservationCancel_InvalidReservationID_NotFound(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationCancel_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationCancel_InternalError(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

func TestReservationCancel_Forbidden(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerCancel(&stubReservationCanceler{fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return reservationuc.CancelReservationOutput{}, nil
	}})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/99/reservations/99/cancel", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}, {Key: "id", Value: "99"}}
	h.Cancel(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

// ============================================================================
// Read endpoint tests
// ============================================================================

type stubReservationGetter struct {
	fn func(ctx context.Context, in reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error)
}

func (s *stubReservationGetter) Execute(ctx context.Context, in reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationListerByUser struct {
	fn func(ctx context.Context, in reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error)
}

func (s *stubReservationListerByUser) Execute(ctx context.Context, in reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationListerUpcomingByUser struct {
	fn func(ctx context.Context, in reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error)
}

func (s *stubReservationListerUpcomingByUser) Execute(ctx context.Context, in reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationListerByDog struct {
	fn func(ctx context.Context, in reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error)
}

func (s *stubReservationListerByDog) Execute(ctx context.Context, in reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationListerByPass struct {
	fn func(ctx context.Context, in reservationuc.ListByPassReservationsInput) (reservationuc.ListByPassReservationsOutput, error)
}

func (s *stubReservationListerByPass) Execute(ctx context.Context, in reservationuc.ListByPassReservationsInput) (reservationuc.ListByPassReservationsOutput, error) {
	return s.fn(ctx, in)
}

type stubReservationListerByActivity struct {
	fn func(ctx context.Context, in reservationuc.ListByActivityReservationsInput) (reservationuc.ListByActivityReservationsOutput, error)
}

func (s *stubReservationListerByActivity) Execute(ctx context.Context, in reservationuc.ListByActivityReservationsInput) (reservationuc.ListByActivityReservationsOutput, error) {
	return s.fn(ctx, in)
}

func newReservationHandlerGet(get ReservationGetter) *ReservationHandler {
	return newReservationHandler(nil, nil, get, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerListByUser(l ReservationListerByUser) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, l, nil, nil, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerListUpcoming(l ReservationListerUpcomingByUser) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, l, nil, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerListByDog(l ReservationListerByDog) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, l, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerListByPass(l ReservationListerByPass) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, l, nil, nil, nil, nil, nil)
}

func newReservationHandlerListByActivity(l ReservationListerByActivity) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, nil, l, nil, nil, nil, nil)
}

func newReservationHandlerNoShow(noShow ReservationNoShower) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, noShow, nil, nil, nil)
}

func newReservationHandlerComplete(complete ReservationCompleter) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, complete, nil, nil)
}

func sampleViewOwnedBy(userID int) *domain.ReservationView {
	now := fixedNow
	return mustSampleReservationView(
		42, 10, 20, userID, 30, userID,
		domain.StatusConfirmed, now,
		"Paseo Río", "Parking Central", now.Add(7*24*time.Hour),
		"Luna", 5,
	)
}

// mustSampleReservationView is the handler-test equivalent of the
// use case helper. Builds a fully-populated ReservationView that
// passes NewReservationView's consistency check (reservation/activity/
// dog/pass ids line up).
func mustSampleReservationView(
	id, activityID, dogID, dogUserID, passID, passUserID int,
	status domain.ReservationStatus,
	createdAt time.Time,
	activityName, activityLocation string,
	activityDate time.Time,
	dogName string,
	passRemaining int,
) *domain.ReservationView {
	reservation, err := domain.NewReservationWithStatus(id, activityID, dogID, passID, status, createdAt)
	if err != nil {
		panic(err)
	}
	activity := domain.MustNewActivity(activityID, activityName, "", activityLocation,
		domain.TypeRoute, 5, 1, activityDate)
	dog, err := domain.NewDog(dogID, dogName, "TestBreed", "ES-TEST",
		24, domain.SexMale, 10, dogUserID)
	if err != nil {
		panic(err)
	}
	pass := domain.MustNewPass(passID, 10, passRemaining, 1000, domain.PassGeneric,
		passUserID, createdAt, createdAt, nil)
	view, err := domain.NewReservationView(reservation, activity, dog, pass)
	if err != nil {
		panic(err)
	}
	return view
}

func TestListByUser_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationListerByUser{
		fn: func(_ context.Context, in reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			assert.Equal(t, 1, in.UserID())
			assert.Equal(t, 50, in.Limit())
			return reservationuc.ListByUserReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByUser(stub)
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listReservationsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Count)
	assert.Equal(t, 1, len(body.Reservations))
	assert.Equal(t, "Luna", body.Reservations[0].DogName)
	assert.Equal(t, "Paseo Río", body.Reservations[0].ActivityName)
	assert.False(t, body.Reservations[0].ActivityClosed)
}

func TestListByUser_InvalidUserID(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListByUser(&stubReservationListerByUser{
		fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			t.Fatal("use case should not be called")
			return reservationuc.ListByUserReservationsOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/abc/reservations", "")
	c.Params = gin.Params{{Key: "user_id", Value: "abc"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByUser_InvalidStatusFilter(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListByUser(&stubReservationListerByUser{
		fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			t.Fatal("use case should not be called on invalid filter")
			return reservationuc.ListByUserReservationsOutput{}, nil
		},
	})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations?status=BOGUS", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"status"`)
}

func TestListByUser_InvalidTimeFilter(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListByUser(&stubReservationListerByUser{
		fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			t.Fatal("use case should not be called on invalid time")
			return reservationuc.ListByUserReservationsOutput{}, nil
		},
	})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations?from=not-a-date", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByUser_Forbidden(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListByUser(&stubReservationListerByUser{fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return reservationuc.ListByUserReservationsOutput{}, nil
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/99/reservations", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

func TestListUpcomingByUser_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationListerUpcomingByUser{
		fn: func(_ context.Context, in reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error) {
			assert.Equal(t, 1, in.UserID())
			return reservationuc.ListUpcomingByUserOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListUpcoming(stub)
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations/upcoming", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListUpcomingByUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listReservationsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Count)
	assert.False(t, body.Reservations[0].ActivityClosed)
}

func TestListUpcomingByUser_InvalidUserID(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListUpcoming(&stubReservationListerUpcomingByUser{
		fn: func(context.Context, reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error) {
			t.Fatal("use case should not be called")
			return reservationuc.ListUpcomingByUserOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/0/reservations/upcoming", "")
	c.Params = gin.Params{{Key: "user_id", Value: "0"}}
	h.ListUpcomingByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListUpcomingByUser_Forbidden(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListUpcoming(&stubReservationListerUpcomingByUser{fn: func(context.Context, reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return reservationuc.ListUpcomingByUserOutput{}, nil
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/99/reservations/upcoming", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}}
	h.ListUpcomingByUser(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationGetter{
		fn: func(_ context.Context, in reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			assert.Equal(t, 1, in.UserID())
			assert.Equal(t, 99, in.ReservationID())
			return reservationuc.GetReservationOutput{View: sampleViewOwnedBy(1)}, nil
		},
	}
	h := newReservationHandlerGet(stub)
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body reservationViewResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.Reservation.ID)
	assert.Equal(t, "Luna", body.Reservation.DogName)
	assert.False(t, body.Reservation.ActivityClosed)
}

func TestGetByID_NotFound(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerGet(&stubReservationGetter{
		fn: func(context.Context, reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			return reservationuc.GetReservationOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetByID_NotOwned(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerGet(&stubReservationGetter{
		fn: func(context.Context, reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			return reservationuc.GetReservationOutput{}, reservationuc.ErrReservationNotOwned
		},
	})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code, "not owned must map to 404 (no leak)")
}

func TestGetByID_Forbidden(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerGet(&stubReservationGetter{fn: func(context.Context, reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
		t.Fatal("use case should not be called for forbidden request")
		return reservationuc.GetReservationOutput{}, nil
	}})
	c, w := setupAuthCtx(http.MethodGet, "/api/v1/users/99/reservations/99", "", withUserID(7))
	c.Params = gin.Params{{Key: "user_id", Value: "99"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"forbidden"`)
}

func TestListByDog_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationListerByDog{
		fn: func(_ context.Context, in reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error) {
			assert.Equal(t, 20, in.DogID())
			return reservationuc.ListByDogReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByDog(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/20/reservations", "")
	c.Params = gin.Params{{Key: "id", Value: "20"}}
	h.ListByDog(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByDog_InvalidDogID(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerListByDog(&stubReservationListerByDog{
		fn: func(context.Context, reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error) {
			t.Fatal("use case should not be called")
			return reservationuc.ListByDogReservationsOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/0/reservations", "")
	c.Params = gin.Params{{Key: "id", Value: "0"}}
	h.ListByDog(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByPass_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationListerByPass{
		fn: func(_ context.Context, in reservationuc.ListByPassReservationsInput) (reservationuc.ListByPassReservationsOutput, error) {
			assert.Equal(t, 30, in.PassID())
			return reservationuc.ListByPassReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByPass(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/passes/30/reservations", "")
	c.Params = gin.Params{{Key: "id", Value: "30"}}
	h.ListByPass(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByActivity_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationListerByActivity{
		fn: func(_ context.Context, in reservationuc.ListByActivityReservationsInput) (reservationuc.ListByActivityReservationsOutput, error) {
			assert.Equal(t, 10, in.ActivityID())
			return reservationuc.ListByActivityReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByActivity(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/10/reservations", "")
	c.Params = gin.Params{{Key: "id", Value: "10"}}
	h.ListByActivity(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// MarkNoShow endpoint tests
// ============================================================================

// newNoShowReservation builds a domain.Reservation in StatusNoShow.
// The UC's runInTx calls reservation.MarkNoShow() which transitions
// StatusConfirmed → StatusNoShow; the handler only serialises the
// result, so the stub returns the post-transition state.
func newNoShowReservation(id int) *domain.Reservation {
	reservation, err := domain.NewReservationWithStatus(id, 10, 20, 30, domain.StatusNoShow, fixedNow)
	if err != nil {
		panic(err)
	}
	return reservation
}

// TestReservationMarkNoShow_Success verifies the happy-path POST
// returns 200 with StatusNoShow.
func TestReservationMarkNoShow_Success(t *testing.T) {
	t.Parallel()
	reservation := newNoShowReservation(99)
	stub := &stubReservationNoShower{
		fn: func(_ context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			assert.Equal(t, 1, in.UserID())
			assert.Equal(t, 99, in.ReservationID())
			return reservationuc.MarkReservationNoShowOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerNoShow(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/no-show", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.MarkNoShow(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body markNoShowResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 99, body.ID)
	assert.Equal(t, "NO_SHOW", body.Status)
}

// TestReservationMarkNoShow_InvalidUserID verifies that a
// non-numeric or non-positive path user_id yields 400 validation.
func TestReservationMarkNoShow_InvalidUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerNoShow(&stubReservationNoShower{
				fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
					t.Fatal("use case should not be called on bad user_id")
					return reservationuc.MarkReservationNoShowOutput{}, nil
				},
			})
			c, w := setupCtx(http.MethodPost, "/api/v1/users/"+tt.pathID+"/reservations/99/no-show", "")
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathID}, {Key: "id", Value: "99"}}

			h.MarkNoShow(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

// TestReservationMarkNoShow_InvalidReservationID verifies that a
// non-numeric or non-positive path reservation id yields 400
// validation.
func TestReservationMarkNoShow_InvalidReservationID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "xyz"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerNoShow(&stubReservationNoShower{
				fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
					t.Fatal("use case should not be called on bad reservation_id")
					return reservationuc.MarkReservationNoShowOutput{}, nil
				},
			})
			c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/"+tt.pathID+"/no-show", "", withUserID(1))
			c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: tt.pathID}}

			h.MarkNoShow(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"reservation_id"`)
		})
	}
}

// TestReservationMarkNoShow_ActivityNotStarted verifies
// ErrActivityNotStarted maps to 400 activity_not_started.
func TestReservationMarkNoShow_ActivityNotStarted(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerNoShow(&stubReservationNoShower{
		fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			return reservationuc.MarkReservationNoShowOutput{}, reservationuc.ErrActivityNotStarted
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/no-show", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.MarkNoShow(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_not_started"`)
}

// TestReservationMarkNoShow_NotFound verifies ErrInvalidReservation
// maps to 404.
func TestReservationMarkNoShow_NotFound(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerNoShow(&stubReservationNoShower{
		fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			return reservationuc.MarkReservationNoShowOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/no-show", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.MarkNoShow(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationMarkNoShow_NotCancellable verifies ErrNotCancellable
// maps to 409 not_cancellable.
func TestReservationMarkNoShow_NotCancellable(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerNoShow(&stubReservationNoShower{
		fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			return reservationuc.MarkReservationNoShowOutput{}, reservationuc.ErrNotCancellable
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/no-show", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.MarkNoShow(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_cancellable"`)
}

// TestReservationMarkNoShow_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationMarkNoShow_InternalError(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerNoShow(&stubReservationNoShower{
		fn: func(context.Context, reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			return reservationuc.MarkReservationNoShowOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/no-show", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.MarkNoShow(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

// ============================================================================
// CompleteReservation endpoint tests
// ============================================================================

// newCompletedReservation builds a domain.Reservation in
// StatusCompleted. The UC's runInTx calls reservation.Complete()
// which transitions StatusConfirmed → StatusCompleted; the handler
// only serialises the result, so the stub returns the
// post-transition state.
func newCompletedReservation(id int) *domain.Reservation {
	reservation, err := domain.NewReservationWithStatus(id, 10, 20, 30, domain.StatusCompleted, fixedNow)
	if err != nil {
		panic(err)
	}
	return reservation
}

// TestReservationComplete_Success verifies the happy-path POST
// returns 200 with StatusCompleted.
func TestReservationComplete_Success(t *testing.T) {
	t.Parallel()
	reservation := newCompletedReservation(99)
	stub := &stubReservationCompleter{
		fn: func(_ context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			assert.Equal(t, 1, in.UserID())
			assert.Equal(t, 99, in.ReservationID())
			return reservationuc.CompleteReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerComplete(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/complete", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.CompleteReservation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body completeReservationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 99, body.ID)
	assert.Equal(t, "COMPLETED", body.Status)
}

// TestReservationComplete_InvalidUserID verifies that a non-numeric
// or non-positive path user_id yields 400 validation.
func TestReservationComplete_InvalidUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerComplete(&stubReservationCompleter{
				fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
					t.Fatal("use case should not be called on bad user_id")
					return reservationuc.CompleteReservationOutput{}, nil
				},
			})
			c, w := setupCtx(http.MethodPost, "/api/v1/users/"+tt.pathID+"/reservations/99/complete", "")
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathID}, {Key: "id", Value: "99"}}

			h.CompleteReservation(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

// TestReservationComplete_InvalidReservationID verifies that a
// non-numeric or non-positive path reservation id yields 400
// validation.
func TestReservationComplete_InvalidReservationID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_numeric", "xyz"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReservationHandlerComplete(&stubReservationCompleter{
				fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
					t.Fatal("use case should not be called on bad reservation_id")
					return reservationuc.CompleteReservationOutput{}, nil
				},
			})
			c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/"+tt.pathID+"/complete", "", withUserID(1))
			c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: tt.pathID}}

			h.CompleteReservation(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"reservation_id"`)
		})
	}
}

// TestReservationComplete_ActivityNotFinished verifies
// ErrActivityNotFinished maps to 400 activity_not_finished.
func TestReservationComplete_ActivityNotFinished(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerComplete(&stubReservationCompleter{
		fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			return reservationuc.CompleteReservationOutput{}, reservationuc.ErrActivityNotFinished
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/complete", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.CompleteReservation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_not_finished"`)
}

// TestReservationComplete_NotFound verifies ErrInvalidReservation
// maps to 404.
func TestReservationComplete_NotFound(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerComplete(&stubReservationCompleter{
		fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			return reservationuc.CompleteReservationOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/complete", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.CompleteReservation(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationComplete_NotCompletable verifies ErrNotCompletable
// maps to 409 not_completable.
func TestReservationComplete_NotCompletable(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerComplete(&stubReservationCompleter{
		fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			return reservationuc.CompleteReservationOutput{}, reservationuc.ErrNotCompletable
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/complete", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.CompleteReservation(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_completable"`)
}

// TestReservationComplete_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationComplete_InternalError(t *testing.T) {
	t.Parallel()
	h := newReservationHandlerComplete(&stubReservationCompleter{
		fn: func(context.Context, reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			return reservationuc.CompleteReservationOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/complete", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.CompleteReservation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

// TestReservationRegister_PendingStatus exposes the resolved status in
// the response when a MEDIA/BAJA conflict holds the slot.
func TestReservationRegister_PendingStatus(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{ID: 99, Status: domain.StatusPendingToConfirm}, nil
		},
	}
	h := newReservationHandlerReg(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body registerReservationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 99, body.ID)
	assert.Equal(t, string(domain.StatusPendingToConfirm), body.Status)
}

// TestReservationRegister_IncompatibleDogsConflict verifies an
// IncompatibleDogsError maps to 409 dog_incompatible.
func TestReservationRegister_IncompatibleDogsConflict(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, &reservationuc.IncompatibleDogsError{
				Conflicts: []domain.CompatibilityConflict{sampleHandlerConflict()},
			}
		},
	}
	h := newReservationHandlerReg(stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody(), withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"dog_incompatible"`)
}

func sampleHandlerConflict() domain.CompatibilityConflict {
	return domain.CompatibilityConflict{
		TriggerName:     "Reactivo a machos enteros",
		TriggerLevel:    domain.IncompatibilityLevelAbsoluta,
		TriggerDogID:    7,
		TriggerDogName:  "Rex",
		TargetTraitCode: "MACHO_ENTERO",
		TargetTraitName: "Macho entero (no castrado)",
		TargetDogID:     5,
		TargetDogName:   "Luna",
	}
}

// TestReservationConfirmPending_Success verifies the happy-path admin
// confirm.
func TestReservationConfirmPending_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationConfirmer{
		fn: func(_ context.Context, in reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error) {
			assert.Equal(t, 99, in.ReservationID())
			reservation := newCancelledReservation(99, domain.StatusConfirmed)
			return reservationuc.ConfirmPendingReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub, nil)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/confirm", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.ConfirmPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"CONFIRMED"`)
}

// TestReservationConfirmPending_InvalidID verifies a non-numeric id
// yields 400 validation.
func TestReservationConfirmPending_InvalidID(t *testing.T) {
	t.Parallel()
	stub := &stubReservationConfirmer{
		fn: func(context.Context, reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error) {
			t.Fatal("use case should not be called on invalid id")
			return reservationuc.ConfirmPendingReservationOutput{}, nil
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub, nil)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/abc/confirm", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "abc"}}

	h.ConfirmPending(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"reservation_id"`)
}

// TestReservationConfirmPending_NotPending verifies ErrNotPending maps
// to 409 not_pending.
func TestReservationConfirmPending_NotPending(t *testing.T) {
	t.Parallel()
	stub := &stubReservationConfirmer{
		fn: func(context.Context, reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error) {
			return reservationuc.ConfirmPendingReservationOutput{}, reservationuc.ErrNotPending
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub, nil)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/confirm", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.ConfirmPending(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_pending"`)
}

// TestReservationRejectPending_Success verifies the happy-path admin
// reject returns the CANCELLED_IN_TIME status.
func TestReservationRejectPending_Success(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRejecter{
		fn: func(_ context.Context, in reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error) {
			assert.Equal(t, 99, in.ReservationID())
			reservation := newCancelledReservation(99, domain.StatusCancelledInTime)
			return reservationuc.RejectPendingReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/reject", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.RejectPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"CANCELLED_IN_TIME"`)
}

// TestReservationRejectPending_NotFound verifies ErrNotFound maps to
// 404.
func TestReservationRejectPending_NotFound(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRejecter{
		fn: func(context.Context, reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error) {
			return reservationuc.RejectPendingReservationOutput{}, reservationuc.ErrNotFound
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/reject", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.RejectPending(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationConfirmPending_NotFound verifies errNotFound maps to
// 404.
func TestReservationConfirmPending_NotFound(t *testing.T) {
	t.Parallel()
	stub := &stubReservationConfirmer{
		fn: func(context.Context, reservationuc.ConfirmPendingReservationInput) (reservationuc.ConfirmPendingReservationOutput, error) {
			return reservationuc.ConfirmPendingReservationOutput{}, reservationuc.ErrNotFound
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub, nil)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/confirm", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.ConfirmPending(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationRejectPending_NotPending verifies ErrNotPending maps to
// 409 not_pending.
func TestReservationRejectPending_NotPending(t *testing.T) {
	t.Parallel()
	stub := &stubReservationRejecter{
		fn: func(context.Context, reservationuc.RejectPendingReservationInput) (reservationuc.RejectPendingReservationOutput, error) {
			return reservationuc.RejectPendingReservationOutput{}, reservationuc.ErrNotPending
		},
	}
	h := newReservationHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, stub)
	c, w := setupAuthCtx(http.MethodPost, "/api/v1/users/1/reservations/99/reject", "", withUserID(1))
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.RejectPending(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_pending"`)
}
