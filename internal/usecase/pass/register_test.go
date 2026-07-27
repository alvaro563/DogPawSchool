package pass

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterPassInput {
	return MustNewRegisterPassInput(10, 12000, domain.PassGeneric, 1, nil)
}

func TestRegisterPassUseCase_Success(t *testing.T) {
	repo := &mockPassRepository{
		create: func(ctx context.Context, pass *domain.Pass) (int, error) {
			assert.Equal(t, 10, pass.NumOfSessions())
			assert.Equal(t, 12000, pass.Price())
			assert.Equal(t, domain.PassGeneric, pass.Type())
			assert.Equal(t, 1, pass.UserID())
			assert.Equal(t, 10, pass.RemainingSessions(), "new pass should start fully available")
			return 42, nil
		},
	}
	uc := NewRegisterPassUseCase(repo)

	output, err := uc.Execute(context.Background(), validRegisterInput())

	assert.NoError(t, err)
	assert.Equal(t, 42, output.ID)
}

func TestRegisterPassUseCase_SuccessWithExpiry(t *testing.T) {
	expiry := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	repo := &mockPassRepository{
		create: func(ctx context.Context, pass *domain.Pass) (int, error) {
			if pass.ExpiresAt() == nil {
				t.Fatal("expected non-nil ExpiresAt")
			}
			assert.Equal(t, expiry, *pass.ExpiresAt())
			return 7, nil
		},
	}
	uc := NewRegisterPassUseCase(repo)
	in := MustNewRegisterPassInput(5, 5000, domain.PassSpecial, 3, &expiry)
	output, err := uc.Execute(context.Background(), in)
	assert.NoError(t, err)
	assert.Equal(t, 7, output.ID)
}

func TestNewRegisterPassInput(t *testing.T) {
	scenarios := []struct {
		name    string
		factory func() (RegisterPassInput, error)
		field   string
	}{
		{"zero_sessions", func() (RegisterPassInput, error) { return NewRegisterPassInput(0, 100, domain.PassGeneric, 1, nil) }, "num_of_sessions"},
		{"negative_sessions", func() (RegisterPassInput, error) { return NewRegisterPassInput(-3, 100, domain.PassGeneric, 1, nil) }, "num_of_sessions"},
		{"negative_price", func() (RegisterPassInput, error) { return NewRegisterPassInput(10, -1, domain.PassGeneric, 1, nil) }, "price"},
		{"invalid_type", func() (RegisterPassInput, error) {
			return NewRegisterPassInput(10, 100, domain.PassType("WRONG"), 1, nil)
		}, "pass_type"},
		{"zero_user_id", func() (RegisterPassInput, error) { return NewRegisterPassInput(10, 100, domain.PassGeneric, 0, nil) }, "user_id"},
		{"negative_user_id", func() (RegisterPassInput, error) { return NewRegisterPassInput(10, 100, domain.PassGeneric, -1, nil) }, "user_id"},
		{"zero_expires_at", func() (RegisterPassInput, error) {
			zt := time.Time{}
			return NewRegisterPassInput(10, 100, domain.PassGeneric, 1, &zt)
		}, "expires_at"},
	}
	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.factory()
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, tt.field, verr.Field)
		})
	}
}

func TestRegisterPassUseCase_RepoError(t *testing.T) {
	repo := &mockPassRepository{
		create: func(ctx context.Context, pass *domain.Pass) (int, error) {
			return 0, sentinelErr
		},
	}
	uc := NewRegisterPassUseCase(repo)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	assert.Error(t, err)
	assert.ErrorIs(t, err, sentinelErr)
	assert.Contains(t, err.Error(), "register pass")
}
