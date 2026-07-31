package incompatibility

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewListIncompatibilitiesInput_InvalidLevel(t *testing.T) {
	t.Parallel()
	bad := domain.IncompatibilityLevel("OTHER")
	_, err := NewListIncompatibilitiesInput(&bad, nil)
	assert.Error(t, err)
	var verr *ValidationError
	assert.True(t, errors.As(err, &verr))
	assert.Equal(t, "level", verr.Field)
}

func TestListIncompatibilitiesUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("list_all_no_filter", func(t *testing.T) {
		incompats := []*domain.Incompatibility{
			mustNewIncompatibility(1, "A", domain.IncompatibilityLevelBaja),
			mustNewIncompatibility(2, "B", domain.IncompatibilityLevelMedia),
		}
		var capturedLevel *domain.IncompatibilityLevel
		mock := &mockIncompatibilityRepository{
			list: func(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error) {
				capturedLevel = level
				return incompats, nil
			},
		}
		uc := NewListIncompatibilitiesUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewListIncompatibilitiesInput(nil, nil))
		assert.NoError(t, err)
		assert.Len(t, out.Incompatibilities, 2)
		assert.Nil(t, capturedLevel, "no level filter should be passed to repo")
	})

	t.Run("list_filtered_by_level", func(t *testing.T) {
		var capturedLevel *domain.IncompatibilityLevel
		mock := &mockIncompatibilityRepository{
			list: func(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error) {
				capturedLevel = level
				return nil, nil
			},
		}
		uc := NewListIncompatibilitiesUseCase(mock)
		media := domain.IncompatibilityLevelMedia
		_, err := uc.Execute(context.Background(), MustNewListIncompatibilitiesInput(&media, nil))
		assert.NoError(t, err)
		assert.NotNil(t, capturedLevel)
		assert.Equal(t, domain.IncompatibilityLevelMedia, *capturedLevel)
	})

	t.Run("empty_list", func(t *testing.T) {
		mock := &mockIncompatibilityRepository{
			list: func(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error) {
				return []*domain.Incompatibility{}, nil
			},
		}
		uc := NewListIncompatibilitiesUseCase(mock)
		out, err := uc.Execute(context.Background(), MustNewListIncompatibilitiesInput(nil, nil))
		assert.NoError(t, err)
		assert.Empty(t, out.Incompatibilities)
	})

	t.Run("repo_error", func(t *testing.T) {
		repoErr := errors.New("db timeout")
		mock := &mockIncompatibilityRepository{
			list: func(ctx context.Context, level *domain.IncompatibilityLevel, kind *domain.IncompatibilityKind) ([]*domain.Incompatibility, error) {
				return nil, repoErr
			},
		}
		uc := NewListIncompatibilitiesUseCase(mock)
		_, err := uc.Execute(context.Background(), MustNewListIncompatibilitiesInput(nil, nil))
		assert.True(t, errors.Is(err, repoErr))
	})
}
