-- Replaces the unconditional UNIQUE constraint with a partial unique
-- index whose predicate matches the HoldsSlot() semantic used everywhere
-- else in the system. Active bookings (CONFIRMED, PENDING_TO_CONFIRM)
-- still cannot duplicate for the same (activity_id, dog_id); cancelled,
-- no-show, forgiven and completed rows are excluded so a user can
-- re-register a dog that was previously removed from an activity.

ALTER TABLE reservations DROP CONSTRAINT uniq_reservation_dog_per_activity;

CREATE UNIQUE INDEX uniq_reservation_dog_active
    ON reservations (activity_id, dog_id)
    WHERE status IN ('CONFIRMED', 'PENDING_TO_CONFIRM');

COMMENT ON INDEX uniq_reservation_dog_active IS
    'Garantiza que un perro solo tiene UNA reserva activa por actividad. '
    'Las reservas terminales (CANCELLED_IN_TIME, CANCELLED_LATE, '
    'FORGIVEN, NO_SHOW, COMPLETED) están excluidas para permitir '
    're-registro tras cancelar o marcar no-show.';
