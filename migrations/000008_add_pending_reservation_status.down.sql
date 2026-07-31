-- ============================================================================
-- DogPaw - Migration 000008 down: remove PENDING_TO_CONFIRM
-- ============================================================================
-- Postgres cannot delete an enum value, so the type is recreated without
-- it. Any PENDING_TO_CONFIRM rows are folded back into CANCELLED_IN_TIME
-- first so the cast does not fail.
-- ============================================================================

BEGIN;

UPDATE reservations
SET status = 'CANCELLED_IN_TIME'
WHERE status = 'PENDING_TO_CONFIRM';

ALTER TABLE reservations ALTER COLUMN status DROP DEFAULT;
ALTER TABLE reservations ALTER COLUMN status TYPE TEXT;

DROP TYPE reservation_status;

CREATE TYPE reservation_status AS ENUM (
    'CONFIRMED',
    'COMPLETED',
    'CANCELLED_IN_TIME',
    'CANCELLED_LATE',
    'FORGIVEN',
    'NO_SHOW'
);

ALTER TABLE reservations ALTER COLUMN status TYPE reservation_status USING status::reservation_status;
ALTER TABLE reservations ALTER COLUMN status SET DEFAULT 'CONFIRMED';

COMMIT;
