package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	doguc "dogpaw/internal/usecase/dog"
)

type stubRegistrar struct {
	fn func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error)
}

func (s *stubRegistrar) Execute(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
	return s.fn(ctx, in)
}

type stubLister struct {
	fn func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

func (s *stubLister) Execute(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
	return s.fn(ctx, in)
}

type stubModifier struct {
	fn func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error)
}

func (s *stubModifier) Execute(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityAdder struct {
	fn func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error)
}

func (s *stubIncompatibilityAdder) Execute(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

type stubIncompatibilityRemover struct {
	fn func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error)
}

func (s *stubIncompatibilityRemover) Execute(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

func newTestHandler(reg DogRegistrar, lst DogListerByOwner) *DogHandler {
	return NewDogHandler(reg, lst, nil, nil, nil)
}

func newTestHandlerFull(reg DogRegistrar, lst DogListerByOwner, mod DogModifier) *DogHandler {
	return NewDogHandler(reg, lst, mod, nil, nil)
}

func newTestHandlerFull4(reg DogRegistrar, lst DogListerByOwner, mod DogModifier, addIncompat DogIncompatibilityAdder) *DogHandler {
	return NewDogHandler(reg, lst, mod, addIncompat, nil)
}

func newTestHandlerFull5(reg DogRegistrar, lst DogListerByOwner, mod DogModifier, addIncompat DogIncompatibilityAdder, removeIncompat DogIncompatibilityRemover) *DogHandler {
	return NewDogHandler(reg, lst, mod, addIncompat, removeIncompat)
}

func mustNewIncompatibility(id int, name string, level domain.IncompatibilityLevel) domain.Incompatibility {
	in, err := domain.NewIncompatibility(id, name, level)
	if err != nil {
		panic(err)
	}
	return *in
}

func setupCtx(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	return c, w
}

func validRegisterBody() string {
	return `{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-1","user_id":1}`
}

func newTestDog(id int) *domain.Dog {
	d, _ := domain.NewDog(id, "Luna", "Labrador", "ES-"+strconv.Itoa(id), 24, domain.SexFemale, 22.5, 1)
	return d
}

func TestRegister_Success(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{ID: 42}, nil
	}}
	h := newTestHandler(stub, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/dogs/42", w.Header().Get("Location"))
	var body registerDogResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", "not json")

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestRegister_BindingValidation(t *testing.T) {
	h := newTestHandler(nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", `{"name":"Luna"}`)

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_UseCaseValidation(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, &doguc.ValidationError{Field: "breed"}
	}}
	h := newTestHandler(stub, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"breed"`)
}

func TestRegister_InvalidUser(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, postgres.ErrInvalidUser
	}}
	h := newTestHandler(stub, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_user_id")
}

func TestRegister_DuplicatePassport(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, postgres.ErrDuplicatePassport
	}}
	h := newTestHandler(stub, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate_passport")
}

func TestRegister_InternalError(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, errors.New("db down")
	}}
	h := newTestHandler(stub, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal")
}

func TestList_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1), newTestDog(2)}
	stub := &stubLister{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		assert.Equal(t, 1, in.OwnerID)
		return doguc.ListByOwnerOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs?owner_id=1&limit=10&offset=0", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 2)
	assert.Equal(t, 2, body.Count)
	assert.Equal(t, 10, body.Limit)
	assert.Equal(t, 0, body.Offset)
}

func TestList_Empty(t *testing.T) {
	stub := &stubLister{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		return doguc.ListByOwnerOutput{Dogs: []*domain.Dog{}}, nil
	}}
	h := newTestHandler(nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs?owner_id=1", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Count)
}

func TestList_InvalidOwnerID(t *testing.T) {
	h := newTestHandler(nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs", "")

	h.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "owner_id")
}

func TestList_NegativeOwnerID(t *testing.T) {
	h := newTestHandler(nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs?owner_id=-5", "")

	h.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_InternalError(t *testing.T) {
	stub := &stubLister{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		return doguc.ListByOwnerOutput{}, errors.New("db down")
	}}
	h := newTestHandler(nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs?owner_id=1", "")

	h.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestModify_Success(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		assert.Equal(t, 42, in.ID)
		assert.NotNil(t, in.Patch.Name)
		assert.Equal(t, "Buddie", *in.Patch.Name)
		return doguc.ModifyDogOutput{ID: 42}, nil
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{"name":"Buddie"}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body modifyDogResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
}

func TestModify_EmptyPatchIsNoop(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		return doguc.ModifyDogOutput{ID: 42}, nil
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestModify_InvalidID_NonNumeric(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/abc", `{"name":"x"}`)

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestModify_InvalidID_ZeroOrNegative(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/0", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "0"}}

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestModify_InvalidJSON(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", "not json")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestModify_UseCaseValidation(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		return doguc.ModifyDogOutput{}, &doguc.ValidationError{Field: "name"}
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{"name":""}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"name"`)
}

func TestModify_NotFound(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		return doguc.ModifyDogOutput{}, doguc.ErrNotFound
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/999", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	h.Modify(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not_found")
}

func TestModify_DuplicatePassport(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		return doguc.ModifyDogOutput{}, postgres.ErrDuplicatePassport
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{"passport":"ES-1"}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate_passport")
}

func TestModify_InternalError(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		return doguc.ModifyDogOutput{}, errors.New("db down")
	}}
	h := newTestHandlerFull(nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddIncompatibility_Success_Added(t *testing.T) {
	incompats := []domain.Incompatibility{
		mustNewIncompatibility(1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta),
		mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja),
	}
	stub := &stubIncompatibilityAdder{fn: func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
		assert.Equal(t, 42, in.DogID)
		assert.Equal(t, 3, in.IncompatibilityID)
		return doguc.AddDogIncompatibilityOutput{
			ID: 42, Added: true, Incompatibilities: incompats,
		}, nil
	}}
	h := newTestHandlerFull4(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body addIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.DogID)
	assert.True(t, body.Added)
	assert.Len(t, body.Incompatibilities, 2)
	assert.Equal(t, 3, body.Incompatibilities[1].ID)
	assert.Equal(t, "Miedo a petardos", body.Incompatibilities[1].Name)
	assert.Equal(t, "BAJA", body.Incompatibilities[1].Level)
}

func TestAddIncompatibility_Idempotent_Returns200(t *testing.T) {
	incompats := []domain.Incompatibility{
		mustNewIncompatibility(3, "Miedo a petardos", domain.IncompatibilityLevelBaja),
	}
	stub := &stubIncompatibilityAdder{fn: func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
		return doguc.AddDogIncompatibilityOutput{
			ID: 42, Added: false, Incompatibilities: incompats,
		}, nil
	}}
	h := newTestHandlerFull4(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body addIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.DogID)
	assert.False(t, body.Added)
	assert.Len(t, body.Incompatibilities, 1)
}

func TestAddIncompatibility_InvalidDogID(t *testing.T) {
	h := newTestHandlerFull4(nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/abc/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestAddIncompatibility_InvalidJSON(t *testing.T) {
	h := newTestHandlerFull4(nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", "not json")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestAddIncompatibility_BindingValidation(t *testing.T) {
	h := newTestHandlerFull4(nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", `{"incompatibility_id":0}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestAddIncompatibility_NotFound_UseCase(t *testing.T) {
	stub := &stubIncompatibilityAdder{fn: func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
		return doguc.AddDogIncompatibilityOutput{}, doguc.ErrNotFound
	}}
	h := newTestHandlerFull4(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/999/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not_found")
}

func TestAddIncompatibility_NotFound_Repo(t *testing.T) {
	stub := &stubIncompatibilityAdder{fn: func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
		return doguc.AddDogIncompatibilityOutput{}, postgres.ErrNotFound
	}}
	h := newTestHandlerFull4(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/999/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not_found")
}

func TestAddIncompatibility_InternalError(t *testing.T) {
	stub := &stubIncompatibilityAdder{fn: func(ctx context.Context, in doguc.AddDogIncompatibilityInput) (doguc.AddDogIncompatibilityOutput, error) {
		return doguc.AddDogIncompatibilityOutput{}, errors.New("db down")
	}}
	h := newTestHandlerFull4(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRemoveIncompatibility_Success_Removed(t *testing.T) {
	incompats := []domain.Incompatibility{
		mustNewIncompatibility(1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta),
	}
	stub := &stubIncompatibilityRemover{fn: func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
		assert.Equal(t, 42, in.DogID)
		assert.Equal(t, 3, in.IncompatibilityID)
		return doguc.RemoveDogIncompatibilityOutput{
			ID: 42, Removed: true, Incompatibilities: incompats,
		}, nil
	}}
	h := newTestHandlerFull5(nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body removeIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.DogID)
	assert.True(t, body.Removed)
	assert.Len(t, body.Incompatibilities, 1)
}

func TestRemoveIncompatibility_Idempotent_NotPresent(t *testing.T) {
	incompats := []domain.Incompatibility{
		mustNewIncompatibility(2, "No tolera cachorros", domain.IncompatibilityLevelMedia),
	}
	stub := &stubIncompatibilityRemover{fn: func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
		return doguc.RemoveDogIncompatibilityOutput{
			ID: 42, Removed: false, Incompatibilities: incompats,
		}, nil
	}}
	h := newTestHandlerFull5(nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body removeIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Removed, "idempotent: incompat was not present")
}

func TestRemoveIncompatibility_InvalidDogID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/abc/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestRemoveIncompatibility_ZeroOrNegativeDogID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/0/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "0"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestRemoveIncompatibility_InvalidIncompatID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/abc", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "abc"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"incompatibility_id"`)
}

func TestRemoveIncompatibility_ZeroIncompatID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/0", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "0"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"incompatibility_id"`)
}

func TestRemoveIncompatibility_DogNotFound(t *testing.T) {
	stub := &stubIncompatibilityRemover{fn: func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
		return doguc.RemoveDogIncompatibilityOutput{}, doguc.ErrNotFound
	}}
	h := newTestHandlerFull5(nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/999/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "999"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not_found")
}

func TestRemoveIncompatibility_UseCaseValidation(t *testing.T) {
	stub := &stubIncompatibilityRemover{fn: func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
		return doguc.RemoveDogIncompatibilityOutput{}, &doguc.ValidationError{Field: "incompatibility_id"}
	}}
	h := newTestHandlerFull5(nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"incompatibility_id"`)
}

func TestRemoveIncompatibility_InternalError(t *testing.T) {
	stub := &stubIncompatibilityRemover{fn: func(ctx context.Context, in doguc.RemoveDogIncompatibilityInput) (doguc.RemoveDogIncompatibilityOutput, error) {
		return doguc.RemoveDogIncompatibilityOutput{}, errors.New("db down")
	}}
	h := newTestHandlerFull5(nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
