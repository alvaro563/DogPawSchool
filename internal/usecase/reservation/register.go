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

// RegisterReservationOutput is the result of a successful create. The
// Status tells the client whether the booking was confirmed outright or
// held pending an admin decision (PENDING_TO_CONFIRM).
type RegisterReservationOutput struct {
	ID     int
	Status domain.ReservationStatus
}

// RegisterReservationUseCase books a dog into an activity paid from
// a pass. The whole flow is wrapped in a single database
// transaction so that, on failure, neither the pass is decremented
// nor a reservation row is left dangling:
//
//  1. The activity must exist and be in the future.
//  2. The activity must have remaining capacity (bookings holding a
//     slot < max_capacity).
//  3. The dog must exist and be owned by UserID.
//  4. The pass must exist, be owned by UserID, not be exhausted,
//     and not be expired.
//  5. The candidate dog is checked against the dogs already holding
//     a slot (confirmed or pending). An ABSOLUTA conflict blocks the
//     booking; MEDIA/BAJA conflicts create the reservation in
//     StatusPendingToConfirm, keeping the slot until an admin
//     confirms or rejects it.
//  6. One pass session is consumed (in memory) and a movement is
//     appended to the audit log.
//  7. The reservation is created.
//
// The use case holds no mutable state: the clock travels with the
// input (RegisterReservationInput freezes it at construction) and the
// compatibility evaluation is a pure domain operation (Dog.ConflictsWith),
// so a single instance is safe to share across concurrent requests, which
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
		id, status, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = RegisterReservationOutput{ID: id, Status: status}
		return nil
	})
	if err != nil {
		return RegisterReservationOutput{}, err
	}
	return output, nil
}

func (uc *RegisterReservationUseCase) runInTx(ctx context.Context, input RegisterReservationInput, now time.Time) (int, domain.ReservationStatus, error) {
	// 1. Activity must exist and be in the future. FOR UPDATE locks the
	// activity row so two concurrent registrations serialize on the
	// capacity check below (materialised conflict for B2).
	activity, err := uc.activityRepo.GetByIDForUpdate(ctx, input.ActivityID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, ErrInvalidActivity
		}
		return 0, domain.StatusConfirmed, fmt.Errorf("get activity %d: %w", input.ActivityID(), err)
	}
	if activity == nil {
		return 0, domain.StatusConfirmed, ErrInvalidActivity
	}
	if activity.IsInThePast(now) {
		return 0, domain.StatusConfirmed, ErrActivityInPast
	}

	// 2. Activity must have remaining capacity. Pending bookings hold
	// their slot until the admin decides, so capacity is measured over
	// HoldsSlot (confirmed or pending).
	existing, err := uc.reservationRepo.ListByActivity(ctx, input.ActivityID())
	if err != nil {
		return 0, domain.StatusConfirmed, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}
	slotHolders := make([]int, 0, len(existing))
	for _, reservation := range existing {
		if reservation.HoldsSlot() {
			slotHolders = append(slotHolders, reservation.DogID())
		}
	}
	if activity.IsFull(len(slotHolders)) {
		return 0, domain.StatusConfirmed, ErrActivityFull
	}

	// 3. Dog must exist and be owned by UserID.
	dog, err := uc.dogRepo.GetByID(ctx, input.DogID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, ErrInvalidDog
		}
		return 0, domain.StatusConfirmed, fmt.Errorf("get dog %d: %w", input.DogID(), err)
	}
	if dog == nil {
		return 0, domain.StatusConfirmed, ErrInvalidDog
	}
	if dog.UserID() != input.UserID() {
		return 0, domain.StatusConfirmed, ErrInvalidDog
	}

	// 3b. Compatibility check. Load the dogs already holding a slot in
	// this activity and evaluate the trigger->trait collisions in both
	// directions against the candidate. The severity of the loudest
	// conflict decides:
	//   - ABSOLUTA: the booking is blocked.
	//   - MEDIA/BAJA: the booking is kept pending, slot held, until an
	//     admin confirms or rejects it.
	others, err := uc.dogRepo.GetByIDs(ctx, slotHolders)
	if err != nil {
		return 0, domain.StatusConfirmed, fmt.Errorf("get dogs holding a slot in activity %d: %w", input.ActivityID(), err)
	}
	conflicts := make([]domain.CompatibilityConflict, 0, len(others))
	for _, other := range others {
		conflicts = append(conflicts, dog.ConflictsWith(other)...)
	}
	status := domain.StatusConfirmed
	if len(conflicts) > 0 {
		if hasAbsoluteConflict(conflicts) {
			return 0, domain.StatusConfirmed, &IncompatibleDogsError{Conflicts: conflicts}
		}
		status = domain.StatusPendingToConfirm
	}

	// 4. Pass must exist, be owned by UserID, not be exhausted, and
	// not be expired. FOR UPDATE locks the pass row so two concurrent
	// consumptions serialise (prevents B3 lost update).
	pass, err := uc.passRepo.GetByIDForUpdate(ctx, input.PassID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, ErrInvalidPass
		}
		return 0, domain.StatusConfirmed, fmt.Errorf("get pass %d: %w", input.PassID(), err)
	}
	if pass == nil {
		return 0, domain.StatusConfirmed, ErrInvalidPass
	}
	if pass.UserID() != input.UserID() {
		return 0, domain.StatusConfirmed, ErrInvalidPass
	}
	if pass.IsExhausted() {
		return 0, domain.StatusConfirmed, ErrPassExhausted
	}
	if pass.IsExpired(now) {
		return 0, domain.StatusConfirmed, ErrPassExpired
	}

	// 5. Consume one pass session. The audit movement is recorded on
	// the aggregate; Update flushes it together with the new counter.
	reason := fmt.Sprintf("Reservation: activity %d, dog %d", input.ActivityID(), input.DogID())
	if _, err := pass.ConsumeSession(reason, now); err != nil {
		return 0, domain.StatusConfirmed, fmt.Errorf("consume pass %d: %w", input.PassID(), err)
	}

	// 6. Persist the pass: counter + audit row, atomically.
	if err := uc.passRepo.Update(ctx, pass); err != nil {
		return 0, domain.StatusConfirmed, fmt.Errorf("update pass %d: %w", input.PassID(), err)
	}

	// 7. Create the reservation in the resolved status.
	reservation, err := domain.NewReservationWithStatus(0, input.ActivityID(), input.DogID(), input.PassID(), status, now)
	if err != nil {
		return 0, domain.StatusConfirmed, fmt.Errorf("build reservation: %w", err)
	}
	id, err := uc.reservationRepo.Create(ctx, reservation)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateReservation) {
			return 0, domain.StatusConfirmed, ErrDuplicateReservationForDog
		}
		return 0, domain.StatusConfirmed, fmt.Errorf("create reservation: %w", err)
	}
	return id, status, nil
}

// hasAbsoluteConflict reports whether any of the conflicts carries the
// blocking severity. ABSOLUTA incompatibilities always veto the booking;
// MEDIA/BAJA only escalate to the pending flow.
func hasAbsoluteConflict(conflicts []domain.CompatibilityConflict) bool {
	for _, conflict := range conflicts {
		if conflict.TriggerLevel == domain.IncompatibilityLevelAbsoluta {
			return true
		}
	}
	return false
}
