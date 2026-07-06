package dog

import (
	"context"

	"dogpaw/internal/domain"
)

type mockDogRepository struct {
	create      func(ctx context.Context, dog *domain.Dog) error
	update      func(ctx context.Context, dog *domain.Dog) error
	getByID     func(ctx context.Context, id int) (*domain.Dog, error)
	listByOwner func(ctx context.Context, userID int) ([]*domain.Dog, error)
	delete      func(ctx context.Context, id int) error
}

func (m *mockDogRepository) Create(ctx context.Context, dog *domain.Dog) error {
	if m.create != nil {
		return m.create(ctx, dog)
	}
	return nil
}

func (m *mockDogRepository) Update(ctx context.Context, dog *domain.Dog) error {
	if m.update != nil {
		return m.update(ctx, dog)
	}
	return nil
}

func (m *mockDogRepository) GetByID(ctx context.Context, id int) (*domain.Dog, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByOwner(ctx context.Context, userID int) ([]*domain.Dog, error) {
	if m.listByOwner != nil {
		return m.listByOwner(ctx, userID)
	}
	return nil, nil
}

func (m *mockDogRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func validIncompatibility() *domain.Incompatibility {
	i, err := domain.NewIncompatibility(1, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta)
	if err != nil {
		panic(err)
	}
	return i
}

func newIncompatibility(id int, nombre string, tipo domain.IncompatibilityLevel) *domain.Incompatibility {
	i, err := domain.NewIncompatibility(id, nombre, tipo)
	if err != nil {
		panic(err)
	}
	return i
}
