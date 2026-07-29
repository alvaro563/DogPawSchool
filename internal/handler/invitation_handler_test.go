package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	invitationuc "dogpaw/internal/usecase/invitation"
)

type stubInvitationCreator struct {
	fn func(ctx context.Context, in invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error)
}

func (s *stubInvitationCreator) Execute(ctx context.Context, in invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
	return s.fn(ctx, in)
}

func newTestInvitationHandler(creator InvitationCreator) *InvitationHandler {
	return NewInvitationHandler(creator)
}

func validCreateInvitationBody() string {
	return `{"created_by":1,"email":"newuser@example.com","role":"REGULAR"}`
}

func TestCreateInvitation_Success(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(_ context.Context, in invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			assert.Equal(t, 1, in.CreatedBy())
			assert.Equal(t, "newuser@example.com", in.Email())
			assert.Equal(t, domain.RoleRegular, in.Role())
			return invitationuc.CreateInvitationOutput{ID: 42, Token: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", validCreateInvitationBody())

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body createInvitationResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42, body.ID)
	assert.Len(t, body.Token, 64)
}

func TestCreateInvitation_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(context.Context, invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return invitationuc.CreateInvitationOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", `not-json`)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvitation_InvalidEmail(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(context.Context, invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return invitationuc.CreateInvitationOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", `{"created_by":1,"email":"bad-email","role":"REGULAR"}`)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvitation_InvalidRole(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(context.Context, invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			t.Fatal("use case should not be called")
			return invitationuc.CreateInvitationOutput{}, nil
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", `{"created_by":1,"email":"u@example.com","role":"INVALID"}`)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvitation_UseCaseError(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(context.Context, invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			return invitationuc.CreateInvitationOutput{}, errors.New("internal error")
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", validCreateInvitationBody())

	h.Create(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateInvitation_DuplicateToken(t *testing.T) {
	t.Parallel()
	h := newTestInvitationHandler(&stubInvitationCreator{
		fn: func(context.Context, invitationuc.CreateInvitationInput) (invitationuc.CreateInvitationOutput, error) {
			return invitationuc.CreateInvitationOutput{}, domain.ErrDuplicateToken
		},
	})
	c, w := setupCtx(http.MethodPost, "/api/v1/invitations", validCreateInvitationBody())

	h.Create(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}
