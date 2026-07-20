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

func newReservationHandler(
	reg ReservationRegisterer,
	cancel ReservationCanceler,
	get ReservationGetter,
	listByUser ReservationListerByUser,
	listUpcoming ReservationListerUpcomingByUser,
	listByDog ReservationListerByDog,
	listByPass ReservationListerByPass,
	listByActivity ReservationListerByActivity,
) *ReservationHandler {
	return NewReservationHandler(reg, cancel, get, listByUser, listUpcoming, listByDog, listByPass, listByActivity)
}

func newReservationHandlerReg(reg ReservationRegisterer) *ReservationHandler {
	return newReservationHandler(reg, nil, nil, nil, nil, nil, nil, nil)
}

func newReservationHandlerCancel(cancel ReservationCanceler) *ReservationHandler {
	return newReservationHandler(nil, cancel, nil, nil, nil, nil, nil, nil)
}

// newCancelledReservation builds a domain.Reservation in the given
// terminal status. Used by handler tests that need a
// *domain.Reservation to wrap in a CancelReservationOutput.
func newCancelledReservation(id int, status domain.ReservationStatus) *domain.Reservation {
	reservation, err := domain.NewReservationWithStatus(id, 10, 20, 30, status, time.Now())
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
	stub := &stubReservationRegisterer{
		fn: func(_ context.Context, in reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			assert.Equal(t, 1, in.UserID, "user_id comes from the path, not the body")
			assert.Equal(t, 42, in.ActivityID)
			assert.Equal(t, 7, in.DogID)
			assert.Equal(t, 3, in.PassID)
			return reservationuc.RegisterReservationOutput{ID: 99}, nil
		},
	}
	h := newReservationHandlerReg(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
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
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			t.Fatal("use case should not be called on invalid body")
			return reservationuc.RegisterReservationOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", `not json`)
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"invalid_request"`)
}

// TestReservationRegister_MissingFields verifies that Gin's binding
// rejects a body with zero-valued ids.
func TestReservationRegister_MissingFields(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			t.Fatal("use case should not be called on missing fields")
			return reservationuc.RegisterReservationOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", `{"activity_id":0,"dog_id":0,"pass_id":0}`)
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"invalid_request"`)
}

// TestReservationRegister_UseCaseValidation verifies that a
// *ValidationError from the use case maps to 400 with the field
// name (here, defending against a future validator that might run
// in the use case rather than the handler).
func TestReservationRegister_UseCaseValidation(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, &reservationuc.ValidationError{Field: "activity_id"}
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"activity_id"`)
}

// TestReservationRegister_ActivityInPast verifies ErrActivityInPast
// maps to 400 activity_in_past.
func TestReservationRegister_ActivityInPast(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrActivityInPast
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_in_past"`)
}

// TestReservationRegister_ActivityFull verifies ErrActivityFull
// maps to 409 activity_full.
func TestReservationRegister_ActivityFull(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrActivityFull
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_full"`)
}

// TestReservationRegister_PassExhausted verifies ErrPassExhausted
// maps to 400 pass_exhausted.
func TestReservationRegister_PassExhausted(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrPassExhausted
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"pass_exhausted"`)
}

// TestReservationRegister_PassExpired verifies ErrPassExpired maps
// to 400 pass_expired.
func TestReservationRegister_PassExpired(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrPassExpired
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"pass_expired"`)
}

// TestReservationRegister_DuplicateReservation verifies
// ErrDuplicateReservationForDog maps to 409.
func TestReservationRegister_DuplicateReservation(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, reservationuc.ErrDuplicateReservationForDog
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"duplicate_reservation"`)
}

// TestReservationRegister_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationRegister_InternalError(t *testing.T) {
	h := newReservationHandlerReg(&stubReservationRegisterer{
		fn: func(context.Context, reservationuc.RegisterReservationInput) (reservationuc.RegisterReservationOutput, error) {
			return reservationuc.RegisterReservationOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations", validRegisterReservationBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
}

// ============================================================================
// Cancel endpoint tests
// ============================================================================

// TestReservationCancel_SuccessInTime verifies the happy-path POST
// returns 200 with the new CANCELLED_IN_TIME status.
func TestReservationCancel_SuccessInTime(t *testing.T) {
	reservation := newCancelledReservation(99, domain.StatusCancelledInTime)
	stub := &stubReservationCanceler{
		fn: func(_ context.Context, in reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			assert.Equal(t, 1, in.UserID, "user_id comes from the path, not the body")
			assert.Equal(t, 99, in.ReservationID)
			return reservationuc.CancelReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerCancel(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
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
	reservation := newCancelledReservation(99, domain.StatusCancelledLate)
	stub := &stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{Reservation: reservation}, nil
		},
	}
	h := newReservationHandlerCancel(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
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
			c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/"+tt.pathID+"/cancel", "")
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
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrAlreadyCancelled
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"already_cancelled"`)
}

// TestReservationCancel_ActivityInPast verifies ErrActivityInPast
// maps to 400.
func TestReservationCancel_ActivityInPast(t *testing.T) {
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrActivityInPast
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"activity_in_past"`)
}

// TestReservationCancel_InvalidReservationID_NotFound verifies
// ErrInvalidReservation maps to 404.
func TestReservationCancel_InvalidReservationID_NotFound(t *testing.T) {
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"not_found"`)
}

// TestReservationCancel_InternalError verifies that an unknown
// error maps to 500 internal.
func TestReservationCancel_InternalError(t *testing.T) {
	h := newReservationHandlerCancel(&stubReservationCanceler{
		fn: func(context.Context, reservationuc.CancelReservationInput) (reservationuc.CancelReservationOutput, error) {
			return reservationuc.CancelReservationOutput{}, errors.New("db connection lost")
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/reservations/99/cancel", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}

	h.Cancel(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"internal"`)
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
	return newReservationHandler(nil, nil, get, nil, nil, nil, nil, nil)
}

func newReservationHandlerListByUser(l ReservationListerByUser) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, l, nil, nil, nil, nil)
}

func newReservationHandlerListUpcoming(l ReservationListerUpcomingByUser) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, l, nil, nil, nil)
}

func newReservationHandlerListByDog(l ReservationListerByDog) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, l, nil, nil)
}

func newReservationHandlerListByPass(l ReservationListerByPass) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, l, nil)
}

func newReservationHandlerListByActivity(l ReservationListerByActivity) *ReservationHandler {
	return newReservationHandler(nil, nil, nil, nil, nil, nil, nil, l)
}

func sampleViewOwnedBy(userID int) *domain.ReservationView {
	now := time.Now()
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
	activity := domain.MustNewActivity(activityID, activityName, activityLocation,
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
	stub := &stubReservationListerByUser{
		fn: func(_ context.Context, in reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			assert.Equal(t, 1, in.UserID)
			assert.Equal(t, 50, in.Limit)
			return reservationuc.ListByUserReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByUser(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listReservationsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Count)
	assert.Equal(t, 1, len(body.Reservations))
	assert.Equal(t, "Luna", body.Reservations[0].DogName)
	assert.Equal(t, "Paseo Río", body.Reservations[0].ActivityName)
}

func TestListByUser_InvalidUserID(t *testing.T) {
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
	h := newReservationHandlerListByUser(&stubReservationListerByUser{
		fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			t.Fatal("use case should not be called on invalid filter")
			return reservationuc.ListByUserReservationsOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations?status=BOGUS", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"status"`)
}

func TestListByUser_InvalidTimeFilter(t *testing.T) {
	h := newReservationHandlerListByUser(&stubReservationListerByUser{
		fn: func(context.Context, reservationuc.ListByUserReservationsInput) (reservationuc.ListByUserReservationsOutput, error) {
			t.Fatal("use case should not be called on invalid time")
			return reservationuc.ListByUserReservationsOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations?from=not-a-date", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListUpcomingByUser_Success(t *testing.T) {
	stub := &stubReservationListerUpcomingByUser{
		fn: func(_ context.Context, in reservationuc.ListUpcomingByUserInput) (reservationuc.ListUpcomingByUserOutput, error) {
			assert.Equal(t, 1, in.UserID)
			return reservationuc.ListUpcomingByUserOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListUpcoming(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations/upcoming", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListUpcomingByUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listReservationsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Count)
}

func TestListUpcomingByUser_InvalidUserID(t *testing.T) {
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

func TestGetByID_Success(t *testing.T) {
	stub := &stubReservationGetter{
		fn: func(_ context.Context, in reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			assert.Equal(t, 1, in.UserID)
			assert.Equal(t, 99, in.ReservationID)
			return reservationuc.GetReservationOutput{View: sampleViewOwnedBy(1)}, nil
		},
	}
	h := newReservationHandlerGet(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body reservationViewResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.Reservation.ID)
	assert.Equal(t, "Luna", body.Reservation.DogName)
}

func TestGetByID_NotFound(t *testing.T) {
	h := newReservationHandlerGet(&stubReservationGetter{
		fn: func(context.Context, reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			return reservationuc.GetReservationOutput{}, reservationuc.ErrInvalidReservation
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetByID_NotOwned(t *testing.T) {
	h := newReservationHandlerGet(&stubReservationGetter{
		fn: func(context.Context, reservationuc.GetReservationInput) (reservationuc.GetReservationOutput, error) {
			return reservationuc.GetReservationOutput{}, reservationuc.ErrReservationNotOwned
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/reservations/99", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}, {Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code, "not owned must map to 404 (no leak)")
}

func TestListByDog_Success(t *testing.T) {
	stub := &stubReservationListerByDog{
		fn: func(_ context.Context, in reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error) {
			assert.Equal(t, 20, in.DogID)
			return reservationuc.ListByDogReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByDog(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/20/reservations", "")
	c.Params = gin.Params{{Key: "dog_id", Value: "20"}}
	h.ListByDog(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByDog_InvalidDogID(t *testing.T) {
	h := newReservationHandlerListByDog(&stubReservationListerByDog{
		fn: func(context.Context, reservationuc.ListByDogReservationsInput) (reservationuc.ListByDogReservationsOutput, error) {
			t.Fatal("use case should not be called")
			return reservationuc.ListByDogReservationsOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/0/reservations", "")
	c.Params = gin.Params{{Key: "dog_id", Value: "0"}}
	h.ListByDog(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByPass_Success(t *testing.T) {
	stub := &stubReservationListerByPass{
		fn: func(_ context.Context, in reservationuc.ListByPassReservationsInput) (reservationuc.ListByPassReservationsOutput, error) {
			assert.Equal(t, 30, in.PassID)
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
	stub := &stubReservationListerByActivity{
		fn: func(_ context.Context, in reservationuc.ListByActivityReservationsInput) (reservationuc.ListByActivityReservationsOutput, error) {
			assert.Equal(t, 10, in.ActivityID)
			return reservationuc.ListByActivityReservationsOutput{Views: []*domain.ReservationView{sampleViewOwnedBy(1)}}, nil
		},
	}
	h := newReservationHandlerListByActivity(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/10/reservations", "")
	c.Params = gin.Params{{Key: "id", Value: "10"}}
	h.ListByActivity(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
