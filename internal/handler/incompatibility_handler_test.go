package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	incompatuc "dogpaw/internal/usecase/incompatibility"
)

type stubIncompatibilityRegisterer struct {
	fn func(ctx context.Context, in incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error)
}

func (s *stubIncompatibilityRegisterer) Execute(ctx context.Context, in incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityLister struct {
	fn func(ctx context.Context, in incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error)
}

func (s *stubIncompatibilityLister) Execute(ctx context.Context, in incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityGetter struct {
	fn func(ctx context.Context, in incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error)
}

func (s *stubIncompatibilityGetter) Execute(ctx context.Context, in incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityModifier struct {
	fn func(ctx context.Context, in incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error)
}

func (s *stubIncompatibilityModifier) Execute(ctx context.Context, in incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityDeleter struct {
	fn func(ctx context.Context, in incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error)
}

func (s *stubIncompatibilityDeleter) Execute(ctx context.Context, in incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

func newIncompatHandler(
	reg IncompatibilityRegisterer,
	lst IncompatibilityLister,
	get IncompatibilityGetter,
	mod IncompatibilityModifier,
	del IncompatibilityDeleter,
) *IncompatibilityHandler {
	return NewIncompatibilityHandler(reg, lst, get, mod, del)
}

func newIncompatHandlerReg(reg IncompatibilityRegisterer) *IncompatibilityHandler {
	return newIncompatHandler(reg, nil, nil, nil, nil)
}

func newIncompatHandlerLst(lst IncompatibilityLister) *IncompatibilityHandler {
	return newIncompatHandler(nil, lst, nil, nil, nil)
}

func newIncompatHandlerGet(get IncompatibilityGetter) *IncompatibilityHandler {
	return newIncompatHandler(nil, nil, get, nil, nil)
}

func newIncompatHandlerMod(mod IncompatibilityModifier) *IncompatibilityHandler {
	return newIncompatHandler(nil, nil, nil, mod, nil)
}

func newIncompatHandlerDel(del IncompatibilityDeleter) *IncompatibilityHandler {
	return newIncompatHandler(nil, nil, nil, nil, del)
}

func TestIncompatRegister_Success(t *testing.T) {
	stub := &stubIncompatibilityRegisterer{fn: func(ctx context.Context, in incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{ID: 5}, nil
	}}
	h := newIncompatHandlerReg(stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", `{"name":"Reacciona mal al transportin","level":"MEDIA"}`)

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body registerIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 5, body.ID)
}

func TestIncompatRegister_InvalidJSON(t *testing.T) {
	h := newIncompatHandlerReg(&stubIncompatibilityRegisterer{fn: func(context.Context, incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", "not json")
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatRegister_BindingValidation(t *testing.T) {
	h := newIncompatHandlerReg(&stubIncompatibilityRegisterer{fn: func(context.Context, incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", `{"name":"","level":"MEDIA"}`)
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatRegister_UseCaseValidation(t *testing.T) {
	h := newIncompatHandlerReg(&stubIncompatibilityRegisterer{fn: func(context.Context, incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{}, &incompatuc.ValidationError{Field: "level"}
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", `{"name":"x","level":"MEDIA"}`)
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatRegister_DuplicateName(t *testing.T) {
	h := newIncompatHandlerReg(&stubIncompatibilityRegisterer{fn: func(context.Context, incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{}, incompatuc.ErrDuplicateName
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", `{"name":"x","level":"MEDIA"}`)
	h.Register(c)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestIncompatRegister_InternalError(t *testing.T) {
	h := newIncompatHandlerReg(&stubIncompatibilityRegisterer{fn: func(context.Context, incompatuc.RegisterIncompatibilityInput) (incompatuc.RegisterIncompatibilityOutput, error) {
		return incompatuc.RegisterIncompatibilityOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPost, "/api/v1/incompatibilities", `{"name":"x","level":"MEDIA"}`)
	h.Register(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIncompatList_Success(t *testing.T) {
	incompats := []*domain.Incompatibility{
		domain.MustNewIncompatibility(1, "A", domain.IncompatibilityLevelBaja),
		domain.MustNewIncompatibility(2, "B", domain.IncompatibilityLevelMedia),
	}
	stub := &stubIncompatibilityLister{fn: func(ctx context.Context, in incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error) {
		assert.Nil(t, in.Level)
		return incompatuc.ListIncompatibilitiesOutput{Incompatibilities: incompats}, nil
	}}
	h := newIncompatHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body listIncompatibilitiesResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Incompatibilities, 2)
}

func TestIncompatList_FilteredByLevel(t *testing.T) {
	var captured *domain.IncompatibilityLevel
	stub := &stubIncompatibilityLister{fn: func(ctx context.Context, in incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error) {
		captured = in.Level
		return incompatuc.ListIncompatibilitiesOutput{Incompatibilities: nil}, nil
	}}
	h := newIncompatHandlerLst(stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities?level=MEDIA", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, captured)
	assert.Equal(t, domain.IncompatibilityLevelMedia, *captured)
}

func TestIncompatList_InvalidLevelFilter(t *testing.T) {
	h := newIncompatHandlerLst(&stubIncompatibilityLister{fn: func(context.Context, incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error) {
		return incompatuc.ListIncompatibilitiesOutput{}, &incompatuc.ValidationError{Field: "level"}
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities?level=OTHER", "")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatList_InternalError(t *testing.T) {
	h := newIncompatHandlerLst(&stubIncompatibilityLister{fn: func(context.Context, incompatuc.ListIncompatibilitiesInput) (incompatuc.ListIncompatibilitiesOutput, error) {
		return incompatuc.ListIncompatibilitiesOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities", "")
	h.List(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIncompatGetByID_Success(t *testing.T) {
	want := domain.MustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
	h := newIncompatHandlerGet(&stubIncompatibilityGetter{fn: func(ctx context.Context, in incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error) {
		assert.Equal(t, 3, in.ID)
		return incompatuc.GetIncompatibilityOutput{Incompatibility: want}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var body incompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 3, body.ID)
	assert.Equal(t, "Miedo a petardos", body.Name)
}

func TestIncompatGetByID_InvalidID(t *testing.T) {
	h := newIncompatHandlerGet(&stubIncompatibilityGetter{fn: func(context.Context, incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error) {
		return incompatuc.GetIncompatibilityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities/abc", "")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatGetByID_NotFound(t *testing.T) {
	h := newIncompatHandlerGet(&stubIncompatibilityGetter{fn: func(context.Context, incompatuc.GetIncompatibilityInput) (incompatuc.GetIncompatibilityOutput, error) {
		return incompatuc.GetIncompatibilityOutput{}, incompatuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/incompatibilities/999", "")
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncompatModify_Success(t *testing.T) {
	existing := domain.MustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja)
	h := newIncompatHandlerMod(&stubIncompatibilityModifier{fn: func(ctx context.Context, in incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
		assert.Equal(t, 3, in.ID)
		assert.NotNil(t, in.Patch.Name)
		assert.Equal(t, "Miedo a petardos y cohetes", *in.Patch.Name)
		return incompatuc.ModifyIncompatibilityOutput{Incompatibility: existing}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/incompatibilities/3", `{"name":"Miedo a petardos y cohetes"}`)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Modify(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIncompatModify_InvalidID(t *testing.T) {
	h := newIncompatHandlerMod(&stubIncompatibilityModifier{fn: func(context.Context, incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
		return incompatuc.ModifyIncompatibilityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/incompatibilities/abc", `{}`)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	h.Modify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatModify_NotFound(t *testing.T) {
	h := newIncompatHandlerMod(&stubIncompatibilityModifier{fn: func(context.Context, incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
		return incompatuc.ModifyIncompatibilityOutput{}, incompatuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/incompatibilities/999", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	h.Modify(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncompatModify_DuplicateName(t *testing.T) {
	h := newIncompatHandlerMod(&stubIncompatibilityModifier{fn: func(context.Context, incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
		return incompatuc.ModifyIncompatibilityOutput{}, incompatuc.ErrDuplicateName
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/incompatibilities/3", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Modify(c)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestIncompatModify_InternalError(t *testing.T) {
	h := newIncompatHandlerMod(&stubIncompatibilityModifier{fn: func(context.Context, incompatuc.ModifyIncompatibilityInput) (incompatuc.ModifyIncompatibilityOutput, error) {
		return incompatuc.ModifyIncompatibilityOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodPatch, "/api/v1/incompatibilities/3", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Modify(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIncompatDelete_Success(t *testing.T) {
	h := newIncompatHandlerDel(&stubIncompatibilityDeleter{fn: func(ctx context.Context, in incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
		assert.Equal(t, 3, in.ID)
		return incompatuc.DeleteIncompatibilityOutput{ID: 3}, nil
	}})
	c, w := setupCtx(http.MethodDelete, "/api/v1/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Delete(c)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestIncompatDelete_InvalidID(t *testing.T) {
	h := newIncompatHandlerDel(&stubIncompatibilityDeleter{fn: func(context.Context, incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
		return incompatuc.DeleteIncompatibilityOutput{}, nil
	}})
	c, w := setupCtx(http.MethodDelete, "/api/v1/incompatibilities/abc", "")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncompatDelete_NotFound(t *testing.T) {
	h := newIncompatHandlerDel(&stubIncompatibilityDeleter{fn: func(context.Context, incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
		return incompatuc.DeleteIncompatibilityOutput{}, incompatuc.ErrNotFound
	}})
	c, w := setupCtx(http.MethodDelete, "/api/v1/incompatibilities/999", "")
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	h.Delete(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncompatDelete_InUse(t *testing.T) {
	h := newIncompatHandlerDel(&stubIncompatibilityDeleter{fn: func(context.Context, incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
		return incompatuc.DeleteIncompatibilityOutput{}, postgres.ErrIncompatibilityInUse
	}})
	c, w := setupCtx(http.MethodDelete, "/api/v1/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Delete(c)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestIncompatDelete_InternalError(t *testing.T) {
	h := newIncompatHandlerDel(&stubIncompatibilityDeleter{fn: func(context.Context, incompatuc.DeleteIncompatibilityInput) (incompatuc.DeleteIncompatibilityOutput, error) {
		return incompatuc.DeleteIncompatibilityOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodDelete, "/api/v1/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	h.Delete(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
