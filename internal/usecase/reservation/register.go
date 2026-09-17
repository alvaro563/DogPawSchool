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
	userID        int
	activityID    int
	dogID         int
	passID        int
	now           time.Time
	adminOverride bool
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

// PendingReason is a stable, language-neutral description of why a
// reservation was held in StatusPendingToConfirm. The handler translates
// each Code into a user-facing message; the domain and use case layers
// never produce localized text.
//
// DogIDs references the dogs that triggered this reason (usually the
// incoming candidate + the existing dog(s) involved in the conflict),
// so the handler can resolve names from its in-memory store.
type PendingReason struct {
	Code   string
	DogIDs []int
}

// SexNeuteredReasonPrefix is prepended to every sex/neutered reason
// code so the handler can route them through a dedicated translator
// while still distinguishing them from non-sex reasons (e.g. the
// special-condition reason).
const SexNeuteredReasonPrefix = "sex_neutered:"

// RegisterReservationOutput is the result of a successful create. The
// Status tells the client whether the booking was confirmed outright or
// held pending an admin decision (PENDING_TO_CONFIRM). PendingReasons
// is non-empty only when Status == StatusPendingToConfirm and lists
// each reason the reservation was held for, in evaluation order.
type RegisterReservationOutput struct {
	ID             int
	Status         domain.ReservationStatus
	PendingReasons []PendingReason
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
		id, status, pendingReasons, err := uc.runInTx(txCtx, input, input.Now())
		if err != nil {
			return err
		}
		output = RegisterReservationOutput{ID: id, Status: status, PendingReasons: pendingReasons}
		return nil
	})
	if err != nil {
		return RegisterReservationOutput{}, err
	}
	return output, nil
}

func (uc *RegisterReservationUseCase) runInTx(ctx context.Context, input RegisterReservationInput, now time.Time) (int, domain.ReservationStatus, []PendingReason, error) {
	// 1. Activity must exist and be in the future. FOR UPDATE locks the
	// activity row so two concurrent registrations serialize on the
	// capacity check below (materialised conflict for B2).
	activity, err := uc.activityRepo.GetByIDForUpdate(ctx, input.ActivityID(), 0, true)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, nil, ErrInvalidActivity
		}
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("get activity %d: %w", input.ActivityID(), err)
	}
	if activity == nil {
		return 0, domain.StatusConfirmed, nil, ErrInvalidActivity
	}
	if activity.IsInThePast(now) {
		return 0, domain.StatusConfirmed, nil, ErrActivityInPast
	}

	// 2. Activity must have remaining capacity. Pending bookings hold
	// their slot until the admin decides, so capacity is measured over
	// HoldsSlot (confirmed or pending).
	existing, err := uc.reservationRepo.ListByActivity(ctx, input.ActivityID())
	if err != nil {
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("list reservations for activity %d: %w", input.ActivityID(), err)
	}
	slotHolders := make([]int, 0, len(existing))
	for _, reservation := range existing {
		if reservation.HoldsSlot() {
			slotHolders = append(slotHolders, reservation.DogID())
		}
	}
	if activity.IsFull(len(slotHolders)) {
		return 0, domain.StatusConfirmed, nil, ErrActivityFull
	}

	// 3. Dog must exist and be owned by UserID.
	dog, err := uc.dogRepo.GetByID(ctx, input.DogID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, nil, ErrInvalidDog
		}
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("get dog %d: %w", input.DogID(), err)
	}
	if dog == nil {
		return 0, domain.StatusConfirmed, nil, ErrInvalidDog
	}
	if !input.adminOverride && dog.UserID() != input.UserID() {
		return 0, domain.StatusConfirmed, nil, ErrInvalidDog
	}
	// 3a. Size match: when the activity targets a single size bracket
	// (SOCIALIZATION_GROUP / ROUTE with sizeTarget != nil), the dog
	// must be of that bracket. Admin override bypasses this check
	// for emergency bookings. Placed BEFORE the compatibility check
	// because size mismatch is a deterministic "you cannot book" —
	// not a "hold pending review".
	if !input.adminOverride && !activity.IsTargetedAtSize(dog.SizeBracket()) {
		return 0, domain.StatusConfirmed, nil, ErrDogSizeMismatch
	}

	// 3b. Compatibility check (sex/neutered + triggers). Load the dogs
	// already holding a slot in this activity so we can evaluate both
	// the sex/neutered rule and the trigger->trait collisions against
	// the candidate. The load is skipped under admin override.
	status := domain.StatusConfirmed
	pendingReasons := make([]PendingReason, 0)
	if !input.adminOverride {
		others, err := uc.dogRepo.GetByIDs(ctx, slotHolders)
		if err != nil {
			return 0, domain.StatusConfirmed, nil, fmt.Errorf("get dogs holding a slot in activity %d: %w", input.ActivityID(), err)
		}

		// 3b.i. Sex/neutered check. The rule is symmetric and applies
		// only to male-male pairs. Two intact males BLOCK the booking;
		// every other male-male combination (castrated+castrated,
		// intact+castrated) escalates to StatusPendingToConfirm with a
		// per-pair PendingReason. Female pairs are never affected.
		sexNeuteredConflicts := make([]domain.SexNeuteredConflict, 0)
		for _, other := range others {
			sexNeuteredConflicts = append(sexNeuteredConflicts, dog.SexNeuteredConflictsWith(other)...)
		}
		if hasBlockingSexConflict(sexNeuteredConflicts) {
			blocking := collectBlockingDogs(sexNeuteredConflicts, others)
			return 0, domain.StatusConfirmed, nil, &SexNeuteredConflictError{
				IncomingDog:  dog,
				BlockingDogs: blocking,
				Conflicts:    sexNeuteredConflicts,
			}
		}
		for _, c := range sexNeuteredConflicts {
			if c.Reason() != "" {
				status = domain.StatusPendingToConfirm
				pendingReasons = append(pendingReasons, PendingReason{
					Code:   SexNeuteredReasonPrefix + c.Reason(),
					DogIDs: []int{c.IncomingDogID, c.ExistingDogID},
				})
			}
		}

		// 3b.ii. Trigger->trait compatibility check. The severity of
		// the loudest conflict decides:
		//   - ABSOLUTA: the booking is blocked.
		//   - MEDIA/BAJA: the booking is kept pending, slot held, until
		//     an admin confirms or rejects it.
		conflicts := make([]domain.CompatibilityConflict, 0, len(others))
		for _, other := range others {
			conflicts = append(conflicts, dog.ConflictsWith(other)...)
		}
		if len(conflicts) > 0 {
			if hasAbsoluteConflict(conflicts) {
				return 0, domain.StatusConfirmed, nil, &IncompatibleDogsError{Conflicts: conflicts}
			}
			status = domain.StatusPendingToConfirm
		}

		// 3b.iii. Special-condition flag. When the candidate dog has a
		// special condition, the booking is escalated to
		// StatusPendingToConfirm for admin review, like triggers and
		// sex/neutered conflicts. Admin override bypasses this together
		// with the rest of the compatibility block.
		if dog.HasSpecialCondition() {
			status = domain.StatusPendingToConfirm
			pendingReasons = append(pendingReasons, PendingReason{
				Code:   domain.ReasonHasSpecialCondition,
				DogIDs: []int{dog.ID()},
			})
		}
	}

	// 4. Pass must exist, be owned by UserID, not be exhausted, and
	// not be expired. FOR UPDATE locks the pass row so two concurrent
	// consumptions serialise (prevents B3 lost update).
	pass, err := uc.passRepo.GetByIDForUpdate(ctx, input.PassID())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.StatusConfirmed, nil, ErrInvalidPass
		}
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("get pass %d: %w", input.PassID(), err)
	}
	if pass == nil {
		return 0, domain.StatusConfirmed, nil, ErrInvalidPass
	}
	if !input.adminOverride && pass.UserID() != input.UserID() {
		return 0, domain.StatusConfirmed, nil, ErrInvalidPass
	}
	if pass.IsExhausted() {
		return 0, domain.StatusConfirmed, nil, ErrPassExhausted
	}
	if pass.IsExpired(now) {
		return 0, domain.StatusConfirmed, nil, ErrPassExpired
	}

	// 4b. Under admin override, the per-entity ownership checks above
	// are skipped, but the dog and the pass must still belong to the
	// same user. Admin authority lets one operator act on behalf of
	// any user; it does not let them combine one user's dog with
	// another user's pass: the booking would still consume a session
	// from a pass that does not belong to the dog's owner. Under
	// normal flow this case is already impossible (dog.UserID() ==
	// userID AND pass.UserID() == userID), so the check is only
	// meaningful on the admin path.
	if input.adminOverride && dog.UserID() != pass.UserID() {
		return 0, domain.StatusConfirmed, nil, ErrDogPassOwnerMismatch
	}

	// 5. Consume one pass session. The audit movement is recorded on
	// the aggregate; Update flushes it together with the new counter.
	reason := fmt.Sprintf("Reservation: activity %d, dog %d", input.ActivityID(), input.DogID())
	if _, err := pass.ConsumeSession(reason, now); err != nil {
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("consume pass %d: %w", input.PassID(), err)
	}

	// 6. Persist the pass: counter + audit row, atomically.
	if err := uc.passRepo.Update(ctx, pass); err != nil {
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("update pass %d: %w", input.PassID(), err)
	}

	// 7. Create the reservation in the resolved status.
	reservation, err := domain.NewReservationWithStatus(0, input.ActivityID(), input.DogID(), input.PassID(), status, now)
	if err != nil {
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("build reservation: %w", err)
	}
	id, err := uc.reservationRepo.Create(ctx, reservation)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateReservation) {
			return 0, domain.StatusConfirmed, nil, ErrDuplicateReservationForDog
		}
		return 0, domain.StatusConfirmed, nil, fmt.Errorf("create reservation: %w", err)
	}
	return id, status, pendingReasons, nil
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

// hasBlockingSexConflict reports whether any sex/neutered conflict is
// the blocking variety: two intact (non-castrated) males. Every other
// male-male combination is non-blocking and surfaces via
// StatusPendingToConfirm.
func hasBlockingSexConflict(conflicts []domain.SexNeuteredConflict) bool {
	for _, conflict := range conflicts {
		if conflict.IsBlocker() {
			return true
		}
	}
	return false
}

// collectBlockingDogs returns the dog entities that participated in a
// blocking sex/neutered conflict. Order matches the order of conflicts
// (first occurrence of each dog wins, dedup by id). Always returns at
// least one element when at least one blocking conflict exists.
func collectBlockingDogs(conflicts []domain.SexNeuteredConflict, others []*domain.Dog) []*domain.Dog {
	byID := make(map[int]*domain.Dog, len(others))
	for _, o := range others {
		byID[o.ID()] = o
	}
	out := make([]*domain.Dog, 0, len(conflicts))
	seen := make(map[int]bool, len(conflicts))
	for _, c := range conflicts {
		if !c.IsBlocker() {
			continue
		}
		if seen[c.ExistingDogID] {
			continue
		}
		if d, ok := byID[c.ExistingDogID]; ok {
			out = append(out, d)
			seen[c.ExistingDogID] = true
		}
	}
	return out
}
