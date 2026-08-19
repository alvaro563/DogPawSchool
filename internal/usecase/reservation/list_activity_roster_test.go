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

// fixedOwner builds a deterministic user for the roster tests. The
// password is a 60-char placeholder so domain.NewUser accepts it.
func fixedOwner(id int, name string) *domain.User {
	u, err := domain.NewUser(id, name, name+"@test.com",
		"hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

// rosterView builds a reservation view anchored to a future activity
// (so the test is deterministic regardless of wall clock time). The
// owner is the dog's UserID() — the use case reads the owner id from
// view.DogUserID() exactly the way the SQL join does.
func rosterView(
	id int, activityID int, dogID int, ownerID int,
	status domain.ReservationStatus,
	createdAt time.Time, dogName string,
) *domain.ReservationView {
	return mustNewReservationView(
		id, activityID, dogID, ownerID,
		1, ownerID, // pass owned by the same owner (sanity)
		status, createdAt,
		"Paseo", "Central", fixedNow.Add(7*24*time.Hour),
		dogName, 5,
	)
}

// ── Input validation ──────────────────────────────────────────────

func TestNewListActivityRosterInput_RejectsZeroOrNegative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		activityID int
	}{
		{"zero", 0},
		{"negative", -7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewListActivityRosterInput(tt.activityID)
			assertValidationError(t, err, "activity_id")
		})
	}
}

// ── Activity not found ────────────────────────────────────────────

func TestListActivityRosterUseCase_ActivityNotFound(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return nil, domain.ErrNotFound },
	}
	uc := NewListActivityRosterUseCase(activityRepo, &mockReservationRepository{}, &stubUserRepository{})
	_, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	assert.ErrorIs(t, err, ErrInvalidActivity)
}

func TestListActivityRosterUseCase_ActivityRepoErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, &mockReservationRepository{}, &stubUserRepository{})
	_, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get activity 10")
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestListActivityRosterUseCase_ListByActivityErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return nil, errors.New("query timeout")
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, &stubUserRepository{})
	_, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list reservations for activity 10")
	assert.Contains(t, err.Error(), "query timeout")
}

func TestListActivityRosterUseCase_GetByIDsErrorWrapped(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow, "Luna"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	_, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve owners for activity 10")
	assert.Contains(t, err.Error(), "db connection lost")
}

// ── Empty / happy paths ───────────────────────────────────────────

func TestListActivityRosterUseCase_Empty(t *testing.T) {
	t.Parallel()
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return validFutureActivity(10), nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return nil, nil
		},
	}
	// userRepo.GetByIDs must NOT be called when the roster is empty
	// (sentinel for the early-return path).
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			t.Fatal("GetByIDs must not be called when there are no reservations")
			return nil, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	require.NotNil(t, output.Activity)
	assert.Equal(t, 10, output.Activity.ID())
	assert.Empty(t, output.Confirmed)
	assert.Empty(t, output.Pending)
}

func TestListActivityRosterUseCase_PartitionsConfirmedAndPending(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow.Add(-3*time.Hour), "Luna"),
		rosterView(2, 10, 21, 1, domain.StatusPendingToConfirm, fixedNow.Add(-2*time.Hour), "Toby"),
		rosterView(3, 10, 22, 1, domain.StatusConfirmed, fixedNow.Add(-1*time.Hour), "Maya"),
		rosterView(4, 10, 23, 99, domain.StatusPendingToConfirm, fixedNow, "Coco"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.ElementsMatch(t, []int{99, 1}, ids, "owner ids must be dedup'd: 99 appears twice")
			return []*domain.User{
				fixedOwner(99, "Ana"),
				fixedOwner(1, "Juan"),
			}, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	require.NotNil(t, output.Activity)
	assert.Equal(t, 10, output.Activity.ID())

	require.Len(t, output.Confirmed, 2)
	assert.Equal(t, 1, output.Confirmed[0].ReservationID())
	assert.Equal(t, 20, output.Confirmed[0].DogID())
	assert.Equal(t, "Luna", output.Confirmed[0].DogName())
	assert.Equal(t, 99, output.Confirmed[0].OwnerID())
	assert.Equal(t, "Ana", output.Confirmed[0].OwnerName())
	assert.Equal(t, 3, output.Confirmed[1].ReservationID())
	assert.Equal(t, "Maya", output.Confirmed[1].DogName())
	assert.Equal(t, 1, output.Confirmed[1].OwnerID())
	assert.Equal(t, "Juan", output.Confirmed[1].OwnerName())

	require.Len(t, output.Pending, 2)
	assert.Equal(t, 2, output.Pending[0].ReservationID())
	assert.Equal(t, "Toby", output.Pending[0].DogName())
	assert.Equal(t, "Juan", output.Pending[0].OwnerName())
	assert.Equal(t, 4, output.Pending[1].ReservationID())
	assert.Equal(t, "Coco", output.Pending[1].DogName())
	assert.Equal(t, "Ana", output.Pending[1].OwnerName())
}

// ── Filter: non-occupied statuses are dropped ─────────────────────

func TestListActivityRosterUseCase_DropsNonOccupiedStatuses(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow, "Luna"),
		rosterView(2, 10, 21, 1, domain.StatusPendingToConfirm, fixedNow, "Toby"),
		rosterView(3, 10, 22, 1, domain.StatusCancelledInTime, fixedNow, "Maya"),
		rosterView(4, 10, 23, 99, domain.StatusCancelledLate, fixedNow, "Coco"),
		rosterView(5, 10, 24, 1, domain.StatusCompleted, fixedNow, "Rex"),
		rosterView(6, 10, 25, 99, domain.StatusNoShow, fixedNow, "Bolt"),
		rosterView(7, 10, 26, 1, domain.StatusForgiven, fixedNow, "Kira"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.ElementsMatch(t, []int{99, 1}, ids, "owners of dropped statuses must not appear")
			return []*domain.User{
				fixedOwner(99, "Ana"),
				fixedOwner(1, "Juan"),
			}, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	assert.Len(t, output.Confirmed, 1, "only CONFIRMED counts")
	assert.Len(t, output.Pending, 1, "only PENDING_TO_CONFIRM counts")
}

// ── Only confirmed / only pending ─────────────────────────────────

func TestListActivityRosterUseCase_OnlyConfirmed(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow, "Luna"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	assert.Len(t, output.Confirmed, 1)
	assert.Empty(t, output.Pending)
}

func TestListActivityRosterUseCase_OnlyPending(t *testing.T) {
	t.Parallel()
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	assert.Empty(t, output.Confirmed)
	assert.Len(t, output.Pending, 1)
}

// ── Owner resolution resilience ──────────────────────────────────

func TestListActivityRosterUseCase_MissingOwnerLeavesEmptyName(t *testing.T) {
	t.Parallel()
	// If the user repo returns fewer users than owners (e.g. an
	// owner was deleted between the reservation insert and the
	// roster fetch), the entry must still be on the roster with an
	// empty OwnerName. We do NOT fail the whole call.
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow, "Luna"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return nil, nil // owner 99 vanished
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	require.Len(t, output.Confirmed, 1)
	assert.Equal(t, 99, output.Confirmed[0].OwnerID())
	assert.Equal(t, "", output.Confirmed[0].OwnerName(), "missing owner → empty name, no error")
}

func TestListActivityRosterUseCase_DuplicateOwnerIDsAreDeduped(t *testing.T) {
	t.Parallel()
	// Three dogs of the same owner → userRepo.GetByIDs must be
	// called with [99] only, not [99, 99, 99].
	activity := validFutureActivity(10)
	views := []*domain.ReservationView{
		rosterView(1, 10, 20, 99, domain.StatusConfirmed, fixedNow, "Luna"),
		rosterView(2, 10, 21, 99, domain.StatusConfirmed, fixedNow, "Toby"),
		rosterView(3, 10, 22, 99, domain.StatusPendingToConfirm, fixedNow, "Maya"),
	}
	activityRepo := &stubActivityRepository{
		getByID: func(context.Context, int) (*domain.Activity, error) { return activity, nil },
	}
	reservationRepo := &mockReservationRepository{
		listByActivityView: func(context.Context, int, int, int) ([]*domain.ReservationView, error) {
			return views, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.Equal(t, []int{99}, ids, "duplicate owner ids must be dedup'd before the user query")
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListActivityRosterInput(10))
	require.NoError(t, err)
	require.Len(t, output.Confirmed, 2)
	require.Len(t, output.Pending, 1)
	for _, e := range output.Confirmed {
		assert.Equal(t, "Ana", e.OwnerName())
	}
	assert.Equal(t, "Ana", output.Pending[0].OwnerName())
}
