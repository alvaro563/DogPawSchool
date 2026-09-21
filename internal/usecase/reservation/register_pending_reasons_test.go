package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

// TestRegisterReservationUseCase_PendingBookingCallsSavePendingReasons
// verifies the audit-trail contract: when a booking escalates to
// StatusPendingToConfirm, the use case MUST call SavePendingReasons
// exactly once with the same reasons that appear in the output,
// in the same evaluation order. The call must happen inside the
// transaction so a write failure here rolls the reservation back.
//
// Confirmed bookings (no pending reasons) MUST NOT call
// SavePendingReasons.
func TestRegisterReservationUseCase_PendingBookingCallsSavePendingReasons(t *testing.T) {
	t.Parallel()
	candidate := newDogWithSex(t, 20, 1, domain.SexMale, false)
	other := newDogWithSex(t, 21, 99, domain.SexMale, true)
	activityRepo, dogRepo, passRepo, reservationRepo := sexNeuteredFlowStubs(candidate, other)
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		assert.Equal(t, domain.StatusPendingToConfirm, r.Status())
		return 99, nil
	}

	var saveCalls int
	var savedReasons []domain.PendingReason
	var savedResID int
	reservationRepo.savePendingReasons = func(_ context.Context, reservationID int, reasons []domain.PendingReason) error {
		saveCalls++
		savedResID = reservationID
		// Copy the slice so later mutations don't bleed into our assertion.
		savedReasons = append(savedReasons[:0], reasons...)
		return nil
	}

	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	out, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)

	assert.Equal(t, domain.StatusPendingToConfirm, out.Status)
	require.Len(t, out.PendingReasons, 1)
	assert.Equal(t, SexNeuteredReasonPrefix+domain.ReasonIntactVsCastrated, out.PendingReasons[0].Code)
	assert.Equal(t, []int{20, 21}, out.PendingReasons[0].DogIDs)

	// SavePendingReasons called exactly once with the booking id and
	// the same reasons.
	assert.Equal(t, 1, saveCalls, "SavePendingReasons should be called once for pending bookings")
	assert.Equal(t, 99, savedResID, "save called with the reservation id returned by Create")
	require.Len(t, savedReasons, 1)
	assert.Equal(t, out.PendingReasons[0].Code, savedReasons[0].Code)
	assert.Equal(t, out.PendingReasons[0].DogIDs, savedReasons[0].DogIDs)
}

// TestRegisterReservationUseCase_ConfirmedBookingSkipsSavePendingReasons
// is the negative companion: a booking that confirms outright must
// not invoke SavePendingReasons (no audit trail for non-pending).
func TestRegisterReservationUseCase_ConfirmedBookingSkipsSavePendingReasons(t *testing.T) {
	t.Parallel()
	candidate := dogWithTrigger(20, 1, mustTrigger(1, "Reactivo a machos enteros", domain.IncompatibilityLevelMedia, "MACHO_ENTERO"))
	other := validDog(21, 1)
	activityRepo, dogRepo, passRepo, reservationRepo := registerFlowStubs(t, candidate, other)
	reservationRepo.create = func(_ context.Context, r *domain.Reservation) (int, error) {
		assert.Equal(t, domain.StatusConfirmed, r.Status())
		return 50, nil
	}
	var saveCalls int
	reservationRepo.savePendingReasons = func(context.Context, int, []domain.PendingReason) error {
		saveCalls++
		return nil
	}

	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	out, err := uc.Execute(context.Background(), validRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, out.Status)
	assert.Empty(t, out.PendingReasons)
	assert.Equal(t, 0, saveCalls, "SavePendingReasons must NOT be called for confirmed bookings")
}

// TestRegisterReservationUseCase_PendingReasonsSaveFailureRollsBack
// verifies the transactional contract: a SavePendingReasons error
// propagates to the caller, the booking is NOT exposed in the
// output, and the use case does NOT swallow the error.
//
// This is the audit-trail equivalent of the "no half-state" rule
// the booking flow already enforces for pass / reservation
// persistence: if the audit row cannot be written, the booking
// cannot exist.
func TestRegisterReservationUseCase_PendingReasonsSaveFailureRollsBack(t *testing.T) {
	t.Parallel()
	candidate := newDogWithSex(t, 20, 1, domain.SexMale, false)
	other := newDogWithSex(t, 21, 99, domain.SexMale, true)
	activityRepo, dogRepo, passRepo, reservationRepo := sexNeuteredFlowStubs(candidate, other)
	reservationRepo.create = func(context.Context, *domain.Reservation) (int, error) {
		return 99, nil
	}
	reservationRepo.savePendingReasons = func(context.Context, int, []domain.PendingReason) error {
		return errors.New("audit table write failed")
	}

	uc := newRegisterUseCase(activityRepo, dogRepo, passRepo, reservationRepo, nil)
	_, err := uc.Execute(context.Background(), validRegisterInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit table write failed")
}
