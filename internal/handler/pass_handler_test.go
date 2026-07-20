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
	passuc "dogpaw/internal/usecase/pass"
)

type stubPassRegisterer struct {
	fn func(ctx context.Context, in passuc.RegisterPassInput) (passuc.RegisterPassOutput, error)
}

func (s *stubPassRegisterer) Execute(ctx context.Context, in passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
	return s.fn(ctx, in)
}

type stubPassModifier struct {
	fn func(ctx context.Context, in passuc.ModifyPassInput) (passuc.ModifyPassOutput, error)
}

func (s *stubPassModifier) Execute(ctx context.Context, in passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
	return s.fn(ctx, in)
}

type stubPassGetter struct {
	fn func(ctx context.Context, in passuc.GetPassInput) (passuc.GetPassOutput, error)
}

func (s *stubPassGetter) Execute(ctx context.Context, in passuc.GetPassInput) (passuc.GetPassOutput, error) {
	return s.fn(ctx, in)
}

type stubPassLister struct {
	fn func(ctx context.Context, in passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error)
}

func (s *stubPassLister) Execute(ctx context.Context, in passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error) {
	return s.fn(ctx, in)
}

type stubPassByUserLister struct {
	fn func(ctx context.Context, in passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error)
}

func (s *stubPassByUserLister) Execute(ctx context.Context, in passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
	return s.fn(ctx, in)
}

func newPassHandler(
	reg PassRegisterer,
	mod PassModifier,
	getter PassGetter,
	lister PassLister,
	byUserLister PassByUserLister,
) *PassHandler {
	return NewPassHandler(reg, mod, getter, lister, byUserLister)
}

func newPassHandlerReg(reg PassRegisterer) *PassHandler {
	return newPassHandler(reg, nil, nil, nil, nil)
}

func newPassHandlerMod(mod PassModifier) *PassHandler {
	return newPassHandler(nil, mod, nil, nil, nil)
}

func newPassHandlerGet(getter PassGetter) *PassHandler {
	return newPassHandler(nil, nil, getter, nil, nil)
}

func newPassHandlerLst(lister PassLister) *PassHandler {
	return newPassHandler(nil, nil, nil, lister, nil)
}

func newPassHandlerByUser(byUserLister PassByUserLister) *PassHandler {
	return newPassHandler(nil, nil, nil, nil, byUserLister)
}

func validRegisterPassBody() string {
	return `{"num_of_sessions":10,"price":12000,"pass_type":"GENERICO"}`
}

func TestPassRegister_Success(t *testing.T) {
	stub := &stubPassRegisterer{fn: func(ctx context.Context, in passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		// user_id comes from the path param, not the body.
		assert.Equal(t, 1, in.UserID)
		assert.Equal(t, 10, in.NumOfSessions)
		assert.Equal(t, 12000, in.Price)
		assert.Equal(t, domain.PassGeneric, in.PassType)
		return passuc.RegisterPassOutput{ID: 42}, nil
	}}
	h := newPassHandlerReg(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes", validRegisterPassBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/passes/42", w.Header().Get("Location"))
	var body registerPassResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
}

func TestPassRegister_InvalidUserID_PathParam(t *testing.T) {
	tests := []struct {
		name      string
		pathValue string
	}{
		{"non_integer", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
				t.Fatal("use case should not be invoked for invalid user_id")
				return passuc.RegisterPassOutput{}, nil
			}})
			c, w := setupCtx(http.MethodPost, "/api/v1/users/"+tt.pathValue+"/passes", validRegisterPassBody())
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathValue}}
			h.Register(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

func TestPassRegister_InvalidJSON(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		t.Fatal("use case should not be invoked for invalid JSON")
		return passuc.RegisterPassOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes", "not json")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestPassRegister_BindingValidation(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		t.Fatal("use case should not be invoked when binding fails")
		return passuc.RegisterPassOutput{}, nil
	}})
	// Missing required fields: num_of_sessions, price, pass_type.
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes", `{"num_of_sessions":10}`)
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPassRegister_BindingInvalidPassType(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		t.Fatal("use case should not be invoked for invalid pass_type")
		return passuc.RegisterPassOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes",
		`{"num_of_sessions":10,"price":1000,"pass_type":"BOGUS"}`)
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPassRegister_UseCaseValidation(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		return passuc.RegisterPassOutput{}, &passuc.ValidationError{Field: "pass_type"}
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes", validRegisterPassBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"pass_type"`)
}

func TestPassRegister_InvalidUserID_FromRepo(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		return passuc.RegisterPassOutput{}, passuc.ErrInvalidUserID
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/9999/passes", validRegisterPassBody())
	c.Params = gin.Params{{Key: "user_id", Value: "9999"}}
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"invalid_user_id"`)
}

func TestPassRegister_InternalError(t *testing.T) {
	h := newPassHandlerReg(&stubPassRegisterer{fn: func(context.Context, passuc.RegisterPassInput) (passuc.RegisterPassOutput, error) {
		return passuc.RegisterPassOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/users/1/passes", validRegisterPassBody())
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.Register(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// newTestPassForHandler creates a domain.Pass used in modify test
// stubs. Helper to keep the test bodies focused.
func newTestPassForHandler(id int) *domain.Pass {
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	return domain.MustNewPass(id, 10, 10, 100, domain.PassGeneric, 1, now, now, nil)
}

func TestPassModify_Success_AllFields(t *testing.T) {
	updated := newTestPassForHandler(1)
	updatedPrice := 15000
	updatedType := domain.PassSpecial
	updatedExpiry := time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC)
	updated.ApplyPatch(domain.PassPatch{Price: &updatedPrice, PassType: &updatedType, ExpiresAt: &updatedExpiry})
	stub := &stubPassModifier{fn: func(ctx context.Context, in passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		assert.Equal(t, 1, in.ID)
		requireNotNilHandler(t, in.Patch.Price)
		assert.Equal(t, 15000, *in.Patch.Price)
		return passuc.ModifyPassOutput{Pass: updated}, nil
	}}
	h := newPassHandlerMod(stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1",
		`{"price":15000,"pass_type":"ESPECIFICO","expires_at":"2027-12-31T23:59:59Z"}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.Modify(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body passResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.ID)
	assert.Equal(t, 15000, body.Price)
	assert.Equal(t, "ESPECIFICO", body.PassType)
}

func TestPassModify_Success_EmptyPatch(t *testing.T) {
	original := newTestPassForHandler(1)
	stub := &stubPassModifier{fn: func(ctx context.Context, in passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		return passuc.ModifyPassOutput{Pass: original}, nil
	}}
	h := newPassHandlerMod(stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1", `{}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body passResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.ID)
}

func TestPassModify_InvalidID(t *testing.T) {
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_integer", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
				t.Fatal("use case should not be invoked for invalid id")
				return passuc.ModifyPassOutput{}, nil
			}})
			c, w := setupCtx(http.MethodPatch, "/api/v1/passes/"+tt.pathID, `{}`)
			c.Params = gin.Params{{Key: "id", Value: tt.pathID}}
			h.Modify(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"id"`)
		})
	}
}

func TestPassModify_InvalidJSON(t *testing.T) {
	h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		t.Fatal("use case should not be invoked for invalid JSON")
		return passuc.ModifyPassOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1", "not json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPassModify_BindingInvalidPassType(t *testing.T) {
	h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		t.Fatal("use case should not be invoked for invalid pass_type")
		return passuc.ModifyPassOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1", `{"pass_type":"BOGUS"}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPassModify_UseCaseValidation(t *testing.T) {
	// expires_at = Go zero time ("0001-01-01T00:00:00Z") passes
	// binding (no tag) but fails the use case's ApplyPatch which
	// rejects zero values.
	h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		return passuc.ModifyPassOutput{}, &passuc.ValidationError{Field: "expires_at"}
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1",
		`{"expires_at":"0001-01-01T00:00:00Z"}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"expires_at"`)
}

func TestPassModify_NotFound(t *testing.T) {
	h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		return passuc.ModifyPassOutput{}, passuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/99", `{"price":1000}`)
	c.Params = gin.Params{{Key: "id", Value: "99"}}
	h.Modify(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPassModify_InternalError(t *testing.T) {
	h := newPassHandlerMod(&stubPassModifier{fn: func(context.Context, passuc.ModifyPassInput) (passuc.ModifyPassOutput, error) {
		return passuc.ModifyPassOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/passes/1", `{"price":1000}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.Modify(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// requireNotNilHandler is a tiny helper that fails the test if v is
// nil. Used in assertion bodies to keep imports lean.
func requireNotNilHandler(t *testing.T, v any) {
	t.Helper()
	if v == nil {
		t.Fatal("expected non-nil")
	}
}

func TestPassGetByID_Success(t *testing.T) {
	want := newTestPassForHandler(7)
	stub := &stubPassGetter{fn: func(ctx context.Context, in passuc.GetPassInput) (passuc.GetPassOutput, error) {
		assert.Equal(t, 7, in.ID)
		return passuc.GetPassOutput{Pass: want}, nil
	}}
	h := newPassHandlerGet(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/passes/7", "")
	c.Params = gin.Params{{Key: "id", Value: "7"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body passResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 7, body.ID)
	assert.Equal(t, 100, body.Price)
	assert.Equal(t, "GENERICO", body.PassType)
}

func TestPassGetByID_InvalidID(t *testing.T) {
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_integer", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPassHandlerGet(&stubPassGetter{fn: func(context.Context, passuc.GetPassInput) (passuc.GetPassOutput, error) {
				t.Fatal("use case should not be invoked for invalid id")
				return passuc.GetPassOutput{}, nil
			}})
			c, w := setupCtx(http.MethodGet, "/api/v1/passes/"+tt.pathID, "")
			c.Params = gin.Params{{Key: "id", Value: tt.pathID}}
			h.GetByID(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"id"`)
		})
	}
}

func TestPassGetByID_NotFound(t *testing.T) {
	h := newPassHandlerGet(&stubPassGetter{fn: func(context.Context, passuc.GetPassInput) (passuc.GetPassOutput, error) {
		return passuc.GetPassOutput{}, passuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/passes/99", "")
	c.Params = gin.Params{{Key: "id", Value: "99"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPassGetByID_InternalError(t *testing.T) {
	h := newPassHandlerGet(&stubPassGetter{fn: func(context.Context, passuc.GetPassInput) (passuc.GetPassOutput, error) {
		return passuc.GetPassOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/passes/1", "")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPassList_Success(t *testing.T) {
	passes := []*domain.Pass{
		newTestPassForHandler(1),
		newTestPassForHandler(2),
	}
	stub := &stubPassLister{fn: func(ctx context.Context, in passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error) {
		assert.Equal(t, 0, in.Limit)
		assert.Equal(t, 0, in.Offset)
		return passuc.ListAllPassesOutput{Passes: passes}, nil
	}}
	h := newPassHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/passes", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listPassesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Passes, 2)
	// Response echoes the *normalized* limit/offset (defaults 50/0).
	assert.Equal(t, 50, body.Limit)
	assert.Equal(t, 0, body.Offset)
	assert.Equal(t, 2, body.Count)
}

func TestPassList_Empty(t *testing.T) {
	h := newPassHandlerLst(&stubPassLister{fn: func(context.Context, passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error) {
		return passuc.ListAllPassesOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/passes", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"passes":[]`)
}

func TestPassList_PaginationPassesThrough(t *testing.T) {
	stub := &stubPassLister{fn: func(ctx context.Context, in passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error) {
		assert.Equal(t, 25, in.Limit)
		assert.Equal(t, 10, in.Offset)
		return passuc.ListAllPassesOutput{}, nil
	}}
	h := newPassHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/passes?limit=25&offset=10", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listPassesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 25, body.Limit)
	assert.Equal(t, 10, body.Offset)
}

func TestPassList_InternalError(t *testing.T) {
	h := newPassHandlerLst(&stubPassLister{fn: func(context.Context, passuc.ListAllPassesInput) (passuc.ListAllPassesOutput, error) {
		return passuc.ListAllPassesOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/passes", "")
	h.List(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPassListByUser_Success(t *testing.T) {
	passes := []*domain.Pass{newTestPassForHandler(1)}
	stub := &stubPassByUserLister{fn: func(ctx context.Context, in passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
		assert.Equal(t, 1, in.UserID)
		return passuc.ListByUserPassesOutput{Passes: passes}, nil
	}}
	h := newPassHandlerByUser(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/passes", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.ListByUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listPassesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Passes, 1)
}

func TestPassListByUser_Empty(t *testing.T) {
	h := newPassHandlerByUser(&stubPassByUserLister{fn: func(context.Context, passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
		return passuc.ListByUserPassesOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/9999/passes", "")
	c.Params = gin.Params{{Key: "user_id", Value: "9999"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"passes":[]`)
}

func TestPassListByUser_InvalidUserID(t *testing.T) {
	tests := []struct {
		name   string
		pathID string
	}{
		{"non_integer", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPassHandlerByUser(&stubPassByUserLister{fn: func(context.Context, passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
				t.Fatal("use case should not be invoked for invalid user_id")
				return passuc.ListByUserPassesOutput{}, nil
			}})
			c, w := setupCtx(http.MethodGet, "/api/v1/users/"+tt.pathID+"/passes", "")
			c.Params = gin.Params{{Key: "user_id", Value: tt.pathID}}
			h.ListByUser(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"field":"user_id"`)
		})
	}
}

func TestPassListByUser_UseCaseValidation(t *testing.T) {
	h := newPassHandlerByUser(&stubPassByUserLister{fn: func(context.Context, passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
		return passuc.ListByUserPassesOutput{}, &passuc.ValidationError{Field: "user_id"}
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/passes", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPassListByUser_InternalError(t *testing.T) {
	h := newPassHandlerByUser(&stubPassByUserLister{fn: func(context.Context, passuc.ListByUserPassesInput) (passuc.ListByUserPassesOutput, error) {
		return passuc.ListByUserPassesOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/users/1/passes", "")
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}
	h.ListByUser(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
