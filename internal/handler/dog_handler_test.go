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

type stubListerAll struct {
	fn func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error)
}

func (s *stubListerAll) Execute(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByOwner struct {
	fn func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error)
}

func (s *stubListerByOwner) Execute(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
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

type stubListerActive struct {
	fn func(ctx context.Context, in doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error)
}

func (s *stubListerActive) Execute(ctx context.Context, in doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByIsActive struct {
	fn func(ctx context.Context, in doguc.ListByIsActiveInput) (doguc.ListByIsActiveOutput, error)
}

func (s *stubListerByIsActive) Execute(ctx context.Context, in doguc.ListByIsActiveInput) (doguc.ListByIsActiveOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByIncompatibility struct {
	fn func(ctx context.Context, in doguc.ListByIncompatibilityInput) (doguc.ListByIncompatibilityOutput, error)
}

func (s *stubListerByIncompatibility) Execute(ctx context.Context, in doguc.ListByIncompatibilityInput) (doguc.ListByIncompatibilityOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByBreed struct {
	fn func(ctx context.Context, in doguc.ListByBreedInput) (doguc.ListByBreedOutput, error)
}

func (s *stubListerByBreed) Execute(ctx context.Context, in doguc.ListByBreedInput) (doguc.ListByBreedOutput, error) {
	return s.fn(ctx, in)
}

type stubListerBySex struct {
	fn func(ctx context.Context, in doguc.ListBySexInput) (doguc.ListBySexOutput, error)
}

func (s *stubListerBySex) Execute(ctx context.Context, in doguc.ListBySexInput) (doguc.ListBySexOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByNeutered struct {
	fn func(ctx context.Context, in doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error)
}

func (s *stubListerByNeutered) Execute(ctx context.Context, in doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByHeat struct {
	fn func(ctx context.Context, in doguc.ListByHeatInput) (doguc.ListByHeatOutput, error)
}

func (s *stubListerByHeat) Execute(ctx context.Context, in doguc.ListByHeatInput) (doguc.ListByHeatOutput, error) {
	return s.fn(ctx, in)
}

type stubListerByAgeBracket struct {
	fn func(ctx context.Context, in doguc.ListByAgeBracketInput) (doguc.ListByAgeBracketOutput, error)
}

func (s *stubListerByAgeBracket) Execute(ctx context.Context, in doguc.ListByAgeBracketInput) (doguc.ListByAgeBracketOutput, error) {
	return s.fn(ctx, in)
}

type stubListerBySizeBracket struct {
	fn func(ctx context.Context, in doguc.ListBySizeBracketInput) (doguc.ListBySizeBracketOutput, error)
}

func (s *stubListerBySizeBracket) Execute(ctx context.Context, in doguc.ListBySizeBracketInput) (doguc.ListBySizeBracketOutput, error) {
	return s.fn(ctx, in)
}

type stubDeleter struct {
	fn func(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error)
}

func (s *stubDeleter) Execute(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error) {
	return s.fn(ctx, in)
}

type stubNeuteredSetter struct {
	fn func(ctx context.Context, in doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error)
}

func (s *stubNeuteredSetter) Execute(ctx context.Context, in doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error) {
	return s.fn(ctx, in)
}

type stubHeatSetter struct {
	fn func(ctx context.Context, in doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error)
}

func (s *stubHeatSetter) Execute(ctx context.Context, in doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error) {
	return s.fn(ctx, in)
}

func newTestHandler(reg DogRegistrar, list DogLister, listByOwner DogListerByOwner) *DogHandler {
	return NewDogHandler(reg, list, listByOwner, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func newTestHandlerFull(reg DogRegistrar, list DogLister, listByOwner DogListerByOwner, mod DogModifier) *DogHandler {
	return NewDogHandler(reg, list, listByOwner, nil, nil, nil, nil, nil, nil, nil, nil, nil, mod, nil, nil, nil, nil, nil)
}

func newTestHandlerFull4(reg DogRegistrar, list DogLister, listByOwner DogListerByOwner, mod DogModifier, addIncompat DogIncompatibilityAdder) *DogHandler {
	return NewDogHandler(reg, list, listByOwner, nil, nil, nil, nil, nil, nil, nil, nil, nil, mod, addIncompat, nil, nil, nil, nil)
}

func newTestHandlerFull5(reg DogRegistrar, list DogLister, listByOwner DogListerByOwner, mod DogModifier, addIncompat DogIncompatibilityAdder, removeIncompat DogIncompatibilityRemover) *DogHandler {
	return NewDogHandler(reg, list, listByOwner, nil, nil, nil, nil, nil, nil, nil, nil, nil, mod, addIncompat, removeIncompat, nil, nil, nil)
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
	h := newTestHandler(stub, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/dogs/42", w.Header().Get("Location"))
	var body registerDogResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", "not json")

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestRegister_BindingValidation(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", `{"name":"Luna"}`)

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_UseCaseValidation(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, &doguc.ValidationError{Field: "breed"}
	}}
	h := newTestHandler(stub, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"breed"`)
}

func TestRegister_InvalidUser(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, postgres.ErrInvalidUser
	}}
	h := newTestHandler(stub, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_user_id")
}

func TestRegister_DuplicatePassport(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, postgres.ErrDuplicatePassport
	}}
	h := newTestHandler(stub, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate_passport")
}

func TestRegister_InternalError(t *testing.T) {
	stub := &stubRegistrar{fn: func(ctx context.Context, in doguc.RegisterDogInput) (doguc.RegisterDogOutput, error) {
		return doguc.RegisterDogOutput{}, errors.New("db down")
	}}
	h := newTestHandler(stub, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs", validRegisterBody())

	h.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal")
}

func TestList_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1), newTestDog(2)}
	stub := &stubListerAll{fn: func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
		return doguc.ListAllDogsOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, stub, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs?limit=10&offset=0", "")

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
	stub := &stubListerAll{fn: func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
		return doguc.ListAllDogsOutput{Dogs: []*domain.Dog{}}, nil
	}}
	h := newTestHandler(nil, stub, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Count)
}

func TestList_InternalError(t *testing.T) {
	stub := &stubListerAll{fn: func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
		return doguc.ListAllDogsOutput{}, errors.New("db down")
	}}
	h := newTestHandler(nil, stub, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs", "")

	h.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListByOwner_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1), newTestDog(2)}
	stub := &stubListerByOwner{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		assert.Equal(t, 1, in.OwnerID)
		return doguc.ListByOwnerOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/1?limit=10&offset=0", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "1"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 2)
	assert.Equal(t, 2, body.Count)
	assert.Equal(t, 10, body.Limit)
	assert.Equal(t, 0, body.Offset)
}

func TestListByOwner_Empty(t *testing.T) {
	stub := &stubListerByOwner{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		return doguc.ListByOwnerOutput{Dogs: []*domain.Dog{}}, nil
	}}
	h := newTestHandler(nil, nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/1", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "1"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Count)
}

func TestListByOwner_InvalidOwnerID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/abc", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "abc"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "owner_id")
}

func TestListByOwner_NegativeOwnerID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/-5", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "-5"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByOwner_ZeroOwnerID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/0", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "0"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByOwner_InternalError(t *testing.T) {
	stub := &stubListerByOwner{fn: func(ctx context.Context, in doguc.ListByOwnerInput) (doguc.ListByOwnerOutput, error) {
		return doguc.ListByOwnerOutput{}, errors.New("db down")
	}}
	h := newTestHandler(nil, nil, stub)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/owner/1", "")
	c.Params = gin.Params{{Key: "owner_id", Value: "1"}}

	h.ListByOwner(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestModify_Success(t *testing.T) {
	stub := &stubModifier{fn: func(ctx context.Context, in doguc.ModifyDogInput) (doguc.ModifyDogOutput, error) {
		assert.Equal(t, 42, in.ID)
		assert.NotNil(t, in.Patch.Name)
		assert.Equal(t, "Buddie", *in.Patch.Name)
		return doguc.ModifyDogOutput{ID: 42}, nil
	}}
	h := newTestHandlerFull(nil, nil, nil, stub)
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
	h := newTestHandlerFull(nil, nil, nil, stub)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42", `{}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Modify(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestModify_InvalidID_NonNumeric(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/abc", `{"name":"x"}`)

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestModify_InvalidID_ZeroOrNegative(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/0", `{"name":"x"}`)
	c.Params = gin.Params{{Key: "id", Value: "0"}}

	h.Modify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestModify_InvalidJSON(t *testing.T) {
	h := newTestHandlerFull(nil, nil, nil, nil)
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
	h := newTestHandlerFull(nil, nil, nil, stub)
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
	h := newTestHandlerFull(nil, nil, nil, stub)
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
	h := newTestHandlerFull(nil, nil, nil, stub)
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
	h := newTestHandlerFull(nil, nil, nil, stub)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/abc/incompatibilities", `{"incompatibility_id":3}`)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestAddIncompatibility_InvalidJSON(t *testing.T) {
	h := newTestHandlerFull4(nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodPost, "/api/v1/dogs/42/incompatibilities", "not json")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.AddIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestAddIncompatibility_BindingValidation(t *testing.T) {
	h := newTestHandlerFull4(nil, nil, nil, nil, nil)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull4(nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body removeIncompatibilityResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Removed, "idempotent: incompat was not present")
}

func TestRemoveIncompatibility_InvalidDogID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/abc/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestRemoveIncompatibility_ZeroOrNegativeDogID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/0/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "0"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestRemoveIncompatibility_InvalidIncompatID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/abc", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "abc"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"incompatibility_id"`)
}

func TestRemoveIncompatibility_ZeroIncompatID(t *testing.T) {
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, nil)
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
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, stub)
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
	h := newTestHandlerFull5(nil, nil, nil, nil, nil, stub)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42/incompatibilities/3", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "incompatibility_id", Value: "3"}}

	h.RemoveIncompatibility(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Tests for the 9 new LIST handler methods and Delete
// ============================================================================

func TestListActive_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1), newTestDog(2)}
	stub := &stubListerActive{fn: func(ctx context.Context, in doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error) {
		return doguc.ListActiveDogsOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listActive = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/active", "")

	h.ListActive(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 2)
	assert.Equal(t, 2, body.Count)
	assert.Equal(t, 50, body.Limit)
}

func TestListActive_InternalError(t *testing.T) {
	stub := &stubListerActive{fn: func(ctx context.Context, in doguc.ListActiveDogsInput) (doguc.ListActiveDogsOutput, error) {
		return doguc.ListActiveDogsOutput{}, errors.New("db down")
	}}
	h := newTestHandler(nil, nil, nil)
	h.listActive = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/active", "")

	h.ListActive(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListByIsActive_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerByIsActive{fn: func(ctx context.Context, in doguc.ListByIsActiveInput) (doguc.ListByIsActiveOutput, error) {
		assert.True(t, in.IsActive)
		return doguc.ListByIsActiveOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByIsActive = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/is_active/true", "")
	c.Params = gin.Params{{Key: "value", Value: "true"}}

	h.ListByIsActive(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 1)
}

func TestListByIsActive_InvalidValue(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/is_active/banana", "")
	c.Params = gin.Params{{Key: "value", Value: "banana"}}

	h.ListByIsActive(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"value"`)
}

func TestListByIncompatibility_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1), newTestDog(2), newTestDog(3)}
	stub := &stubListerByIncompatibility{fn: func(ctx context.Context, in doguc.ListByIncompatibilityInput) (doguc.ListByIncompatibilityOutput, error) {
		assert.Equal(t, 5, in.IncompatibilityID)
		return doguc.ListByIncompatibilityOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByIncompatibility = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/incompatibility/5", "")
	c.Params = gin.Params{{Key: "incompat_id", Value: "5"}}

	h.ListByIncompatibility(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 3)
}

func TestListByIncompatibility_InvalidID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/incompatibility/abc", "")
	c.Params = gin.Params{{Key: "incompat_id", Value: "abc"}}

	h.ListByIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "incompatibility_id")
}

func TestListByIncompatibility_ZeroID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/incompatibility/0", "")
	c.Params = gin.Params{{Key: "incompat_id", Value: "0"}}

	h.ListByIncompatibility(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByBreed_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerByBreed{fn: func(ctx context.Context, in doguc.ListByBreedInput) (doguc.ListByBreedOutput, error) {
		assert.Equal(t, "Labrador", in.Breed)
		return doguc.ListByBreedOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByBreed = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/breed/Labrador", "")
	c.Params = gin.Params{{Key: "breed", Value: "Labrador"}}

	h.ListByBreed(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 1)
}

func TestListByBreed_Empty(t *testing.T) {
	stub := &stubListerByBreed{fn: func(ctx context.Context, in doguc.ListByBreedInput) (doguc.ListByBreedOutput, error) {
		return doguc.ListByBreedOutput{Dogs: []*domain.Dog{}}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByBreed = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/breed/NoExist", "")
	c.Params = gin.Params{{Key: "breed", Value: "NoExist"}}

	h.ListByBreed(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Count)
}

func TestListBySex_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerBySex{fn: func(ctx context.Context, in doguc.ListBySexInput) (doguc.ListBySexOutput, error) {
		assert.Equal(t, domain.SexFemale, in.Sex)
		return doguc.ListBySexOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listBySex = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/sex/FEMALE", "")
	c.Params = gin.Params{{Key: "sex", Value: "FEMALE"}}

	h.ListBySex(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 1)
}

func TestListBySex_InvalidSex(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/sex/INVALID", "")
	c.Params = gin.Params{{Key: "sex", Value: "INVALID"}}

	h.ListBySex(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "sex")
}

func TestListByNeutered_True(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerByNeutered{fn: func(ctx context.Context, in doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error) {
		assert.True(t, in.Neutered)
		return doguc.ListByNeuteredOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByNeutered = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/neutered/true", "")
	c.Params = gin.Params{{Key: "value", Value: "true"}}

	h.ListByNeutered(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByNeutered_False(t *testing.T) {
	stub := &stubListerByNeutered{fn: func(ctx context.Context, in doguc.ListByNeuteredInput) (doguc.ListByNeuteredOutput, error) {
		assert.False(t, in.Neutered)
		return doguc.ListByNeuteredOutput{Dogs: []*domain.Dog{}}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByNeutered = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/neutered/false", "")
	c.Params = gin.Params{{Key: "value", Value: "false"}}

	h.ListByNeutered(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByNeutered_InvalidValue(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/neutered/banana", "")
	c.Params = gin.Params{{Key: "value", Value: "banana"}}

	h.ListByNeutered(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByHeat_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerByHeat{fn: func(ctx context.Context, in doguc.ListByHeatInput) (doguc.ListByHeatOutput, error) {
		assert.True(t, in.Heat)
		return doguc.ListByHeatOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByHeat = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/heat/true", "")
	c.Params = gin.Params{{Key: "value", Value: "true"}}

	h.ListByHeat(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByHeat_InvalidValue(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/heat/banana", "")
	c.Params = gin.Params{{Key: "value", Value: "banana"}}

	h.ListByHeat(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListByAgeBracket_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerByAgeBracket{fn: func(ctx context.Context, in doguc.ListByAgeBracketInput) (doguc.ListByAgeBracketOutput, error) {
		assert.Equal(t, domain.AgeBracketChildren, in.AgeBracket)
		return doguc.ListByAgeBracketOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listByAgeBracket = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/age/CHILDREN", "")
	c.Params = gin.Params{{Key: "bracket", Value: "CHILDREN"}}

	h.ListByAgeBracket(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListByAgeBracket_InvalidBracket(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/age/BOGUS", "")
	c.Params = gin.Params{{Key: "bracket", Value: "BOGUS"}}

	h.ListByAgeBracket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "age_bracket")
}

func TestListBySizeBracket_Success(t *testing.T) {
	dogs := []*domain.Dog{newTestDog(1)}
	stub := &stubListerBySizeBracket{fn: func(ctx context.Context, in doguc.ListBySizeBracketInput) (doguc.ListBySizeBracketOutput, error) {
		assert.Equal(t, domain.SizeBracketLarge, in.SizeBracket)
		return doguc.ListBySizeBracketOutput{Dogs: dogs}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.listBySizeBracket = stub
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/size/LARGE", "")
	c.Params = gin.Params{{Key: "bracket", Value: "LARGE"}}

	h.ListBySizeBracket(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListBySizeBracket_InvalidBracket(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs/size/BOGUS", "")
	c.Params = gin.Params{{Key: "bracket", Value: "BOGUS"}}

	h.ListBySizeBracket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "size_bracket")
}

func TestDelete_Success(t *testing.T) {
	var capturedID int
	stub := &stubDeleter{fn: func(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error) {
		capturedID = in.ID
		return doguc.DeleteDogOutput{}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.delete = stub
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 42, capturedID)
}

func TestDelete_NotFound(t *testing.T) {
	stub := &stubDeleter{fn: func(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error) {
		return doguc.DeleteDogOutput{}, postgres.ErrNotFound
	}}
	h := newTestHandler(nil, nil, nil)
	h.delete = stub
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/9999", "")
	c.Params = gin.Params{{Key: "id", Value: "9999"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_InvalidID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/abc", "")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "id")
}

func TestDelete_ZeroID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/0", "")
	c.Params = gin.Params{{Key: "id", Value: "0"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_UseCaseValidationError(t *testing.T) {
	stub := &stubDeleter{fn: func(ctx context.Context, in doguc.DeleteDogInput) (doguc.DeleteDogOutput, error) {
		return doguc.DeleteDogOutput{}, &doguc.ValidationError{Field: "id"}
	}}
	h := newTestHandler(nil, nil, nil)
	h.delete = stub
	c, w := setupCtx(http.MethodDelete, "/api/v1/dogs/42", "")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

// ============================================================================
// Incompatibilities in list responses (additive breaking change)
// ============================================================================

func TestList_IncludesIncompatibilities(t *testing.T) {
	// Build a dog with 2 incompatibilities using the domain API.
	dog, err := domain.NewDog(1, "Luna", "Labrador", "ES-Luna", 24,
		domain.SexFemale, 22.5, 1)
	assert.NoError(t, err)
	in1 := mustNewIncompatibility(1, "Miedoso con extraños", domain.IncompatibilityLevelBaja)
	in2 := mustNewIncompatibility(2, "Protección de recursos", domain.IncompatibilityLevelAbsoluta)
	_, _ = dog.AddIncompatibility(&in1)
	_, _ = dog.AddIncompatibility(&in2)

	stub := &stubListerAll{fn: func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
		return doguc.ListAllDogsOutput{Dogs: []*domain.Dog{dog}}, nil
	}}
	h := newTestHandler(nil, stub, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDogsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Dogs, 1)
	assert.Len(t, body.Dogs[0].Incompatibilities, 2,
		"list response must include incompatibilities")
	assert.Equal(t, "Miedoso con extraños", body.Dogs[0].Incompatibilities[0].Name)
	assert.Equal(t, "Protección de recursos", body.Dogs[0].Incompatibilities[1].Name)
}

func TestList_IncludesEmptyIncompatibilitiesArray(t *testing.T) {
	// Dogs with no incompats must still have the field present (as []), not null.
	dog, err := domain.NewDog(1, "Luna", "Labrador", "ES-Luna", 24,
		domain.SexFemale, 22.5, 1)
	assert.NoError(t, err)

	stub := &stubListerAll{fn: func(ctx context.Context, in doguc.ListAllDogsInput) (doguc.ListAllDogsOutput, error) {
		return doguc.ListAllDogsOutput{Dogs: []*domain.Dog{dog}}, nil
	}}
	h := newTestHandler(nil, stub, nil)
	c, w := setupCtx(http.MethodGet, "/api/v1/dogs", "")

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Field must be present and empty, not absent / null.
	assert.Contains(t, w.Body.String(), `"incompatibilities":[]`,
		"incompatibilities must serialize as an empty array, not null")
}

// ============================================================================
// SetNeutered / SetHeat handler tests
// ============================================================================

func TestSetNeutered_Success(t *testing.T) {
	stub := &stubNeuteredSetter{fn: func(ctx context.Context, in doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error) {
		assert.Equal(t, 42, in.ID)
		assert.True(t, in.Neutered)
		return doguc.SetDogNeuteredOutput{ID: 42, Neutered: true, Sex: domain.SexFemale}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.setNeutered = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/42/neutered", `{"neutered":true}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.SetNeutered(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body setNeuteredResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
	assert.True(t, body.Neutered)
	assert.Equal(t, "FEMALE", body.Sex)
}

func TestSetNeutered_NotFound(t *testing.T) {
	stub := &stubNeuteredSetter{fn: func(ctx context.Context, in doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error) {
		return doguc.SetDogNeuteredOutput{}, postgres.ErrNotFound
	}}
	h := newTestHandler(nil, nil, nil)
	h.setNeutered = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/9999/neutered", `{"neutered":true}`)
	c.Params = gin.Params{{Key: "id", Value: "9999"}}

	h.SetNeutered(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetNeutered_InvalidID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/abc/neutered", `{"neutered":true}`)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.SetNeutered(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetNeutered_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/1/neutered", `not json`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.SetNeutered(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_request")
}

func TestSetNeutered_UseCaseValidation(t *testing.T) {
	stub := &stubNeuteredSetter{fn: func(ctx context.Context, in doguc.SetDogNeuteredInput) (doguc.SetDogNeuteredOutput, error) {
		return doguc.SetDogNeuteredOutput{}, &doguc.ValidationError{Field: "id"}
	}}
	h := newTestHandler(nil, nil, nil)
	h.setNeutered = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/1/neutered", `{"neutered":true}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.SetNeutered(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"id"`)
}

func TestSetHeat_Success_Female(t *testing.T) {
	stub := &stubHeatSetter{fn: func(ctx context.Context, in doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error) {
		assert.Equal(t, 2, in.ID)
		assert.True(t, in.Heat)
		return doguc.SetDogHeatOutput{ID: 2, Heat: true, Sex: domain.SexFemale}, nil
	}}
	h := newTestHandler(nil, nil, nil)
	h.setHeat = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/2/heat", `{"heat":true}`)
	c.Params = gin.Params{{Key: "id", Value: "2"}}

	h.SetHeat(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body setHeatResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Heat)
	assert.Equal(t, "FEMALE", body.Sex)
}

func TestSetHeat_RejectedOnMale(t *testing.T) {
	stub := &stubHeatSetter{fn: func(ctx context.Context, in doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error) {
		return doguc.SetDogHeatOutput{}, doguc.ErrInvalidHeatForSex
	}}
	h := newTestHandler(nil, nil, nil)
	h.setHeat = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/9/heat", `{"heat":true}`)
	c.Params = gin.Params{{Key: "id", Value: "9"}}

	h.SetHeat(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_heat_for_sex")
}

func TestSetHeat_NotFound(t *testing.T) {
	stub := &stubHeatSetter{fn: func(ctx context.Context, in doguc.SetDogHeatInput) (doguc.SetDogHeatOutput, error) {
		return doguc.SetDogHeatOutput{}, postgres.ErrNotFound
	}}
	h := newTestHandler(nil, nil, nil)
	h.setHeat = stub
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/9999/heat", `{"heat":false}`)
	c.Params = gin.Params{{Key: "id", Value: "9999"}}

	h.SetHeat(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetHeat_InvalidID(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/0/heat", `{"heat":false}`)
	c.Params = gin.Params{{Key: "id", Value: "0"}}

	h.SetHeat(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetHeat_InvalidJSON(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, w := setupCtx(http.MethodPatch, "/api/v1/dogs/1/heat", `{"heat":"yes"}`)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.SetHeat(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
