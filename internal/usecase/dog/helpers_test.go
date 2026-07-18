package dog

import (
	"context"

	"dogpaw/internal/domain"
)

type mockDogRepository struct {
	create                func(ctx context.Context, dog *domain.Dog) (int, error)
	update                func(ctx context.Context, dog *domain.Dog) error
	getByID               func(ctx context.Context, id int) (*domain.Dog, error)
	listByOwner           func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error)
	listAll               func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error)
	listByIncompatibility func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error)
	listByBreed           func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error)
	listBySex             func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error)
	listByNeutered        func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error)
	listByHeat            func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error)
	listByIsActive        func(ctx context.Context, isActive bool, limit, offset int) ([]*domain.Dog, error)
	listByAgeBracket      func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error)
	listBySizeBracket     func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error)
	setDogNeutered        func(ctx context.Context, id int, neutered bool) error
	setDogHeat            func(ctx context.Context, id int, heat bool) error
	delete                func(ctx context.Context, id int) error
}

func (m *mockDogRepository) Create(ctx context.Context, dog *domain.Dog) (int, error) {
	if m.create != nil {
		return m.create(ctx, dog)
	}
	return 0, nil
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

func (m *mockDogRepository) ListByOwner(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
	if m.listByOwner != nil {
		return m.listByOwner(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
	if m.listAll != nil {
		return m.listAll(ctx, activeOnly, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByIncompatibility(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
	if m.listByIncompatibility != nil {
		return m.listByIncompatibility(ctx, incompatibilityID, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByBreed(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
	if m.listByBreed != nil {
		return m.listByBreed(ctx, breed, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListBySex(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
	if m.listBySex != nil {
		return m.listBySex(ctx, sex, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByNeutered(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
	if m.listByNeutered != nil {
		return m.listByNeutered(ctx, neutered, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByHeat(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
	if m.listByHeat != nil {
		return m.listByHeat(ctx, heat, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByIsActive(ctx context.Context, isActive bool, limit, offset int) ([]*domain.Dog, error) {
	if m.listByIsActive != nil {
		return m.listByIsActive(ctx, isActive, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListByAgeBracket(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
	if m.listByAgeBracket != nil {
		return m.listByAgeBracket(ctx, bracket, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) ListBySizeBracket(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
	if m.listBySizeBracket != nil {
		return m.listBySizeBracket(ctx, bracket, limit, offset)
	}
	return nil, nil
}

func (m *mockDogRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func (m *mockDogRepository) SetDogNeutered(ctx context.Context, id int, neutered bool) error {
	if m.setDogNeutered != nil {
		return m.setDogNeutered(ctx, id, neutered)
	}
	return nil
}

func (m *mockDogRepository) SetDogHeat(ctx context.Context, id int, heat bool) error {
	if m.setDogHeat != nil {
		return m.setDogHeat(ctx, id, heat)
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
