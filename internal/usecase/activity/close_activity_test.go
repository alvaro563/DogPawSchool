package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	reservationuc "dogpaw/internal/usecase/reservation"
)

// closeFinishedActivity returns an activity that started at
// fixedNow - 25h with duration 1h, so it ended at fixedNow - 24h.
func closeFinishedActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-25*time.Hour))
}

// closeFinishedClosedActivity is closeFinishedActivity in the state a
// repository would hand back for an already-closed row.
func closeFinishedClosedActivity(id int) *domain.Activity {
	activity, err := domain.ReconstituteActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 1, fixedNow.Add(-25*time.Hour), true)
	if err != nil {
		panic(err)
	}
	return activity
}

// closeOngoingActivity returns an activity that started at
// fixedNow - 30min with duration 2h, so it ends at fixedNow + 1.5h.
func closeOngoingActivity(id int) *domain.Activity {
	return domain.MustNewActivity(id, "Paseo", "Central", domain.TypeRoute, 5, 2, fixedNow.Add(-30*time.Minute))
}

// stubTransactorActivity is a no-op transactor for activity tests.
type stubTransactorActivity struct{}

func (s *stubTransactorActivity) WithinTx(_ context.Context, fn func(ctx context.Context) error) error {
	return fn(context.Background())
}

// stubNoShower implements reservationNoShower for tests.
type stubNoShower struct {
	fn func(ctx context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error)
}

func (s *stubNoShower) Execute(ctx context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
	if s.fn != nil {
		return s.fn(ctx, in)
	}
	return reservationuc.MarkReservationNoShowOutput{Reservation: mustNewClosedReservation(in.ReservationID())}, nil
}

// stubCompleter implements reservationCompleter for tests.
type stubCompleter struct {
	fn func(ctx context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error)
}

func (s *stubCompleter) Execute(ctx context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
	if s.fn != nil {
		return s.fn(ctx, in)
	}
	return reservationuc.CompleteReservationOutput{Reservation: mustNewCompletedReservation(in.ReservationID())}, nil
}

func mustNewClosedReservation(id int) *domain.Reservation {
	r, _ := domain.NewReservationWithStatus(id, 10, 20, 30, domain.StatusNoShow, fixedNow)
	return r
}

func mustNewCompletedReservation(id int) *domain.Reservation {
	r, _ := domain.NewReservationWithStatus(id, 10, 20, 30, domain.StatusCompleted, fixedNow)
	return r
}

// confirmedRes builds a confirmed reservation with the given IDs.
func confirmedRes(id, activityID, dogID, passID int) *domain.Reservation {
	r, _ := domain.NewReservationWithStatus(id, activityID, dogID, passID, domain.StatusConfirmed, fixedNow)
	return r
}

// closeValidDog returns a dog owned by userID.
func closeValidDog(id, userID int) *domain.Dog {
	d, _ := domain.NewDog(id, "Luna", "Labrador", "ES-123", 24, domain.SexFemale, 25, userID)
	return d
}

// stubDogRepoForClose is a simple dog repo for close tests.
type stubDogRepoForClose struct {
	getByID func(ctx context.Context, id int) (*domain.Dog, error)
}

func (s *stubDogRepoForClose) Create(_ context.Context, _ *domain.Dog) (int, error) { return 0, nil }
func (s *stubDogRepoForClose) GetByID(ctx context.Context, id int) (*domain.Dog, error) {
	if s.getByID != nil {
		return s.getByID(ctx, id)
	}
	return nil, nil
}
func (s *stubDogRepoForClose) Update(_ context.Context, _ *domain.Dog) error { return nil }
func (s *stubDogRepoForClose) Delete(_ context.Context, _ int) error         { return nil }
func (s *stubDogRepoForClose) ListByOwner(_ context.Context, _, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListAll(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByIncompatibility(_ context.Context, _, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByBreed(_ context.Context, _ string, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListBySex(_ context.Context, _ domain.Sex, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByNeutered(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByHeat(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByIsActive(_ context.Context, _ bool, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListByAgeBracket(_ context.Context, _ domain.AgeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}
func (s *stubDogRepoForClose) ListBySizeBracket(_ context.Context, _ domain.SizeBracket, _, _ int) ([]*domain.Dog, error) {
	return nil, nil
}

// stubResRepoForClose is a simple reservation repo for close tests.
type stubResRepoForClose struct {
	listByActivity func(ctx context.Context, activityID int) ([]*domain.Reservation, error)
}

func (s *stubResRepoForClose) Create(_ context.Context, _ *domain.Reservation) (int, error) {
	return 0, nil
}
func (s *stubResRepoForClose) Update(_ context.Context, _ *domain.Reservation) error { return nil }
func (s *stubResRepoForClose) GetByID(_ context.Context, _ int) (*domain.Reservation, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByActivity(ctx context.Context, activityID int) ([]*domain.Reservation, error) {
	if s.listByActivity != nil {
		return s.listByActivity(ctx, activityID)
	}
	return nil, nil
}
func (s *stubResRepoForClose) ListByDog(_ context.Context, _ int) ([]*domain.Reservation, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByPass(_ context.Context, _ int) ([]*domain.Reservation, error) {
	return nil, nil
}
func (s *stubResRepoForClose) GetView(_ context.Context, _ int) (*domain.ReservationView, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByUserView(_ context.Context, _ int, _ *domain.ReservationStatus, _, _ *time.Time, _, _ int) ([]*domain.ReservationView, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByUserUpcomingView(_ context.Context, _, _, _ int) ([]*domain.ReservationView, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByDogView(_ context.Context, _, _, _ int) ([]*domain.ReservationView, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByPassView(_ context.Context, _, _, _ int) ([]*domain.ReservationView, error) {
	return nil, nil
}
func (s *stubResRepoForClose) ListByActivityView(_ context.Context, _, _, _ int) ([]*domain.ReservationView, error) {
	return nil, nil
}

// fixedNow is a deterministic clock used by all close tests.
var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func validCloseInput() CloseActivityInput {
	return MustNewCloseActivityInput(10, nil, func() time.Time { return fixedNow })
}

func TestNewCloseActivityInput(t *testing.T) {
	scenarios := []struct {
		name      string
		activity  int
		noShowIDs []int
		field     string
	}{
		{"zero_activity_id", 0, nil, "activity_id"},
		{"negative_activity_id", -1, nil, "activity_id"},
		{"zero_no_show_id", 10, []int{0}, "no_show_reservation_ids[0]"},
		{"negative_no_show_id", 10, []int{-5}, "no_show_reservation_ids[0]"},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := NewCloseActivityInput(s.activity, s.noShowIDs, func() time.Time { return fixedNow })
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, s.field, verr.Field)
		})
	}
}

func TestNewCloseActivityInput_Deduplicates(t *testing.T) {
	in, err := NewCloseActivityInput(10, []int{1, 2, 1, 3, 2}, func() time.Time { return fixedNow })
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, in.NoShowReservationIDs())
}

func TestCloseActivityUseCase_Success_AllComplete(t *testing.T) {
	activity := closeFinishedActivity(10)
	res1 := confirmedRes(1, 10, 20, 30)
	res2 := confirmedRes(2, 10, 21, 31)
	dog1 := closeValidDog(20, 1)
	dog2 := closeValidDog(21, 1)

	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, id int) (*domain.Activity, error) {
			assert.Equal(t, 10, id)
			return activity, nil
		},
		update: func(_ context.Context, a *domain.Activity) error {
			assert.True(t, a.IsClosed())
			return nil
		},
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{res1, res2}, nil
		},
	}
	dogRepo := &stubDogRepoForClose{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			if id == 20 {
				return dog1, nil
			}
			return dog2, nil
		},
	}
	completer := &stubCompleter{}
	noShower := &stubNoShower{}

	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, dogRepo, resRepo,
		noShower, completer,
	)
	output, err := uc.Execute(context.Background(), validCloseInput())
	require.NoError(t, err)
	assert.True(t, output.Activity.IsClosed())
}

func TestCloseActivityUseCase_Success_AllNoShow(t *testing.T) {
	activity := closeFinishedActivity(10)
	res1 := confirmedRes(1, 10, 20, 30)
	dog1 := closeValidDog(20, 1)

	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
		update:  func(_ context.Context, _ *domain.Activity) error { return nil },
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{res1}, nil
		},
	}
	dogRepo := &stubDogRepoForClose{
		getByID: func(_ context.Context, _ int) (*domain.Dog, error) { return dog1, nil },
	}
	noShower := &stubNoShower{}
	completer := &stubCompleter{}

	input := MustNewCloseActivityInput(10, []int{1}, func() time.Time { return fixedNow })
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, dogRepo, resRepo,
		noShower, completer,
	)
	output, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, output.Activity.IsClosed())
}

func TestCloseActivityUseCase_Success_Mixed(t *testing.T) {
	activity := closeFinishedActivity(10)
	res1 := confirmedRes(1, 10, 20, 30)
	res2 := confirmedRes(2, 10, 21, 31)
	dog1 := closeValidDog(20, 1)
	dog2 := closeValidDog(21, 1)

	var noShowCalled, completeCalled bool
	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
		update:  func(_ context.Context, _ *domain.Activity) error { return nil },
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{res1, res2}, nil
		},
	}
	dogRepo := &stubDogRepoForClose{
		getByID: func(_ context.Context, id int) (*domain.Dog, error) {
			if id == 20 {
				return dog1, nil
			}
			return dog2, nil
		},
	}
	noShower := &stubNoShower{
		fn: func(_ context.Context, in reservationuc.MarkReservationNoShowInput) (reservationuc.MarkReservationNoShowOutput, error) {
			noShowCalled = true
			assert.Equal(t, 1, in.ReservationID())
			return reservationuc.MarkReservationNoShowOutput{Reservation: mustNewClosedReservation(in.ReservationID())}, nil
		},
	}
	completer := &stubCompleter{
		fn: func(_ context.Context, in reservationuc.CompleteReservationInput) (reservationuc.CompleteReservationOutput, error) {
			completeCalled = true
			assert.Equal(t, 2, in.ReservationID())
			return reservationuc.CompleteReservationOutput{Reservation: mustNewCompletedReservation(in.ReservationID())}, nil
		},
	}

	input := MustNewCloseActivityInput(10, []int{1}, func() time.Time { return fixedNow })
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, dogRepo, resRepo,
		noShower, completer,
	)
	_, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, noShowCalled, "noShower should have been called for reservation 1")
	assert.True(t, completeCalled, "completer should have been called for reservation 2")
}

func TestCloseActivityUseCase_ActivityNotFound(t *testing.T) {
	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return nil, nil },
	}
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, nil,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), validCloseInput())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCloseActivityUseCase_ActivityNotFinished(t *testing.T) {
	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) {
			return closeOngoingActivity(10), nil
		},
	}
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, nil,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), validCloseInput())
	assert.ErrorIs(t, err, ErrNotFinished)
}

func TestCloseActivityUseCase_AlreadyClosed(t *testing.T) {
	activity := closeFinishedClosedActivity(10)

	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
	}
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, nil,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), validCloseInput())
	assert.ErrorIs(t, err, ErrAlreadyClosed)
}

func TestCloseActivityUseCase_NoShowID_NotFound(t *testing.T) {
	activity := closeFinishedActivity(10)
	res1 := confirmedRes(1, 10, 20, 30)

	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return []*domain.Reservation{res1}, nil
		},
	}
	input := MustNewCloseActivityInput(10, []int{999}, func() time.Time { return fixedNow })
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, resRepo,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), input)
	assert.ErrorIs(t, err, ErrReservationNotFound)
}

func TestCloseActivityUseCase_NoShowID_WrongActivity(t *testing.T) {
	activity := closeFinishedActivity(10)
	// No reservations at all for this activity.
	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return nil, nil
		},
	}
	// noShowID 99 does not exist in any CONFIRMED reservation.
	input := MustNewCloseActivityInput(10, []int{99}, func() time.Time { return fixedNow })
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, resRepo,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), input)
	assert.ErrorIs(t, err, ErrReservationNotFound)
}

func TestCloseActivityUseCase_RepoError_Wrapped(t *testing.T) {
	repoErr := errors.New("db connection lost")
	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return nil, repoErr },
	}
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, nil,
		&stubNoShower{}, &stubCompleter{},
	)
	_, err := uc.Execute(context.Background(), validCloseInput())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
	assert.Contains(t, err.Error(), "get activity")
}

func TestCloseActivityUseCase_EmptyConfirmedList(t *testing.T) {
	activity := closeFinishedActivity(10)

	activityRepo := &mockActivityRepository{
		getByID: func(_ context.Context, _ int) (*domain.Activity, error) { return activity, nil },
		update:  func(_ context.Context, _ *domain.Activity) error { return nil },
	}
	resRepo := &stubResRepoForClose{
		listByActivity: func(_ context.Context, _ int) ([]*domain.Reservation, error) {
			return nil, nil
		},
	}
	uc := NewCloseActivityUseCase(
		&stubTransactorActivity{}, activityRepo, nil, resRepo,
		&stubNoShower{}, &stubCompleter{},
	)
	output, err := uc.Execute(context.Background(), validCloseInput())
	require.NoError(t, err)
	assert.True(t, output.Activity.IsClosed())
}
