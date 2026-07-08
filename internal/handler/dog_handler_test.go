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

func newTestHandler(reg DogRegistrar, lst DogListerByOwner) *DogHandler {
	return NewDogHandler(reg, lst)
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
