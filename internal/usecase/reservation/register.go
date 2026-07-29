package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// RegisterReservationInput is the validated command to book a dog
// into an activity, paid from a pass. All fields are private: the
// only way to obtain one is NewRegisterReservationInput.
type RegisterReservationInput struct {
	userID     int
	activityID int
	dogID      int
	passID     int
	now        time.Time
}

func (in RegisterReservationInput) UserID() int     { return in.userID }
func (in RegisterReservationInput) ActivityID() int { return in.activityID }
func (in RegisterReservationInput) DogID() int      { return in.dogID }
func (in RegisterReservationInput) PassID() int     { return in.passID }
func (in RegisterReservationInput) Now() time.Time  { return in.now }

// NewRegisterReservationInput validates the four ids and accepts a
// now-Provider so the use case can be tested with a fixed clock. A
// nil provider is replaced with time.Now.
func NewRegisterReservationInput(userID, activityID, dogID, passID int, now func() time.Time) (RegisterReservationInput, error) {
	if userID <= 0 {
		return RegisterReservationInput{}, &ValidationError{Field: "user_id"}
	}
	if activityID <= 0 {
		return RegisterReservationInput{}, &ValidationError{Field: "activity_id"}
	}
	if dogID <= 0 {
		return RegisterReservationInput{}, &ValidationError{Field: "dog_id"}
	}
	if passID <= 0 {
		return RegisterReservationInput{}, &ValidationError{Field: "pass_id"}
	}
	if now == nil {
		now = time.Now
	}
	return RegisterReservationInput{userID: userID, activityID: activityID, dogID: dogID, passID: passID, now: now()}, nil
}

// MustNewRegisterReservationInput panics on validation error. For
// tests with a real clock.
func MustNewRegisterReservationInput(userID, activityID, dogID, passID int, now func() time.Time) RegisterReservationInput {
	in, err := NewRegisterReservationInput(userID, activityID, dogID, passID, now)
	if err != nil {
		panic(err)
	}
	return in
}

// RegisterReservationOutput is the result of a successful create.
type RegisterReservationOutput struct {
	ID int
}

// RegisterReservationUseCase books a dog into an activity paid from
// a pass. The whole flow is wrapped in a single database
// transaction so that, on failure, neither the pass is decremented
// nor a reservation row is left dangling:
//
//  1. The activity must exist and be in the future.
//  2. The activity must have remaining capacity (CONFIRMED
//     bookings < max_capacity).
//  3. The dog must exist and be owned by UserID.
//  4. The pass must exist, be owned by UserID, not be exhausted,
//     and not be expired.
//  5. One pass session is consumed (in memory) and a movement is
//     appended to the audit log.
//  6. The reservation is created in StatusConfirmed.
//
// The use case holds no mutable state: the clock travels with the
// input (RegisterReservationInput freezes it at construction), so a
// single instance is safe to share across concurrent requests, which
// is exactly how the router wires it.
type RegisterReservationUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
}

func NewRegisterReservationUseCase(
	transactor Transactor,
	activityRepo domain.ActivityRepository,
	dogRepo domain.DogRepository,
	passRepo domain.PassRepository,
	reservationRepo domain.ReservationRepository,
) *RegisterReservationUseCase {
	return &RegisterReservationUseCase{
		transactor:      transactor,
		activityRepo:    activityRepo,
		dogRepo:         dogRepo,
		passRepo:        passRepo,
		reservationRepo: reservationRepo,
	}
}

// Execute runs the create flow atomically.
func (uc *RegisterReservationUseCase) Execute(ctx context.Context, input RegisterReservationInput) (RegisterReservationOutput, error) {
	var output RegisterReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		id, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = RegisterReservationOutput{ID: id}
		return nil
	})
	if err != nil {
		return RegisterReservationOutput{}, err
	}
	return output, nil
}

func (uc *RegisterReservationUseCase) runInTx(ctx context.Context, input RegisterReservationInput, now time.Time) (int, error) {
	// 1. Activity must exist and be in the future.
	activity, err := uc.activityRepo.GetByID(ctx, input.ActivityID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, ErrInvalidActivity
		}
		return 0, fmt.Errorf("get activity %d: %w", input.ActivityID(), err)
	}
	if activity == nil {
		return 0, ErrInvalidActivity
	}
	if activity.IsInThePast(now) {
		return 0, ErrActivityInPast
	}

	// 2. Activity must have remaining capacity.
	existing, err := uc.reservationRepo.ListByActivity(ctx, input.ActivityID())
	if err != nil {
		return 0, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}
	confirmed := 0
	for _, reservation := range existing {
		if reservation.IsConfirmed() {
			confirmed++
		}
	}
	if activity.IsFull(confirmed) {
		return 0, ErrActivityFull
	}

	// 3. Dog must exist and be owned by UserID.
	dog, err := uc.dogRepo.GetByID(ctx, input.DogID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, ErrInvalidDog
		}
		return 0, fmt.Errorf("get dog %d: %w", input.DogID(), err)
	}
	if dog == nil {
		return 0, ErrInvalidDog
	}
	if dog.UserID() != input.UserID() {
		return 0, ErrInvalidDog
	}

	// 4. Pass must exist, be owned by UserID, not be exhausted, and
	// not be expired.
	pass, err := uc.passRepo.GetByID(ctx, input.PassID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, ErrInvalidPass
		}
		return 0, fmt.Errorf("get pass %d: %w", input.PassID(), err)
	}
	if pass == nil {
		return 0, ErrInvalidPass
	}
	if pass.UserID() != input.UserID() {
		return 0, ErrInvalidPass
	}
	if pass.IsExhausted() {
		return 0, ErrPassExhausted
	}
	if pass.IsExpired(now) {
		return 0, ErrPassExpired
	}

	// 5. Consume one pass session. The audit movement is recorded on
	// the aggregate; Update flushes it together with the new counter.
	reason := fmt.Sprintf("Reservation: activity %d, dog %d", input.ActivityID(), input.DogID())
	if _, err := pass.ConsumeSession(reason, now); err != nil {
		return 0, fmt.Errorf("consume pass %d: %w", input.PassID(), err)
	}

	// 6. Persist the pass: counter + audit row, atomically.
	if err := uc.passRepo.Update(ctx, pass); err != nil {
		return 0, fmt.Errorf("update pass %d: %w", input.PassID(), err)
	}

	// 7. Create the reservation.
	reservation, err := domain.NewReservation(0, input.ActivityID(), input.DogID(), input.PassID(), now)
	if err != nil {
		return 0, fmt.Errorf("build reservation: %w", err)
	}
	id, err := uc.reservationRepo.Create(ctx, reservation)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateReservation) {
			return 0, ErrDuplicateReservationForDog
		}
		return 0, fmt.Errorf("create reservation: %w", err)
	}
	return id, nil
}
