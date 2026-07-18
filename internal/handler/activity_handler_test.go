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
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	activityuc "dogpaw/internal/usecase/activity"
)

type stubActivityRegisterer struct {
	fn func(ctx context.Context, in activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error)
}

func (s *stubActivityRegisterer) Execute(ctx context.Context, in activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
	return s.fn(ctx, in)
}

type stubActivityGetter struct {
	fn func(ctx context.Context, in activityuc.GetActivityInput) (activityuc.GetActivityOutput, error)
}

func (s *stubActivityGetter) Execute(ctx context.Context, in activityuc.GetActivityInput) (activityuc.GetActivityOutput, error) {
	return s.fn(ctx, in)
}

type stubActivityModifier struct {
	fn func(ctx context.Context, in activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error)
}

func (s *stubActivityModifier) Execute(ctx context.Context, in activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
	return s.fn(ctx, in)
}

type stubActivityLister struct {
	fn func(ctx context.Context, in activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error)
}

func (s *stubActivityLister) Execute(ctx context.Context, in activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error) {
	return s.fn(ctx, in)
}

type stubActivityUpcomingLister struct {
	fn func(ctx context.Context, in activityuc.ListUpcomingActivitiesInput) (activityuc.ListUpcomingActivitiesOutput, error)
}

func (s *stubActivityUpcomingLister) Execute(ctx context.Context, in activityuc.ListUpcomingActivitiesInput) (activityuc.ListUpcomingActivitiesOutput, error) {
	return s.fn(ctx, in)
}

func newActivityHandler(
	reg ActivityRegisterer,
	get ActivityGetter,
	mod ActivityModifier,
	lst ActivityLister,
	upcoming ActivityUpcomingLister,
) *ActivityHandler {
	return NewActivityHandler(reg, get, mod, lst, upcoming)
}

func newActivityHandlerReg(reg ActivityRegisterer) *ActivityHandler {
	return newActivityHandler(reg, nil, nil, nil, nil)
}

func newActivityHandlerGet(get ActivityGetter) *ActivityHandler {
	return newActivityHandler(nil, get, nil, nil, nil)
}

func newActivityHandlerMod(mod ActivityModifier) *ActivityHandler {
	return newActivityHandler(nil, nil, mod, nil, nil)
}

func newActivityHandlerLst(lst ActivityLister) *ActivityHandler {
	return newActivityHandler(nil, nil, nil, lst, nil)
}

func newActivityHandlerUp(up ActivityUpcomingLister) *ActivityHandler {
	return newActivityHandler(nil, nil, nil, nil, up)
}

func validRegisterActivityBody() string {
	return `{"name":"Paseo Río","location":"Parking Central","activity_type":"ROUTE","max_capacity":8,"duration_in_hours":2,"date":"2026-08-01T10:00:00Z"}`
}

func newTestActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1,
		mustParseActivityTime("2026-08-01T10:00:00Z"))
}

func mustParseActivityTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// TestActivityRegister_Success verifies the happy-path POST creates the
// resource and returns 201 with the new id and a Location header.
func TestActivityRegister_Success(t *testing.T) {
	stub := &stubActivityRegisterer{fn: func(ctx context.Context, in activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
		return activityuc.RegisterActivityOutput{ID: 42}, nil
	}}
	h := newActivityHandlerReg(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/activities", validRegisterActivityBody())

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/activities/42", w.Header().Get("Location"))
	var body registerActivityResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
}

// TestActivityRegister_InvalidJSON verifies that non-JSON bodies are
// rejected with 400 invalid_request before any use case is called.
func TestActivityRegister_InvalidJSON(t *testing.T) {
	h := newActivityHandlerReg(&stubActivityRegisterer{fn: func(context.Context, activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
		t.Fatal("use case should not be invoked for invalid JSON")
		return activityuc.RegisterActivityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/activities", "not json")
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

// TestActivityRegister_BindingValidation verifies that Gin's binding
// rules (required, oneof, gt) reject incomplete payloads with 400.
func TestActivityRegister_BindingValidation(t *testing.T) {
	h := newActivityHandlerReg(&stubActivityRegisterer{fn: func(context.Context, activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
		t.Fatal("use case should not be invoked when binding fails")
		return activityuc.RegisterActivityOutput{}, nil
	}})
	// Missing required fields (name, activity_type, max_capacity, etc.).
	c, w := setupCtx(http.MethodPost, "/api/v1/activities", `{"name":"Paseo"}`)
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestActivityRegister_UseCaseValidation verifies that a *ValidationError
// returned by the use case is mapped to 400 with the field name.
func TestActivityRegister_UseCaseValidation(t *testing.T) {
	h := newActivityHandlerReg(&stubActivityRegisterer{fn: func(context.Context, activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
		return activityuc.RegisterActivityOutput{}, &activityuc.ValidationError{Field: "max_capacity"}
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/activities", validRegisterActivityBody())
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"max_capacity"`)
}

// TestActivityRegister_InternalError verifies that unexpected errors
// fall through to a generic 500.
func TestActivityRegister_InternalError(t *testing.T) {
	h := newActivityHandlerReg(&stubActivityRegisterer{fn: func(context.Context, activityuc.RegisterActivityInput) (activityuc.RegisterActivityOutput, error) {
		return activityuc.RegisterActivityOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/activities", validRegisterActivityBody())
	h.Register(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestActivityGetByID_Success verifies the happy-path GET returns 200
// and the full activity payload.
func TestActivityGetByID_Success(t *testing.T) {
	want := newTestActivity(7)
	stub := &stubActivityGetter{fn: func(ctx context.Context, in activityuc.GetActivityInput) (activityuc.GetActivityOutput, error) {
		assert.Equal(t, 7, in.ID)
		return activityuc.GetActivityOutput{Activity: want}, nil
	}}
	h := newActivityHandlerGet(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/7", "")
	c.Params = gin.Params{{Key: "id", Value: "7"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body activityResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 7, body.ID)
	assert.Equal(t, "Paseo", body.Name)
	assert.Equal(t, "ROUTE", body.ActivityType)
}

// TestActivityGetByID_InvalidID verifies that non-integer or non-positive
// ids are rejected with 400 before the use case is called.
func TestActivityGetByID_InvalidID(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		queryParam string
	}{
		{"non_integer", "abc", "abc"},
		{"zero", "0", "0"},
		{"negative", "-5", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newActivityHandlerGet(&stubActivityGetter{fn: func(context.Context, activityuc.GetActivityInput) (activityuc.GetActivityOutput, error) {
				t.Fatal("use case should not be invoked for invalid id")
				return activityuc.GetActivityOutput{}, nil
			}})
			c, w := setupCtx(http.MethodGet, "/api/v1/activities/"+tt.pathID, "")
			c.Params = gin.Params{{Key: "id", Value: tt.queryParam}}
			h.GetByID(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestActivityGetByID_NotFound verifies that use-case ErrNotFound is
// mapped to 404.
func TestActivityGetByID_NotFound(t *testing.T) {
	h := newActivityHandlerGet(&stubActivityGetter{fn: func(context.Context, activityuc.GetActivityInput) (activityuc.GetActivityOutput, error) {
		return activityuc.GetActivityOutput{}, activityuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/99", "")
	c.Params = gin.Params{{Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestActivityList_Success verifies the happy-path GET returns 200 with
// the full activity array, plus the normalized limit/offset echoed back.
func TestActivityList_Success(t *testing.T) {
	activities := []*domain.Activity{
		domain.MustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, mustParseActivityTime("2026-08-01T10:00:00Z")),
		domain.MustNewActivity(2, "b", "l", domain.TypeRoute, 5, 1, mustParseActivityTime("2026-08-02T10:00:00Z")),
	}
	stub := &stubActivityLister{fn: func(ctx context.Context, in activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error) {
		// The handler passes raw query string values (0 when absent);
		// the use case normalizes them before calling the repo. We
		// assert the raw values here.
		assert.Equal(t, 0, in.Limit)
		assert.Equal(t, 0, in.Offset)
		return activityuc.ListAllActivitiesOutput{Activities: activities}, nil
	}}
	h := newActivityHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listActivitiesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Activities, 2)
	// The response echoes the *normalized* limit/offset.
	assert.Equal(t, 50, body.Limit)
	assert.Equal(t, 0, body.Offset)
	assert.Equal(t, 2, body.Count)
}

// TestActivityList_PaginationPassesThrough verifies that the
// limit/offset query parameters are passed to the use case.
func TestActivityList_PaginationPassesThrough(t *testing.T) {
	stub := &stubActivityLister{fn: func(ctx context.Context, in activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error) {
		assert.Equal(t, 25, in.Limit)
		assert.Equal(t, 10, in.Offset)
		return activityuc.ListAllActivitiesOutput{}, nil
	}}
	h := newActivityHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities?limit=25&offset=10", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestActivityList_Empty verifies that the list endpoint returns an
// empty array (not null) when no activities are found.
func TestActivityList_Empty(t *testing.T) {
	h := newActivityHandlerLst(&stubActivityLister{fn: func(context.Context, activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error) {
		return activityuc.ListAllActivitiesOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/activities", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	// Verify the body has "activities":[] not "activities":null.
	assert.Contains(t, w.Body.String(), `"activities":[]`)
}

// TestActivityList_InternalError verifies that an unexpected repo
// error is mapped to 500.
func TestActivityList_InternalError(t *testing.T) {
	h := newActivityHandlerLst(&stubActivityLister{fn: func(context.Context, activityuc.ListAllActivitiesInput) (activityuc.ListAllActivitiesOutput, error) {
		return activityuc.ListAllActivitiesOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/activities", "")
	h.List(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestActivityListUpcoming_Success verifies the upcoming endpoint
// delegates to the dedicated use case.
func TestActivityListUpcoming_Success(t *testing.T) {
	future := domain.MustNewActivity(1, "a", "l", domain.TypeRoute, 5, 1, mustParseActivityTime("2030-01-01T10:00:00Z"))
	stub := &stubActivityUpcomingLister{fn: func(ctx context.Context, in activityuc.ListUpcomingActivitiesInput) (activityuc.ListUpcomingActivitiesOutput, error) {
		return activityuc.ListUpcomingActivitiesOutput{Activities: []*domain.Activity{future}}, nil
	}}
	h := newActivityHandlerUp(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/upcoming", "")
	h.ListUpcoming(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listActivitiesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Activities, 1)
}

// TestActivityListUpcoming_Empty verifies the upcoming endpoint
// returns an empty array when nothing is scheduled.
func TestActivityListUpcoming_Empty(t *testing.T) {
	h := newActivityHandlerUp(&stubActivityUpcomingLister{fn: func(context.Context, activityuc.ListUpcomingActivitiesInput) (activityuc.ListUpcomingActivitiesOutput, error) {
		return activityuc.ListUpcomingActivitiesOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/activities/upcoming", "")
	h.ListUpcoming(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"activities":[]`)
}

// TestActivityModify_Success verifies the PATCH endpoint returns 200
// with the updated activity.
func TestActivityModify_Success(t *testing.T) {
	updated := domain.MustNewActivity(1, "Paseo Largo", "Central", domain.TypeRoute, 12, 2,
		mustParseActivityTime("2026-08-01T10:00:00Z"))
	stub := &stubActivityModifier{fn: func(ctx context.Context, in activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
		assert.Equal(t, 1, in.ID)
		require.NotNil(t, in.Patch.Name)
		assert.Equal(t, "Paseo Largo", *in.Patch.Name)
		require.NotNil(t, in.Patch.MaxCapacity)
		assert.Equal(t, 12, *in.Patch.MaxCapacity)
		return activityuc.ModifyActivityOutput{Activity: updated}, nil
	}}
	h := newActivityHandlerMod(stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/activities/1",
		`{"name":"Paseo Largo","max_capacity":12}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.Modify(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body activityResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Paseo Largo", body.Name)
	assert.Equal(t, 12, body.MaxCapacity)
}

// TestActivityModify_InvalidID verifies that non-positive ids are
// rejected with 400.
func TestActivityModify_InvalidID(t *testing.T) {
	h := newActivityHandlerMod(&stubActivityModifier{fn: func(context.Context, activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
		t.Fatal("use case should not be invoked for invalid id")
		return activityuc.ModifyActivityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/activities/0", `{}`)
	c.Params = gin.Params{{Key: "id", Value: "0"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestActivityModify_InvalidJSON verifies that a malformed body is
// rejected with 400.
func TestActivityModify_InvalidJSON(t *testing.T) {
	h := newActivityHandlerMod(&stubActivityModifier{fn: func(context.Context, activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
		t.Fatal("use case should not be invoked for invalid JSON")
		return activityuc.ModifyActivityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/activities/1", "not json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestActivityModify_UseCaseValidation verifies that a
// *ValidationError is mapped to 400 with the field name.
func TestActivityModify_UseCaseValidation(t *testing.T) {
	h := newActivityHandlerMod(&stubActivityModifier{fn: func(context.Context, activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
		return activityuc.ModifyActivityOutput{}, &activityuc.ValidationError{Field: "activity_type"}
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/activities/1", `{"activity_type":"INVALID"}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"activity_type"`)
}

// TestActivityModify_NotFound verifies that use-case ErrNotFound is
// mapped to 404.
func TestActivityModify_NotFound(t *testing.T) {
	h := newActivityHandlerMod(&stubActivityModifier{fn: func(context.Context, activityuc.ModifyActivityInput) (activityuc.ModifyActivityOutput, error) {
		return activityuc.ModifyActivityOutput{}, activityuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/activities/99", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "99"}}
	h.Modify(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
