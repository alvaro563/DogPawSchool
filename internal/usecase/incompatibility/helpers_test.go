package incompatibility

import (
	"context"

	"dogpaw/internal/domain"
)

type mockIncompatibilityRepository struct {
	getIncompatibilityByID func(ctx context.Context, id int) (*domain.Incompatibility, error)
	create                 func(ctx context.Context, incomp *domain.Incompatibility) (int, error)
	list                   func(ctx context.Context, level *domain.IncompatibilityLevel) ([]*domain.Incompatibility, error)
	update                 func(ctx context.Context, incomp *domain.Incompatibility) error
	delete                 func(ctx context.Context, id int) error
}

func (m *mockIncompatibilityRepository) GetIncompatibilityByID(ctx context.Context, id int) (*domain.Incompatibility, error) {
	if m.getIncompatibilityByID != nil {
		return m.getIncompatibilityByID(ctx, id)
	}
	return nil, nil
}

func (m *mockIncompatibilityRepository) Create(ctx context.Context, incomp *domain.Incompatibility) (int, error) {
	if m.create != nil {
		return m.create(ctx, incomp)
	}
	return 0, nil
}

func (m *mockIncompatibilityRepository) List(ctx context.Context, level *domain.IncompatibilityLevel) ([]*domain.Incompatibility, error) {
	if m.list != nil {
		return m.list(ctx, level)
	}
	return nil, nil
}

func (m *mockIncompatibilityRepository) Update(ctx context.Context, incomp *domain.Incompatibility) error {
	if m.update != nil {
		return m.update(ctx, incomp)
	}
	return nil
}

func (m *mockIncompatibilityRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func mustNewIncompatibility(id int, name string, level domain.IncompatibilityLevel) *domain.Incompatibility {
	in, err := domain.NewIncompatibility(id, name, level)
	if err != nil {
		panic(err)
	}
	return in
}
