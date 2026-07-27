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

func TestNewListByOwnerInput(t *testing.T) {
	scenarios := []struct {
		name    string
		ownerID int
	}{
		{"zero_owner_id", 0},
		{"negative_owner_id", -1},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := NewListByOwnerInput(s.ownerID, 10, 0)
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "owner_id", verr.Field)
		})
	}
}

func TestListByOwnerUseCase_Execute(t *testing.T) {
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
		out, err := uc.Execute(context.Background(), MustNewListByOwnerInput(42, 20, 5))
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
		out, err := uc.Execute(context.Background(), MustNewListByOwnerInput(42, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByOwnerInput(42, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByOwnerInput(42, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByOwnerInput(42, 10000, 0))
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
		out, err := uc.Execute(context.Background(), MustNewListAllDogsInput(10, 0))
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
		out, err := uc.Execute(context.Background(), MustNewListAllDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListAllDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListAllDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListAllDogsInput(10000, -5))
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
		out, err := uc.Execute(context.Background(), MustNewListActiveDogsInput(20, 0))
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
		out, err := uc.Execute(context.Background(), MustNewListActiveDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListActiveDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListActiveDogsInput(0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListActiveDogsInput(10000, 0))
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNewListByIncompatibilityInput(t *testing.T) {
	scenarios := []struct {
		name   string
		incomp int
	}{
		{"zero_incompatibility_id", 0},
		{"negative_incompatibility_id", -1},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := NewListByIncompatibilityInput(s.incomp, 10, 0)
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, "incompatibility_id", verr.Field)
		})
	}
}

func TestListByIncompatibilityUseCase_Execute(t *testing.T) {
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
		out, err := uc.Execute(context.Background(), MustNewListByIncompatibilityInput(7, 30, 10))
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
		out, err := uc.Execute(context.Background(), MustNewListByIncompatibilityInput(7, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByIncompatibilityInput(7, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByIncompatibilityInput(7, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByIncompatibilityInput(7, 10000, 0))
		assert.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNormalizePagination(t *testing.T) {
	t.Run("defaults_when_zero", func(t *testing.T) {
		limit, offset := normalizePagination(0, 0)
		assert.Equal(t, 50, limit)
		assert.Equal(t, 0, offset)
	})

	t.Run("negative_offset_clamps_to_zero", func(t *testing.T) {
		_, offset := normalizePagination(10, -5)
		assert.Equal(t, 0, offset)
	})

	t.Run("limit_caps_at_max", func(t *testing.T) {
		limit, _ := normalizePagination(10000, 0)
		assert.Equal(t, 100, limit)
	})

	t.Run("valid_values_pass_through", func(t *testing.T) {
		limit, offset := normalizePagination(25, 50)
		assert.Equal(t, 25, limit)
		assert.Equal(t, 50, offset)
	})
}

func TestNewListByBreedInput(t *testing.T) {
	_, err := NewListByBreedInput("", 10, 0)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "breed", verr.Field)
}

func TestListByBreedUseCase_Execute(t *testing.T) {
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
		out, err := uc.Execute(context.Background(), MustNewListByBreedInput("Labrador", 20, 5))
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
		out, err := uc.Execute(context.Background(), MustNewListByBreedInput("X", 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByBreedInput("X", 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByBreedInput("X", 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByBreedInput("X", 9999, 0))
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNewListBySexInput(t *testing.T) {
	// empty sex is invalid (zero value)
	_, err := NewListBySexInput(domain.Sex(""), 10, 0)
	assert.Error(t, err)
	// invalid sex value
	_, err = NewListBySexInput(domain.Sex("UNKNOWN"), 10, 0)
	assert.Error(t, err)
}

func TestListBySexUseCase_Execute(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var capturedSex domain.Sex
		mock := &mockDogRepository{
			listBySex: func(ctx context.Context, sex domain.Sex, limit, offset int) ([]*domain.Dog, error) {
				capturedSex = sex
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListBySexUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewListBySexInput(domain.SexFemale, 10, 0))
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
		out, _ := uc.Execute(context.Background(), MustNewListBySexInput(domain.SexMale, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListBySexInput(domain.SexMale, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListBySexInput(domain.SexMale, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListBySexInput(domain.SexMale, 9999, 0))
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
		out, err := uc.Execute(context.Background(), MustNewListByNeuteredInput(true, 10, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByNeuteredInput(false, 0, 0))
		assert.False(t, capturedNeutered)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByNeutered: func(ctx context.Context, neutered bool, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByNeuteredUseCase(mock)
		out, _ := uc.Execute(context.Background(), MustNewListByNeuteredInput(false, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByNeuteredInput(false, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByNeuteredInput(false, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByNeuteredInput(false, 9999, 0))
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
		out, err := uc.Execute(context.Background(), MustNewListByHeatInput(true, 10, 0))
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
		out, _ := uc.Execute(context.Background(), MustNewListByHeatInput(false, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByHeatInput(false, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByHeatInput(false, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByHeatInput(false, 9999, 0))
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNewListByAgeBracketInput(t *testing.T) {
	// empty bracket
	_, err := NewListByAgeBracketInput(domain.AgeBracket(""), 10, 0)
	assert.Error(t, err)
	// invalid bracket
	_, err = NewListByAgeBracketInput(domain.AgeBracket("BOGUS"), 10, 0)
	assert.Error(t, err)
}

func TestListByAgeBracketUseCase_Execute(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var capturedBracket domain.AgeBracket
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketTeenager, 10, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketUnknown, 0, 0))
		assert.Equal(t, domain.AgeBracketUnknown, capturedBracket)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listByAgeBracket: func(ctx context.Context, bracket domain.AgeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListByAgeBracketUseCase(mock)
		out, _ := uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketAdult, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketAdult, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketAdult, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListByAgeBracketInput(domain.AgeBracketAdult, 9999, 0))
		assert.Equal(t, 100, capturedLimit)
	})
}

func TestNewListBySizeBracketInput(t *testing.T) {
	// empty bracket
	_, err := NewListBySizeBracketInput(domain.SizeBracket(""), 10, 0)
	assert.Error(t, err)
	// invalid bracket
	_, err = NewListBySizeBracketInput(domain.SizeBracket("BOGUS"), 10, 0)
	assert.Error(t, err)
}

func TestListBySizeBracketUseCase_Execute(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var capturedBracket domain.SizeBracket
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				capturedBracket = bracket
				return []*domain.Dog{newTestDogForList(1)}, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketLarge, 10, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketUnknown, 0, 0))
		assert.Equal(t, domain.SizeBracketUnknown, capturedBracket)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockDogRepository{
			listBySizeBracket: func(ctx context.Context, bracket domain.SizeBracket, limit, offset int) ([]*domain.Dog, error) {
				return nil, nil
			},
		}
		uc := NewListBySizeBracketUseCase(mock)
		out, _ := uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketMini, 0, 0))
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
		_, err := uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketMini, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketMini, 0, 0))
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
		_, _ = uc.Execute(context.Background(), MustNewListBySizeBracketInput(domain.SizeBracketMini, 9999, 0))
		assert.Equal(t, 100, capturedLimit)
	})
}
