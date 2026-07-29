package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func newTestPass(id int) *domain.Pass {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	return domain.MustNewPass(id, 10, 10, 100, domain.PassGeneric, 1, now, now, nil)
}

func TestModifyPassUseCase_Success_AppliesAllFields(t *testing.T) {
	t.Parallel()
	original := newTestPass(1)
	var saved *domain.Pass
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(ctx context.Context, pass *domain.Pass) error {
			saved = pass
			return nil
		},
	}
	uc := NewModifyPassUseCase(repo)

	newPrice := 15000
	newType := domain.PassSpecial
	newExpiry := time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC)
	patch := domain.PassPatch{
		Price:     &newPrice,
		PassType:  &newType,
		ExpiresAt: &newExpiry,
	}
	in := MustNewModifyPassInput(1, patch)
	output, err := uc.Execute(context.Background(), in)

	assert.NoError(t, err)
	assert.Equal(t, 15000, output.Pass.Price())
	assert.Equal(t, domain.PassSpecial, output.Pass.Type())
	assert.Equal(t, &newExpiry, output.Pass.ExpiresAt())
	assert.NotNil(t, saved, "update should be called on non-empty patch")
	assert.Equal(t, 15000, saved.Price())
}

func TestModifyPassUseCase_Success_EmptyPatchIsNoOp(t *testing.T) {
	t.Parallel()
	original := newTestPass(1)
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(context.Context, *domain.Pass) error {
			t.Fatal("update should not be called on empty patch")
			return nil
		},
	}
	uc := NewModifyPassUseCase(repo)
	in := MustNewModifyPassInput(1, domain.PassPatch{})
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, 100, output.Pass.Price())
	assert.Equal(t, domain.PassGeneric, output.Pass.Type())
}

func TestModifyPassUseCase_NotFound(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, nil
		},
		update: func(context.Context, *domain.Pass) error {
			t.Fatal("update should not be called when pass is missing")
			return nil
		},
	}
	uc := NewModifyPassUseCase(repo)
	in := MustNewModifyPassInput(99, domain.PassPatch{})
	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNewModifyPassInput_InvalidID(t *testing.T) {
	t.Parallel()
	for _, id := range []int{0, -1} {
		_, err := NewModifyPassInput(id, domain.PassPatch{})
		assertValidationError(t, err, "id")
	}
}

func TestModifyPassUseCase_PatchValidationErrors(t *testing.T) {
	t.Parallel()
	original := newTestPass(1)
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(context.Context, *domain.Pass) error {
			t.Fatal("update should not be called on patch validation error")
			return nil
		},
	}
	uc := NewModifyPassUseCase(repo)

	negativePrice := -1
	invalidType := domain.PassType("INVALID")
	zeroTime := time.Time{}

	tests := []struct {
		name      string
		patch     domain.PassPatch
		wantField string
	}{
		{"negative_price", domain.PassPatch{Price: &negativePrice}, "price"},
		{"invalid_type", domain.PassPatch{PassType: &invalidType}, "pass_type"},
		{"zero_expires_at", domain.PassPatch{ExpiresAt: &zeroTime}, "expires_at"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := MustNewModifyPassInput(1, tt.patch)
			_, err := uc.Execute(context.Background(), in)
			assertValidationError(t, err, tt.wantField)
		})
	}

	// After the failed patches, the original pass is still intact.
	assert.Equal(t, 100, original.Price())
	assert.Equal(t, domain.PassGeneric, original.Type())
}

func TestModifyPassUseCase_NonEditableFieldsUnchanged(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	original := domain.MustNewPass(42, 10, 10, 100, domain.PassGeneric, 7, now, now, nil)
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return original, nil
		},
		update: func(ctx context.Context, pass *domain.Pass) error {
			return nil
		},
	}
	uc := NewModifyPassUseCase(repo)

	newPrice := 200
	in := MustNewModifyPassInput(42, domain.PassPatch{Price: &newPrice})
	_, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, 200, original.Price())
	assert.Equal(t, 42, original.ID())
	assert.Equal(t, 10, original.NumOfSessions())
	assert.Equal(t, 10, original.RemainingSessions())
	assert.Equal(t, 7, original.UserID())
	assert.Equal(t, now, original.CreatedAt())
}

func TestModifyPassUseCase_RepoError_OnGet(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return nil, sentinelErr
		},
	}
	uc := NewModifyPassUseCase(repo)
	in := MustNewModifyPassInput(1, domain.PassPatch{})
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "get pass 1")
}

func TestModifyPassUseCase_RepoError_OnUpdate(t *testing.T) {
	t.Parallel()
	repo := &mockPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return newTestPass(1), nil
		},
		update: func(ctx context.Context, pass *domain.Pass) error {
			return sentinelErr
		},
	}
	uc := NewModifyPassUseCase(repo)
	newPrice := 200
	in := MustNewModifyPassInput(1, domain.PassPatch{Price: &newPrice})
	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "update pass 1")
}
