package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

// Transactor is the minimum interface the use case needs to wrap its
// work in a database transaction. Implemented by
// postgres.Transactor. The use case does not depend on database/sql
// directly: this indirection keeps the use case testable with a
// fake Transactor.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// RegisterReservationInput is the validated payload for creating a
// new reservation. UserID is the owner of the dog and the pass and
// is taken from the URL path by the handler (not from the body, to
// match the /users/:user_id/passes pattern).
type RegisterReservationInput struct {
	UserID     int
	ActivityID int
	DogID      int
	PassID     int
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
type RegisterReservationUseCase struct {
	transactor      Transactor
	activityRepo    domain.ActivityRepository
	dogRepo         domain.DogRepository
	passRepo        domain.PassRepository
	reservationRepo domain.ReservationRepository
	now             func() time.Time
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
		now:             time.Now,
	}
}

// Execute runs the create flow atomically. Returns a typed error
// from this package (ErrInvalidActivity, ErrActivityInPast, etc.) for
// the expected failure modes; the handler maps each to a specific
// HTTP status. Any other error is wrapped with %w so the handler
// can log it as a 500.
func (uc *RegisterReservationUseCase) Execute(ctx context.Context, input RegisterReservationInput) (RegisterReservationOutput, error) {
	if err := input.validate(); err != nil {
		return RegisterReservationOutput{}, err
	}

	var output RegisterReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		id, err := uc.runInTx(txCtx, input)
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

// runInTx performs every step of the create flow inside the
// transaction-bound context. Keeping this private and side-effect
// free (apart from the explicit repository calls) makes the
// Transactor.WithinTx closure easy to read.
func (uc *RegisterReservationUseCase) runInTx(ctx context.Context, input RegisterReservationInput) (int, error) {
	now := uc.now()

	// 1. Activity must exist and be in the future.
	activity, err := uc.activityRepo.GetByID(ctx, input.ActivityID)
	if err != nil {
		if errors.Is(err, postgres.ErrActivityNotFound) {
			return 0, ErrInvalidActivity
		}
		return 0, fmt.Errorf("get activity %d: %w", input.ActivityID, err)
	}
	if activity.IsInThePast(now) {
		return 0, ErrActivityInPast
	}

	// 2. Activity must have remaining capacity. We list every
	// reservation for the activity and count the CONFIRMED ones; a
	// slot is "taken" only by a CONFIRMED booking, because
	// cancellations and no-shows already free their slot.
	existing, err := uc.reservationRepo.ListByActivity(ctx, input.ActivityID)
	if err != nil {
		return 0, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID, err)
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

	// 3. Dog must exist and be owned by UserID. The same error is
	// returned for "not found" and "owned by another user" so we
	// do not leak the existence of other users' dogs.
	dog, err := uc.dogRepo.GetByID(ctx, input.DogID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return 0, ErrInvalidDog
		}
		return 0, fmt.Errorf("get dog %d: %w", input.DogID, err)
	}
	if dog.UserID() != input.UserID {
		return 0, ErrInvalidDog
	}

	// 4. Pass must exist, be owned by UserID, not be exhausted, and
	// not be expired. Same rationale as ErrInvalidDog for the
	// ownership check.
	pass, err := uc.passRepo.GetByID(ctx, input.PassID)
	if err != nil {
		if errors.Is(err, postgres.ErrPassNotFound) {
			return 0, ErrInvalidPass
		}
		return 0, fmt.Errorf("get pass %d: %w", input.PassID, err)
	}
	if pass.UserID() != input.UserID {
		return 0, ErrInvalidPass
	}
	if pass.IsExhausted() {
		return 0, ErrPassExhausted
	}
	if pass.IsExpired(now) {
		return 0, ErrPassExpired
	}

	// 5. Consume one pass session. The reason includes the
	// activity and dog ids so the audit log is self-explanatory.
	reason := fmt.Sprintf("Reservation: activity %d, dog %d", input.ActivityID, input.DogID)
	movement, err := pass.ConsumeSession(reason, now)
	if err != nil {
		return 0, fmt.Errorf("consume pass %d: %w", input.PassID, err)
	}

	// 6. Persist the pass update + the movement. Both happen in
	// the same transaction as the reservation Create, so a failure
	// here rolls everything back.
	if err := uc.passRepo.Update(ctx, pass); err != nil {
		return 0, fmt.Errorf("update pass %d: %w", input.PassID, err)
	}
	if err := uc.passRepo.AddMovement(ctx, &movement); err != nil {
		return 0, fmt.Errorf("add movement for pass %d: %w", input.PassID, err)
	}

	// 7. Create the reservation. NewReservation forces
	// StatusConfirmed; the use case does not expose a way to start
	// in any other state.
	reservation, err := domain.NewReservation(0, input.ActivityID, input.DogID, input.PassID, now)
	if err != nil {
		return 0, fmt.Errorf("build reservation: %w", err)
	}
	id, err := uc.reservationRepo.Create(ctx, reservation)
	if err != nil {
		if errors.Is(err, postgres.ErrDuplicateReservation) {
			return 0, ErrDuplicateReservationForDog
		}
		return 0, fmt.Errorf("create reservation: %w", err)
	}
	return id, nil
}

func (input RegisterReservationInput) validate() error {
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	if input.ActivityID <= 0 {
		return &ValidationError{Field: "activity_id"}
	}
	if input.DogID <= 0 {
		return &ValidationError{Field: "dog_id"}
	}
	if input.PassID <= 0 {
		return &ValidationError{Field: "pass_id"}
	}
	return nil
}

// ErrDuplicateReservationForDog is returned when the
// UNIQUE (activity_id, dog_id) constraint fires at insert time. This
// is the only way to detect a duplicate booking when the existing
// row is not in StatusConfirmed (e.g., the user previously cancelled
// in time and is trying to rebook — which we want to allow; the
// constraint is enforced to keep the history clean, so we surface
// this as 409 and let the user cancel+rebook explicitly). A future
// use case may relax this when the workflow is clear.
var ErrDuplicateReservationForDog = errors.New("dog already booked for this activity")
