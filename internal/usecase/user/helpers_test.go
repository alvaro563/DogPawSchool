package user

import (
	"context"

	"dogpaw/internal/domain"
)

// mockUserRepository implements domain.UserRepository for unit tests.
// Each method delegates to a closure field so individual tests can
// customize behavior on a per-method basis; any unset field returns
// the zero value with no error.
type mockUserRepository struct {
	create       func(ctx context.Context, user *domain.User) error
	update       func(ctx context.Context, user *domain.User) error
	getByID      func(ctx context.Context, id int) (*domain.User, error)
	getByEmail   func(ctx context.Context, email string) (*domain.User, error)
	listAll      func(ctx context.Context) ([]*domain.User, error)
	listAllPaged func(ctx context.Context, limit, offset int) ([]*domain.User, error)
	delete       func(ctx context.Context, id int) error
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.create != nil {
		return m.create(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.update != nil {
		return m.update(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmail != nil {
		return m.getByEmail(ctx, email)
	}
	return nil, nil
}

func (m *mockUserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	if m.listAll != nil {
		return m.listAll(ctx)
	}
	return nil, nil
}

func (m *mockUserRepository) ListAllPaged(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if m.listAllPaged != nil {
		return m.listAllPaged(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// newTestUser builds a valid active user for tests. The password is a
// 60-char placeholder to satisfy domain.NewUser validation.
func newTestUser(id int) *domain.User {
	u, err := domain.NewUser(id, "Test User", "test@example.com", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}
