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

// pendingView builds a reservation view anchored to a future
// activity. Reuses mustNewReservationView from the roster suite
// because the shape is identical — only the use case filter (the
// status) differs and is asserted by each test.
func pendingView(
	id int, activityID int, dogID int, ownerID int,
	status domain.ReservationStatus,
	createdAt time.Time, dogName string,
) *domain.ReservationView {
	return mustNewReservationView(
		id, activityID, dogID, ownerID,
		1, ownerID,
		status, createdAt,
		"Paseo", "Central", fixedNow.Add(7*24*time.Hour),
		dogName, 5,
	)
}

// ── Input validation ──────────────────────────────────────────────

func TestNewListPendingReservationsInput_NormalizesPagination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		limit, offset  int
		wantLimit      int
		wantOffset     int
	}{
		{"defaults", 0, 0, 50, 0},
		{"negative_offset_reset", 100, -7, 100, 0},
		{"limit_capped_at_100", 200, 0, 100, 0},
		{"positive_limit_kept", 25, 0, 25, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewListPendingReservationsInput(tt.limit, tt.offset)
			require.NoError(t, err, "factory returns no error: pagination only")
			assert.Equal(t, tt.wantLimit, in.Limit())
			assert.Equal(t, tt.wantOffset, in.Offset())
		})
	}
}

// ── Repository error wrapping ────────────────────────────────────

func TestListPendingReservationsUseCase_RepoErrorWrapped(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return nil, errors.New("query timeout")
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, &stubUserRepository{})
	_, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list pending reservations")
	assert.Contains(t, err.Error(), "query timeout")
}

func TestListPendingReservationsUseCase_GetByIDsErrorWrapped(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	_, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve owners for pending reservations")
	assert.Contains(t, err.Error(), "db connection lost")
}

// ── Empty / happy paths ──────────────────────────────────────────

func TestListPendingReservationsUseCase_Empty(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return nil, nil
		},
	}
	// userRepo.GetByIDs must NOT be called when the list is empty.
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			t.Fatal("GetByIDs must not be called when there are no pending reservations")
			return nil, nil
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	assert.Empty(t, output.Pending)
}

// ── Status filtering: only PENDING_TO_CONFIRM ─────────────────────

func TestListPendingReservationsUseCase_OnlyPendingReturned(t *testing.T) {
	t.Parallel()
	// Repository-level guarantee (SQL WHERE): the repo only
	// returns PENDING_TO_CONFIRM rows. The use case must not
	// re-filter. We assert this with an explicit mock that
	// returns only pending entries.
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
				pendingView(2, 10, 21, 99, domain.StatusPendingToConfirm, fixedNow.Add(time.Minute), "Toby"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.ElementsMatch(t, []int{99}, ids, "single owner, dedup'd")
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	require.Len(t, output.Pending, 2)
	assert.Equal(t, 1, output.Pending[0].ReservationID())
	assert.Equal(t, "Luna", output.Pending[0].DogName())
	assert.Equal(t, "Ana", output.Pending[0].OwnerName())
	assert.Equal(t, 2, output.Pending[1].ReservationID())
}

// ── Owner name resolution ────────────────────────────────────────

func TestListPendingReservationsUseCase_ResolvesMultipleOwners(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 1, domain.StatusPendingToConfirm, fixedNow, "Luna"),
				pendingView(2, 11, 21, 99, domain.StatusPendingToConfirm, fixedNow.Add(time.Minute), "Toby"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.ElementsMatch(t, []int{1, 99}, ids, "two distinct owners → two ids")
			return []*domain.User{
				fixedOwner(1, "Juan"),
				fixedOwner(99, "Ana"),
			}, nil
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	require.Len(t, output.Pending, 2)

	// Build a map so we don't depend on the user repo's order.
	byID := map[string]string{}
	for _, e := range output.Pending {
		byID[e.DogName()] = e.OwnerName()
	}
	assert.Equal(t, "Juan", byID["Luna"])
	assert.Equal(t, "Ana", byID["Toby"])
}

// ── Duplicate owner dedup ────────────────────────────────────────

func TestListPendingReservationsUseCase_DuplicateOwnerIDsDeduped(t *testing.T) {
	t.Parallel()
	// Three pending bookings of the same owner → userRepo.GetByIDs
	// must be called with [99] only, not [99, 99, 99].
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
				pendingView(2, 10, 21, 99, domain.StatusPendingToConfirm, fixedNow.Add(time.Minute), "Toby"),
				pendingView(3, 10, 22, 99, domain.StatusPendingToConfirm, fixedNow.Add(2*time.Minute), "Maya"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(_ context.Context, ids []int) ([]*domain.User, error) {
			assert.Equal(t, []int{99}, ids, "duplicate owner ids must be dedup'd before the user query")
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	require.Len(t, output.Pending, 3)
	for _, e := range output.Pending {
		assert.Equal(t, "Ana", e.OwnerName())
	}
}

// ── Missing owner resilience ─────────────────────────────────────

func TestListPendingReservationsUseCase_MissingOwnerLeavesEmptyName(t *testing.T) {
	t.Parallel()
	// If the user repo returns an empty slice (owner deleted),
	// every entry must still be on the output with an empty
	// OwnerName. The call must NOT fail.
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return nil, nil // owner 99 vanished
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	require.Len(t, output.Pending, 1)
	assert.Equal(t, 99, output.Pending[0].OwnerID())
	assert.Equal(t, "", output.Pending[0].OwnerName(), "missing owner → empty name, no error")
}

// ── Pending reasons audit trail ──────────────────────────────────

// TestListPendingReservationsUseCase_LoadsReasonsBatched verifies
// that the use case calls ListPendingReasonsByReservations once per
// page with the full set of reservation IDs, and that the per-entry
// Reasons() accessor returns the matching slice in ordinal order.
//
// Missing reservations in the map (i.e. no reasons recorded) leave
// the entry with a nil Reasons slice — the wire layer must omit the
// field in that case.
func TestListPendingReservationsUseCase_LoadsReasonsBatched(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
				pendingView(2, 10, 21, 99, domain.StatusPendingToConfirm, fixedNow, "Toby"),
				pendingView(3, 10, 22, 99, domain.StatusPendingToConfirm, fixedNow, "Maya"),
			}, nil
		},
		listPendingReasonsByReservations: func(_ context.Context, ids []int) (map[int][]domain.PendingReason, error) {
			assert.ElementsMatch(t, []int{1, 2, 3}, ids, "single batched query with every page id")
			return map[int][]domain.PendingReason{
				1: {{Code: SexNeuteredReasonPrefix + domain.ReasonIntactVsCastrated, DogIDs: []int{20, 30}}},
				3: {{Code: domain.ReasonHasSpecialCondition, DogIDs: []int{22}}},
				// 2 has no recorded reasons → absent from map.
			}, nil
		},
	}
	userRepo := &stubUserRepository{
		getByIDs: func(context.Context, []int) ([]*domain.User, error) {
			return []*domain.User{fixedOwner(99, "Ana")}, nil
		},
	}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	output, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.NoError(t, err)
	require.Len(t, output.Pending, 3)

	// Reservation 1: one sex/neutered reason.
	require.Len(t, output.Pending[0].Reasons(), 1)
	assert.Equal(t, SexNeuteredReasonPrefix+domain.ReasonIntactVsCastrated, output.Pending[0].Reasons()[0].Code)
	assert.Equal(t, []int{20, 30}, output.Pending[0].Reasons()[0].DogIDs)

	// Reservation 2: no reasons recorded → nil.
	assert.Nil(t, output.Pending[1].Reasons())

	// Reservation 3: one special-condition reason.
	require.Len(t, output.Pending[2].Reasons(), 1)
	assert.Equal(t, domain.ReasonHasSpecialCondition, output.Pending[2].Reasons()[0].Code)
	assert.Equal(t, []int{22}, output.Pending[2].Reasons()[0].DogIDs)
}

// TestListPendingReservationsUseCase_ReasonsBatchErrorWrapped makes
// sure a failure in the reasons query is wrapped and propagated
// rather than silently swallowed.
func TestListPendingReservationsUseCase_ReasonsBatchErrorWrapped(t *testing.T) {
	t.Parallel()
	reservationRepo := &mockReservationRepository{
		listPendingView: func(context.Context, int, int) ([]*domain.ReservationView, error) {
			return []*domain.ReservationView{
				pendingView(1, 10, 20, 99, domain.StatusPendingToConfirm, fixedNow, "Luna"),
			}, nil
		},
		listPendingReasonsByReservations: func(context.Context, []int) (map[int][]domain.PendingReason, error) {
			return nil, errors.New("audit table unreachable")
		},
	}
	userRepo := &stubUserRepository{}
	uc := NewListPendingReservationsUseCase(reservationRepo, userRepo)
	_, err := uc.Execute(context.Background(), MustNewListPendingReservationsInput(100, 0))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve pending reasons")
	assert.Contains(t, err.Error(), "audit table unreachable")
}
