package reservation

import (
	"context"
	"time"

	"dogpaw/internal/domain"
)

type RegisterAdminReservationInput struct {
	activityID int
	dogID      int
	passID     int
	now        time.Time
}

func (in RegisterAdminReservationInput) ActivityID() int { return in.activityID }
func (in RegisterAdminReservationInput) DogID() int      { return in.dogID }
func (in RegisterAdminReservationInput) PassID() int     { return in.passID }
func (in RegisterAdminReservationInput) Now() time.Time  { return in.now }

func NewRegisterAdminReservationInput(activityID, dogID, passID int, now func() time.Time) (RegisterAdminReservationInput, error) {
	if activityID <= 0 { return RegisterAdminReservationInput{}, &ValidationError{Field: "activity_id"} }
	if dogID <= 0 { return RegisterAdminReservationInput{}, &ValidationError{Field: "dog_id"} }
	if passID <= 0 { return RegisterAdminReservationInput{}, &ValidationError{Field: "pass_id"} }
	if now == nil { now = time.Now }
	return RegisterAdminReservationInput{activityID: activityID, dogID: dogID, passID: passID, now: now()}, nil
}

type RegisterAdminReservationOutput struct {
	ID     int
	Status domain.ReservationStatus
}

type RegisterAdminReservationUseCase struct {
	register *RegisterReservationUseCase
}

func NewRegisterAdminReservationUseCase(transactor Transactor, activityRepo domain.ActivityRepository, dogRepo domain.DogRepository, passRepo domain.PassRepository, reservationRepo domain.ReservationRepository) *RegisterAdminReservationUseCase {
	return &RegisterAdminReservationUseCase{register: NewRegisterReservationUseCase(transactor, activityRepo, dogRepo, passRepo, reservationRepo)}
}

func (uc *RegisterAdminReservationUseCase) Execute(ctx context.Context, input RegisterAdminReservationInput) (RegisterAdminReservationOutput, error) {
	out, err := uc.register.Execute(ctx, RegisterReservationInput{
		activityID: input.ActivityID(), dogID: input.DogID(), passID: input.PassID(), now: input.Now(), adminOverride: true,
	})
	if err != nil { return RegisterAdminReservationOutput{}, err }
	return RegisterAdminReservationOutput{ID: out.ID, Status: out.Status}, nil
}
