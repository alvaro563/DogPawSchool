package pass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func validRegisterInput() RegisterPassInput {
	return RegisterPassInput{
		NumOfSessions: 10,
		Price:         12000,
		PassType:      domain.PassGeneric,
		UserID:        1,
		ExpiresAt:     nil,
	}
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
	output, err := uc.Execute(context.Background(), RegisterPassInput{
		NumOfSessions: 5,
		Price:         5000,
		PassType:      domain.PassSpecial,
		UserID:        3,
		ExpiresAt:     &expiry,
	})
	assert.NoError(t, err)
	assert.Equal(t, 7, output.ID)
}

func TestRegisterPassUseCase_ValidationErrors(t *testing.T) {
	base := validRegisterInput()
	tests := []struct {
		name      string
		mutate    func(input *RegisterPassInput)
		wantField string
	}{
		{
			name:      "zero_sessions",
			mutate:    func(i *RegisterPassInput) { i.NumOfSessions = 0 },
			wantField: "num_of_sessions",
		},
		{
			name:      "negative_sessions",
			mutate:    func(i *RegisterPassInput) { i.NumOfSessions = -3 },
			wantField: "num_of_sessions",
		},
		{
			name:      "negative_price",
			mutate:    func(i *RegisterPassInput) { i.Price = -1 },
			wantField: "price",
		},
		{
			name:      "invalid_type",
			mutate:    func(i *RegisterPassInput) { i.PassType = domain.PassType("WRONG") },
			wantField: "pass_type",
		},
		{
			name:      "zero_user_id",
			mutate:    func(i *RegisterPassInput) { i.UserID = 0 },
			wantField: "user_id",
		},
		{
			name:      "negative_user_id",
			mutate:    func(i *RegisterPassInput) { i.UserID = -1 },
			wantField: "user_id",
		},
		{
			name:      "zero_expires_at",
			mutate:    func(i *RegisterPassInput) { i.ExpiresAt = &time.Time{} },
			wantField: "expires_at",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)
			repo := &mockPassRepository{
				create: func(context.Context, *domain.Pass) (int, error) {
					t.Fatal("create should not be called on validation error")
					return 0, nil
				},
			}
			uc := NewRegisterPassUseCase(repo)
			_, err := uc.Execute(context.Background(), input)
			assertValidationError(t, err, tt.wantField)
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
