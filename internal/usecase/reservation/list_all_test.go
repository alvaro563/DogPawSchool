package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestNewListAllReservationsInput(t *testing.T) {
	t.Parallel()

	in, err := NewListAllReservationsInput(50, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 50, in.Limit())
	assert.Equal(t, 0, in.Offset())
	assert.Nil(t, in.Status())

	in, err = NewListAllReservationsInput(0, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 50, in.Limit())
	assert.Equal(t, 0, in.Offset())

	in, err = NewListAllReservationsInput(200, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 100, in.Limit())

	t.Run("status_filter_accepted", func(t *testing.T) {
		st := domain.StatusCancelledLate
		in, err := NewListAllReservationsInput(50, 0, &st)
		require.NoError(t, err)
		require.NotNil(t, in.Status())
		assert.Equal(t, domain.StatusCancelledLate, *in.Status())
	})

	t.Run("invalid_status_rejected", func(t *testing.T) {
		bad := domain.ReservationStatus("NOPE")
		_, err := NewListAllReservationsInput(50, 0, &bad)
		require.Error(t, err)
		var verr *ValidationError
		require.True(t, errors.As(err, &verr))
		assert.Equal(t, "status", verr.Field)
	})
}

func TestListAllReservationsUseCase_Execute(t *testing.T) {
	t.Parallel()

	future := fixedNow.Add(72 * time.Hour)
	view1 := mustNewReservationView(1, 10, 20, 1, 5, 1, domain.StatusConfirmed, fixedNow, "Paseo", "Central", future, "Luna", 10)
	view2 := mustNewReservationView(2, 11, 21, 2, 6, 2, domain.StatusConfirmed, fixedNow, "Ruta", "Rio", future, "Toby", 8)

	t.Run("returns_views", func(t *testing.T) {
		t.Parallel()
		expected := []*domain.ReservationView{view1, view2}
		repo := &mockReservationRepository{
			listAllView: func(_ context.Context, _, _ int, _ *domain.ReservationStatus) ([]*domain.ReservationView, error) {
				return expected, nil
			},
		}
		uc := NewListAllReservationsUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListAllReservationsInput(50, 0, nil))
		require.NoError(t, err)
		assert.Equal(t, expected, out.Views)
	})

	t.Run("forwards_status_to_repo", func(t *testing.T) {
		t.Parallel()
		st := domain.StatusCancelledLate
		var captured *domain.ReservationStatus
		repo := &mockReservationRepository{
			listAllView: func(_ context.Context, _, _ int, got *domain.ReservationStatus) ([]*domain.ReservationView, error) {
				captured = got
				return nil, nil
			},
		}
		uc := NewListAllReservationsUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewListAllReservationsInput(50, 0, &st))
		require.NoError(t, err)
		require.NotNil(t, captured)
		assert.Equal(t, domain.StatusCancelledLate, *captured)
	})

	t.Run("empty_list", func(t *testing.T) {
		t.Parallel()
		repo := &mockReservationRepository{
			listAllView: func(_ context.Context, _, _ int, _ *domain.ReservationStatus) ([]*domain.ReservationView, error) {
				return nil, nil
			},
		}
		uc := NewListAllReservationsUseCase(repo)
		out, err := uc.Execute(context.Background(), MustNewListAllReservationsInput(50, 0, nil))
		require.NoError(t, err)
		assert.Empty(t, out.Views)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()
		repoErr := errors.New("db failure")
		repo := &mockReservationRepository{
			listAllView: func(_ context.Context, _, _ int, _ *domain.ReservationStatus) ([]*domain.ReservationView, error) {
				return nil, repoErr
			},
		}
		uc := NewListAllReservationsUseCase(repo)
		_, err := uc.Execute(context.Background(), MustNewListAllReservationsInput(50, 0, nil))
		assert.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})
}
