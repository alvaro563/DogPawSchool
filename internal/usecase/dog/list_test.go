package dog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestDogForList(id int) *domain.Dog {
	d, err := domain.NewDog(id, "Test", "Breed", "Pass", 24, domain.SexMale, 10.0, 1)
	if err != nil {
		panic(err)
	}
	return d
}

func TestListByOwnerUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListByOwnerInput
			expectedField string
		}{
			{"zero_owner_id", ListByOwnerInput{Limit: 10, Offset: 0}, "owner_id"},
			{"negative_owner_id", ListByOwnerInput{OwnerID: -1, Limit: 10, Offset: 0}, "owner_id"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListByOwnerUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedLimit, capturedOffset, capturedUserID int
		mock := &mockDogRepository{
			listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
				capturedUserID = userID
				capturedLimit = limit
				capturedOffset = offset
				return []*domain.Dog{newTestDogForList(1), newTestDogForList(2)}, nil
			},
		}
		uc := NewListByOwnerUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByOwnerInput{OwnerID: 42, Limit: 20, Offset: 5})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 2)
		assert.Equal(t, 42, capturedUserID)
		assert.Equal(t, 20, capturedLimit)
		assert.Equal(t, 5, capturedOffset)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByOwnerUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByOwnerInput{OwnerID: 42})
		assert.NoError(t, err)
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mock := &mockDogRepository{
			listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByOwnerUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByOwnerInput{OwnerID: 42})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults_when_zero", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByOwnerUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByOwnerInput{OwnerID: 42})
		assert.NoError(t, err)
		assert.Equal(t, 50, capturedLimit, "default page limit should be 50")
	})

	t.Run("pagination_caps_at_max", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByOwner: func(ctx context.Context, userID, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByOwnerUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByOwnerInput{OwnerID: 42, Limit: 10000})
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit, "limit should cap at 100")
	})
}

func TestListAllDogsUseCase_Execute(t *testing.T) {
	t.Run("happy_path_with_activeOnly_false", func(t *testing.T) {
		var capturedActiveOnly bool
		var capturedLimit, capturedOffset int
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedActiveOnly = activeOnly
				capturedLimit = limit
				capturedOffset = offset
				return []*domain.Dog{newTestDogForList(1), newTestDogForList(2), newTestDogForList(3)}, nil
			},
		}
		uc := NewListAllDogsUseCase(mock)
		out, err := uc.Execute(context.Background(), ListAllDogsInput{Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 3)
		assert.False(t, capturedActiveOnly, "ListAllDogsUseCase must pass activeOnly=false")
		assert.Equal(t, 10, capturedLimit)
		assert.Equal(t, 0, capturedOffset)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListAllDogsUseCase(mock)
		out, err := uc.Execute(context.Background(), ListAllDogsInput{})
		assert.NoError(t, err)
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListAllDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListAllDogsInput{})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListAllDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListAllDogsInput{})
		assert.NoError(t, err)
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListAllDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListAllDogsInput{Limit: 10000, Offset: -5})
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListActiveDogsUseCase_Execute(t *testing.T) {
	t.Run("happy_path_with_activeOnly_true", func(t *testing.T) {
		var capturedActiveOnly bool
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedActiveOnly = activeOnly
				return []*domain.Dog{newTestDogForList(10), newTestDogForList(11)}, nil
			},
		}
		uc := NewListActiveDogsUseCase(mock)
		out, err := uc.Execute(context.Background(), ListActiveDogsInput{Limit: 20, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 2)
		assert.True(t, capturedActiveOnly, "ListActiveDogsUseCase must pass activeOnly=true")
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListActiveDogsUseCase(mock)
		out, err := uc.Execute(context.Background(), ListActiveDogsInput{})
		assert.NoError(t, err)
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListActiveDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListActiveDogsInput{})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListActiveDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListActiveDogsInput{})
		assert.NoError(t, err)
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listAll: func(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListActiveDogsUseCase(mock)
		_, err := uc.Execute(context.Background(), ListActiveDogsInput{Limit: 10000})
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListByIncompatibilityUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListByIncompatibilityInput
			expectedField string
		}{
			{"zero_incompatibility_id", ListByIncompatibilityInput{Limit: 10, Offset: 0}, "incompatibility_id"},
			{"negative_incompatibility_id", ListByIncompatibilityInput{IncompatibilityID: -1, Limit: 10, Offset: 0}, "incompatibility_id"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListByIncompatibilityUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedIncompID, capturedLimit, capturedOffset int
		mock := &mockDogRepository{
			listByIncompatibility: func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
				capturedIncompID = incompatibilityID
				capturedLimit = limit
				capturedOffset = offset
				return []*domain.Dog{newTestDogForList(1), newTestDogForList(2)}, nil
			},
		}
		uc := NewListByIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByIncompatibilityInput{IncompatibilityID: 7, Limit: 30, Offset: 10})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 2)
		assert.Equal(t, 7, capturedIncompID)
		assert.Equal(t, 30, capturedLimit)
		assert.Equal(t, 10, capturedOffset)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByIncompatibility: func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByIncompatibilityUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByIncompatibilityInput{IncompatibilityID: 7})
		assert.NoError(t, err)
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mock := &mockDogRepository{
			listByIncompatibility: func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByIncompatibilityInput{IncompatibilityID: 7})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByIncompatibility: func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByIncompatibilityInput{IncompatibilityID: 7})
		assert.NoError(t, err)
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByIncompatibility: func(ctx context.Context, incompatibilityID, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByIncompatibilityUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByIncompatibilityInput{IncompatibilityID: 7, Limit: 10000})
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNormalizePagination(t *testing.T) {
	t.Run("defaults_when_zero", func(t *testing.T) {
		limit, offset := NormalizePagination(0, 0)
		assert.Equal(t, 50, limit)
		assert.Equal(t, 0, offset)
	})

	t.Run("negative_offset_clamps_to_zero", func(t *testing.T) {
		_, offset := NormalizePagination(10, -5)
		assert.Equal(t, 0, offset)
	})

	t.Run("limit_caps_at_max", func(t *testing.T) {
		limit, _ := NormalizePagination(10000, 0)
		assert.Equal(t, 100, limit)
	})

	t.Run("valid_values_pass_through", func(t *testing.T) {
		limit, offset := NormalizePagination(25, 50)
		assert.Equal(t, 25, limit)
		assert.Equal(t, 50, offset)
	})
}

func TestListByBreedUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListByBreedInput
			expectedField string
		}{
			{"empty_breed", ListByBreedInput{Limit: 10, Offset: 0}, "breed"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListByBreedUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedBreed string
		var capturedLimit, capturedOffset int
		mock := &mockDogRepository{
			listByBreed: func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
				capturedBreed = breed
				capturedLimit = limit
				capturedOffset = offset
				return []*domain.Dog{newTestDogForList(1), newTestDogForList(2)}, nil
			},
		}
		uc := NewListByBreedUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByBreedInput{Breed: "Labrador", Limit: 20, Offset: 5})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 2)
		assert.Equal(t, "Labrador", capturedBreed)
		assert.Equal(t, 20, capturedLimit)
		assert.Equal(t, 5, capturedOffset)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByBreed: func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByBreedUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByBreedInput{Breed: "X"})
		assert.NoError(t, err)
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listByBreed: func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByBreedUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByBreedInput{Breed: "X"})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByBreed: func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByBreedUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByBreedInput{Breed: "X"})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByBreed: func(ctx context.Context, breed string, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByBreedUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByBreedInput{Breed: "X", Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListBySexUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListBySexInput
			expectedField string
		}{
			{"empty_sex", ListBySexInput{Limit: 10, Offset: 0}, "sex"},
			{"invalid_sex", ListBySexInput{Sex: domain.Sex("UNKNOWN"), Limit: 10, Offset: 0}, "sex"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListBySexUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedSex domain.Sex
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				capturedSex = sex
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListBySexUseCase(mock)
		out, err := uc.Execute(context.Background(), ListBySexInput{Sex: domain.SexFemale, Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 1)
		assert.Equal(t, domain.SexFemale, capturedSex)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListBySexUseCase(mock)
		out, _ := uc.Execute(context.Background(), ListBySexInput{Sex: domain.SexMale})
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListBySexUseCase(mock)
		_, err := uc.Execute(context.Background(), ListBySexInput{Sex: domain.SexMale})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListBySexUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListBySexInput{Sex: domain.SexMale})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListBySexUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListBySexInput{Sex: domain.SexMale, Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListByNeuteredUseCase_Execute(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var capturedNeutered bool
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				capturedNeutered = neutered
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByNeuteredInput{Neutered: true, Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 1)
		assert.True(t, capturedNeutered)
	})

	t.Run("happy_path_false", func(t *testing.T) {
		var capturedNeutered bool
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				capturedNeutered = neutered
				return nil, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByNeuteredInput{Neutered: false})
		assert.False(t, capturedNeutered)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		out, _ := uc.Execute(context.Background(), ListByNeuteredInput{})
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByNeuteredInput{})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByNeuteredInput{})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByNeuteredInput{Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListByHeatUseCase_Execute(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var capturedHeat bool
		mock := &mockDogRepository{
			listByHeat: func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
				capturedHeat = heat
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListByHeatUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByHeatInput{Heat: true, Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 1)
		assert.True(t, capturedHeat)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByHeat: func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByHeatUseCase(mock)
		out, _ := uc.Execute(context.Background(), ListByHeatInput{})
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listByHeat: func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByHeatUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByHeatInput{})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByHeat: func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByHeatUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByHeatInput{})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByHeat: func(ctx context.Context, heat bool, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByHeatUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByHeatInput{Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListByAgeBracketUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListByAgeBracketInput
			expectedField string
		}{
			{"empty_bracket", ListByAgeBracketInput{Limit: 10, Offset: 0}, "age_bracket"},
			{"invalid_bracket", ListByAgeBracketInput{AgeBracket: domain.AgeBracket("BOGUS"), Limit: 10, Offset: 0}, "age_bracket"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListByAgeBracketUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedBracket domain.AgeBracket
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		out, err := uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketTeenager, Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 1)
		assert.Equal(t, domain.AgeBracketTeenager, capturedBracket)
	})

	t.Run("happy_path_unknown_bracket", func(t *testing.T) {
		var capturedBracket domain.AgeBracket
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return nil, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketUnknown})
		assert.Equal(t, domain.AgeBracketUnknown, capturedBracket)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		out, _ := uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketAdult})
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		_, err := uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketAdult})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketAdult})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListByAgeBracketInput{AgeBracket: domain.AgeBracketAdult, Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestListBySizeBracketUseCase_Execute(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		scenarios := []struct {
			name          string
			input         ListBySizeBracketInput
			expectedField string
		}{
			{"empty_bracket", ListBySizeBracketInput{Limit: 10, Offset: 0}, "size_bracket"},
			{"invalid_bracket", ListBySizeBracketInput{SizeBracket: domain.SizeBracket("BOGUS"), Limit: 10, Offset: 0}, "size_bracket"},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				mock := &mockDogRepository{}
				uc := NewListBySizeBracketUseCase(mock)
				_, err := uc.Execute(context.Background(), s.input)
				assert.Error(t, err)
				var verr *ValidationError
				assert.True(t, errors.As(err, &verr))
				assert.Equal(t, s.expectedField, verr.Field)
			})
		}
	})

	t.Run("happy_path", func(t *testing.T) {
		var capturedBracket domain.SizeBracket
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		out, err := uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketLarge, Limit: 10, Offset: 0})
		assert.NoError(t, err)
		assert.Len(t, out.Dogs, 1)
		assert.Equal(t, domain.SizeBracketLarge, capturedBracket)
	})

	t.Run("happy_path_unknown_bracket", func(t *testing.T) {
		var capturedBracket domain.SizeBracket
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return nil, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketUnknown})
		assert.Equal(t, domain.SizeBracketUnknown, capturedBracket)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		out, _ := uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketMini})
		assert.Empty(t, out.Dogs)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, repoErr
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		_, err := uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketMini})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr))
	})

	t.Run("pagination_defaults", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketMini})
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("pagination_caps", func(t *testing.T) {
		var capturedLimit int
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		_, _ = uc.Execute(context.Background(), ListBySizeBracketInput{SizeBracket: domain.SizeBracketMini, Limit: 9999})
		assert.Equal(t, 100, capturedLimit)
	})
}
