package reservation

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ConfirmPendingReservationInput is the validated admin command to
// promote a PENDING_TO_CONFIRM reservation to CONFIRMED.
type ConfirmPendingReservationInput struct {
	reservationID int
}

func (in ConfirmPendingReservationInput) ReservationID() int { return in.reservationID }

// NewConfirmPendingReservationInput validates reservationID > 0.
func NewConfirmPendingReservationInput(reservationID int) (ConfirmPendingReservationInput, error) {
	if reservationID <= 0 {
		return ConfirmPendingReservationInput{}, &ValidationError{Field: "reservation_id"}
	}
	return ConfirmPendingReservationInput{reservationID: reservationID}, nil
}

// MustNewConfirmPendingReservationInput panics on validation error. For tests.
func MustNewConfirmPendingReservationInput(reservationID int) ConfirmPendingReservationInput {
	in, err := NewConfirmPendingReservationInput(reservationID)
	if err != nil {
		panic(err)
	}
	return in
}

// ConfirmPendingReservationOutput is the result of a successful confirm.
type ConfirmPendingReservationOutput struct {
	Reservation *domain.Reservation
}

// ConfirmPendingReservationUseCase lets an admin promote a reservation
// that was held in StatusPendingToConfirm (created with only MEDIA/BAJA
// compatibility conflicts) to StatusConfirmed. The slot is already
// held, so no capacity or pass bookkeeping is needed.
type ConfirmPendingReservationUseCase struct {
	transactor      Transactor
	reservationRepo domain.ReservationRepository
}

func NewConfirmPendingReservationUseCase(
	transactor Transactor,
	reservationRepo domain.ReservationRepository,
) *ConfirmPendingReservationUseCase {
	return &ConfirmPendingReservationUseCase{
		transactor:      transactor,
		reservationRepo: reservationRepo,
	}
}

func (uc *ConfirmPendingReservationUseCase) Execute(ctx context.Context, input ConfirmPendingReservationInput) (ConfirmPendingReservationOutput, error) {
	var output ConfirmPendingReservationOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		reservation, err := uc.reservationRepo.GetByID(txCtx, input.ReservationID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("get reservation %d: %w", input.ReservationID(), err)
		}
		if reservation == nil {
			return ErrNotFound
		}
		if !reservation.IsPending() {
			return fmt.Errorf("%w: current status is %s", ErrNotPending, reservation.Status())
		}
		if err := reservation.ConfirmPending(); err != nil {
			return fmt.Errorf("%w: %v", ErrNotPending, err)
		}
		if err := uc.reservationRepo.Update(txCtx, reservation); err != nil {
			return fmt.Errorf("update reservation %d: %w", input.ReservationID(), err)
		}
		output = ConfirmPendingReservationOutput{Reservation: reservation}
		return nil
	})
	if err != nil {
		return ConfirmPendingReservationOutput{}, err
	}
	return output, nil
}
