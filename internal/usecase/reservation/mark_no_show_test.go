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

// markPastActivity returns an activity anchored at fixedNow - 24h,
// so the "activity has started" check always passes.
func markPastActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-24*time.Hour))
}

// markFutureActivity returns an activity anchored at fixedNow + 7d,
// so the "activity has started" check always fails.
func markFutureActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(7*24*time.Hour))
}

// confirmedReservation returns a confirmed reservation pointing at
// the given activity/dog/pass.
func confirmedReservation(id, activityID, dogID, passID int) *domain.Reservation {
	return mustNewReservation(id, activityID, dogID, passID, domain.StatusConfirmed, fixedNow)
}

// newMarkUseCase wires the use case with default no-op mocks.
func newMarkUseCase(
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	reservationRepo domain.ReservationRepository,
	transactor Transactor,
) *MarkReservationNoShowUseCase {
	if transactor == nil {
		transactor = &stubTransactor{}
	}
	return NewMarkReservationNoShowUseCase(transactor, activityRepo, dogRepo, reservationRepo, func() time.Time { return fixedNow })
}

func validMarkInput() MarkReservationNoShowInput {
	return MustNewMarkReservationNoShowInput(1, 99, func() time.Time { return fixedNow })
}

func TestNewMarkReservationNoShowInput(t *testing.T) {
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
			_, err := NewMarkReservationNoShowInput(s.user, s.resID, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, s.field, verr.Field)
		})
	}
}

func TestMarkReservationNoShow_Success_ActivityStarted(t *testing.T) {
	userID := 1
	activity := markPastActivity(10)
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
			assert.Equal(t, domain.StatusNoShow, r.Status())
			return nil
		},
	}
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	output, err := uc.Execute(context.Background(), validMarkInput())
	require.NoError(t, err)
	require.NotNil(t, output.Reservation)
	assert.Equal(t, domain.StatusNoShow, output.Reservation.Status())
}

func TestMarkReservationNoShow_NotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{} // not reached
	dogRepo := &stubDogRepository{}           // not reached
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return nil, nil
		},
	}
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	assert.ErrorIs(t, err, ErrInvalidReservation)
}

func TestMarkReservationNoShow_ActivityNotFound(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, domain.ErrNotFound
		},
	}
	dogRepo := &stubDogRepository{} // not reached
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmedReservation(99, 10, 20, 30), nil
		},
	}
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestMarkReservationNoShow_ActivityNotYetStarted(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return markFutureActivity(10), nil
		},
	}
	dogRepo := &stubDogRepository{} // not reached (load dog is after the activity check)
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) {
			return confirmedReservation(99, 10, 20, 30), nil
		},
		update: func(context.Context, *domain.Reservation) error {
			t.Fatal("update must not be called when the activity has not started")
			return nil
		},
	}
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	assert.ErrorIs(t, err, ErrActivityNotStarted)
}

func TestMarkReservationNoShow_AlreadyNotCancellable(t *testing.T) {
	// Try each non-Confirmed status. The use case should translate
	// the domain's "cannot mark no-show, current status is X"
	// error into ErrNotCancellable.
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
				getByID: func(context.Context, int) (*domain.Activity, error) { return markPastActivity(10), nil },
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
			uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
			_, err := uc.Execute(context.Background(), validMarkInput())
			assert.ErrorIs(t, err, ErrNotCancellable)
		})
	}
}

func TestMarkReservationNoShow_DogNotOwnedByUser(t *testing.T) {
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return markPastActivity(10), nil },
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
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	assert.ErrorIs(t, err, ErrInvalidDog, "should not leak dog ownership")
}

func TestMarkReservationNoShow_NoPassUpdateOrMovement(t *testing.T) {
	// The use case must NOT call the pass repo at all: no
	// ConsumeSession, no RefundSession, no AddMovement. The
	// stub for the pass repo is intentionally nil (we use a
	// typed-nil trap) — but the cleaner way is to just not pass
	// a pass repo, since the use case does not depend on one.
	userID := 1
	activity := markPastActivity(10)
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
			assert.Equal(t, domain.StatusNoShow, r.Status())
			return nil
		},
	}
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	require.NoError(t, err)
}

func TestMarkReservationNoShow_ReservationRepoErrorIsWrapped(t *testing.T) {
	repoErr := errors.New("db connection lost")
	reservationRepo := &mockReservationRepository{
		getByID: func(context.Context, int) (*domain.Reservation, error) { return nil, repoErr },
	}
	activityRepo := &stubActivityRepository{} // not reached
	dogRepo := &stubDogRepository{}           // not reached
	uc := newMarkUseCase(activityRepo, dogRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validMarkInput())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
	assert.Contains(t, err.Error(), "get reservation")
}
