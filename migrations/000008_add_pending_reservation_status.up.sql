-- ============================================================================
-- DogPaw - Migration 000008: PENDING_TO_CONFIRM reservation status
-- ============================================================================
-- The Traits & Triggers pending flow (000007) lets MEDIA/BAJA conflicts
-- hold a slot until an admin confirms or rejects the booking. That
-- reservation is persisted in PENDING_TO_CONFIRM, a value that the
-- reservation_status enum (000001) never gained. Add it.
-- ============================================================================

ALTER TYPE reservation_status ADD VALUE 'PENDING_TO_CONFIRM';
