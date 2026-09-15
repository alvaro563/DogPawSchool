package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

// mockDogWithOwnerLister is a local mock for the narrow port used by
// the new use case. It does NOT touch mockDogRepository from
// helpers_test.go — keeping the existing DogRepository mock intact
// (Open/Closed).
type mockDogWithOwnerLister struct {
	listActiveWithOwner func(ctx context.Context, limit, offset int) ([]*DogWithOwner, error)
}

func (m *mockDogWithOwnerLister) ListActiveWithOwner(ctx context.Context, limit, offset int) ([]*DogWithOwner, error) {
	if m.listActiveWithOwner != nil {
		return m.listActiveWithOwner(ctx, limit, offset)
	}
	return nil, nil
}

func TestListActiveDogsWithOwnerUseCase_Execute_HappyPath(t *testing.T) {
	dog, err := domain.NewDog(20, "Test", "Breed", "Pass", 24, domain.SexMale, 10.0, 1)
	require.NoError(t, err)
	repo := &mockDogWithOwnerLister{
		listActiveWithOwner: func(_ context.Context, limit, offset int) ([]*DogWithOwner, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 0, offset)
			return []*DogWithOwner{{Dog: dog, OwnerName: "María"}}, nil
		},
	}
	uc := NewListActiveDogsWithOwnerUseCase(repo)
	in, err := NewListActiveDogsWithOwnerInput(50, 0)
	require.NoError(t, err)

	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.Len(t, out.Items, 1)
	assert.Equal(t, "María", out.Items[0].OwnerName)
	assert.Same(t, dog, out.Items[0].Dog)
}

func TestListActiveDogsWithOwnerUseCase_Execute_RepoError(t *testing.T) {
	want := errors.New("db down")
	repo := &mockDogWithOwnerLister{
		listActiveWithOwner: func(_ context.Context, _ int, _ int) ([]*DogWithOwner, error) {
			return nil, want
		},
	}
	uc := NewListActiveDogsWithOwnerUseCase(repo)
	in, err := NewListActiveDogsWithOwnerInput(10, 0)
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), in)
	require.Error(t, err)
	assert.ErrorIs(t, err, want)
}

func TestListActiveDogsWithOwnerUseCase_Execute_Empty(t *testing.T) {
	repo := &mockDogWithOwnerLister{
		listActiveWithOwner: func(_ context.Context, _ int, _ int) ([]*DogWithOwner, error) {
			return []*DogWithOwner{}, nil
		},
	}
	uc := NewListActiveDogsWithOwnerUseCase(repo)
	in, err := NewListActiveDogsWithOwnerInput(10, 0)
	require.NoError(t, err)

	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	assert.Empty(t, out.Items)
}

// Compile-time check: the use case struct still satisfies domain
// interface expectations without leaking infra types.
var _ DogWithOwnerLister = (*mockDogWithOwnerLister)(nil)

// Sanity: ensure the existing domain.DogRepository interface is not
// affected — this test would fail to compile if a future change
// accidentally widened the contract.
var _ domain.DogRepository = (*mockDogRepository)(nil)
