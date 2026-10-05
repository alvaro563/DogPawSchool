package activity

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewDeleteActivityInput(t *testing.T) {
	t.Parallel()
	_, err := NewDeleteActivityInput(0)
	assertValidationError(t, err, "id")
	_, err = NewDeleteActivityInput(-3)
	assertValidationError(t, err, "id")
	in, err := NewDeleteActivityInput(7)
	assert.NoError(t, err)
	assert.Equal(t, 7, in.ID())
}

func TestDeleteActivityUseCase_Success(t *testing.T) {
	t.Parallel()
	var deletedID int
	repo := &mockActivityRepository{
		delete: func(ctx context.Context, id int) error {
			deletedID = id
			return nil
		},
	}
	uc := NewDeleteActivityUseCase(repo)

	_, err := uc.Execute(context.Background(), MustNewDeleteActivityInput(15))

	assert.NoError(t, err)
	assert.Equal(t, 15, deletedID)
}

func TestDeleteActivityUseCase_NotFound(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		delete: func(ctx context.Context, id int) error {
			return domain.ErrNotFound
		},
	}
	uc := NewDeleteActivityUseCase(repo)

	_, err := uc.Execute(context.Background(), MustNewDeleteActivityInput(404))

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteActivityUseCase_HasReservations(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		delete: func(ctx context.Context, id int) error {
			return domain.ErrActivityHasReservations
		},
	}
	uc := NewDeleteActivityUseCase(repo)

	_, err := uc.Execute(context.Background(), MustNewDeleteActivityInput(3))

	assert.ErrorIs(t, err, ErrActivityHasReservations)
	assert.ErrorIs(t, err, domain.ErrActivityHasReservations)
}

func TestDeleteActivityUseCase_UnexpectedError(t *testing.T) {
	t.Parallel()
	repo := &mockActivityRepository{
		delete: func(ctx context.Context, id int) error {
			return errors.New("connection lost")
		},
	}
	uc := NewDeleteActivityUseCase(repo)

	_, err := uc.Execute(context.Background(), MustNewDeleteActivityInput(3))

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrActivityHasReservations)
}
