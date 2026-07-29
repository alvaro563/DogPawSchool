package invitation

import (
	"context"

	"dogpaw/internal/domain"
)

// mockInvitationRepository implements domain.InvitationRepository for
// unit tests. Each method delegates to a closure field so individual
// tests can customize behavior; any unset field returns the zero value
// with no error.
type mockInvitationRepository struct {
	create              func(ctx context.Context, inv *domain.Invitation) (int, error)
	getByID             func(ctx context.Context, id int) (*domain.Invitation, error)
	getByToken          func(ctx context.Context, token string) (*domain.Invitation, error)
	getByTokenForUpdate func(ctx context.Context, token string) (*domain.Invitation, error)
	update              func(ctx context.Context, inv *domain.Invitation) error
	listPending         func(ctx context.Context, limit, offset int) ([]*domain.Invitation, error)
	listByEmail         func(ctx context.Context, email string) ([]*domain.Invitation, error)
}

func (m *mockInvitationRepository) Create(ctx context.Context, inv *domain.Invitation) (int, error) {
	if m.create != nil {
		return m.create(ctx, inv)
	}
	return 0, nil
}

func (m *mockInvitationRepository) GetByID(ctx context.Context, id int) (*domain.Invitation, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockInvitationRepository) GetByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	if m.getByToken != nil {
		return m.getByToken(ctx, token)
	}
	return nil, nil
}

func (m *mockInvitationRepository) GetByTokenForUpdate(ctx context.Context, token string) (*domain.Invitation, error) {
	if m.getByTokenForUpdate != nil {
		return m.getByTokenForUpdate(ctx, token)
	}
	return nil, nil
}

func (m *mockInvitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	if m.update != nil {
		return m.update(ctx, inv)
	}
	return nil
}

func (m *mockInvitationRepository) ListPending(ctx context.Context, limit, offset int) ([]*domain.Invitation, error) {
	if m.listPending != nil {
		return m.listPending(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockInvitationRepository) ListByEmail(ctx context.Context, email string) ([]*domain.Invitation, error) {
	if m.listByEmail != nil {
		return m.listByEmail(ctx, email)
	}
	return nil, nil
}
