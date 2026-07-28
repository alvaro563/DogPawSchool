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

// completeFinishedActivity returns an activity that started at
// fixedNow - 25h with duration 1h, so it ended at fixedNow - 24h.
// The "activity finished" check always passes.
func completeFinishedActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-25*time.Hour))
}

// completeOngoingActivity returns an activity that started at
// fixedNow - 30min with duration 2h, so it ends at fixedNow + 1.5h.
// The "activity finished" check always fails.
func completeOngoingActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 2, fixedNow.Add(-30*time.Minute))
}

// newCompleteUseCase wires the use case with default no-op mocks.
func newCompleteUseCase(
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	transactor Transactor,
) *CompleteReservationUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewCompleteReservationUseCase(transactor, activityRepo, dogRepo, reservationRepo)
}

func validCompleteInput() CompleteReservationInput {
	return MustNewCompleteReservationInput(1, 99, func() time.Time { return fixedNow })
}

func TestNewCompleteReservationInput(t *testing.T) {
	scenarios := []struct {
		name  string
		user  int
		resID int
		field string
	}{
		{"zero_user_id", 0, 99, "user_id"},
		{"negative_user_id", -1, 99, "user_id"},
		{"zero_reservation_id", 1, 0, "reservation_id"},
		{"negative_reservation_id", 1, -5, "reservation_id"},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := NewCompleteReservationInput(s.user, s.resID, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, s.field, verr.Field)
		})
	}
}

func TestCompleteReservationInput_NilNow(t *testing.T) {
	in, err := NewCompleteReservationInput(1, 99, nil)
	require.NoError(t, err)
	assert.False(t, in.Now().IsZero(), "nil now should default to time.Now")
}

func TestCompleteReservationUseCase_Success_ActivityFinished(t *testing.T) {
	userID := 1
	activity := completeFinishedActivity(10)
	dog := validDog(20, userID)
	reservation := confirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(_ context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 10, id)
			return activity, nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			assert.Equal(t, 20, id)
			return dog, nil
		},
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(_ context.Context, id int) (*domain.Reservation, error) {
			assert.Equal(t, 99, id)
			return reservation, nil
		},
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCompleted, r.Status())
			return nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validCompleteInput())
	require.NoError(t, err)
	require.NotNil(t, output.Reservation)
	assert.Equal(t, domain.StatusCompleted, output.Reservation.Status())
}

func TestCompleteReservationUseCase_NotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{}
	dogRepo := &stubDogRepository{}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	assert.ErrorIs(t, err, ErrInvalidReservation)
}

func TestCompleteReservationUseCase_ActivityNotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, domain.ErrNotFound
		},
	}
	dogRepo := &stubDogRepository{}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmedReservation(99, 10, 20, 30), nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestCompleteReservationUseCase_ActivityNotFinished(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return completeOngoingActivity(10), nil
		},
	}
	dogRepo := &stubDogRepository{}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmedReservation(99, 10, 20, 30), nil
		},
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("update must not be called when the activity has not finished")
			return nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	assert.ErrorIs(t, err, ErrActivityNotFinished)
}

func TestCompleteReservationUseCase_AlreadyNotCompletable(t *testing.T) {
	statuses := []domain.ReservationStatus{
		domain.StatusCancelledInTime,
		domain.StatusCancelledLate,
		domain.StatusCompleted,
		domain.StatusForgiven,
		domain.StatusNoShow,
	}
	for _, st := range statuses {
		t.Run(string(st), func(t *testing.T) {
			r := mustNewReservation(99, 10, 20, 30, st, fixedNow)
			activityRepo := &stubActivityRepository{
				getByID: func(context.Context, int) (*domain.Activity, error) { return completeFinishedActivity(10), nil },
			}
			dogRepo := &stubDogRepository{
				getByID: func(context.Context, int) (*domain.Dog, error) { return validDog(20, 1), nil },
			}
			reservationRepo := &mockReservationRepository{
				getByID: func(context.Context, int) (*domain.Reservation, error) { return r, nil },
				update: func(context.Context, *domain.Reservation) error {
					t.Fatal("update must not be called on non-CONFIRMED reservation")
					return nil
				},
			}
			uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
			_, err := uc.Execute(context.Background(), validCompleteInput())
			assert.ErrorIs(t, err, ErrNotCompletable)
		})
	}
}

func TestCompleteReservationUseCase_DogNotOwnedByUser(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return completeFinishedActivity(10), nil },
	}
	// Dog belongs to user 99, but the request is for user 1.
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return validDog(20, 99), nil },
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmedReservation(99, 10, 20, 30), nil
		},
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("update must not be called when dog is owned by another user")
			return nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	assert.ErrorIs(t, err, ErrInvalidDog, "should not leak dog ownership")
}

func TestCompleteReservationUseCase_NoPassUpdate(t *testing.T) {
	userID := 1
	activity := completeFinishedActivity(10)
	dog := validDog(20, userID)
	reservation := confirmedReservation(99, 10, 20, 30)

	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	dogRepo := &stubDogRepository{
		getByID: func(context.Context, int) (*domain.Dog, error) { return dog, nil },
	}
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return reservation, nil },
		update: func(_ context.Context, r *domain.Reservation) error {
			assert.Equal(t, domain.StatusCompleted, r.Status())
			return nil
		},
	}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	require.NoError(t, err)
}

func TestCompleteReservationUseCase_ReservationRepoErrorIsWrapped(t *testing.T) {
	repoErr := errors.New("db connection lost")
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return nil, repoErr },
	}
	activityRepo := &stubActivityRepository{}
	dogRepo := &stubDogRepository{}
	uc := newCompleteUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validCompleteInput())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
	assert.Contains(t, err.Error(), "get reservation")
}
