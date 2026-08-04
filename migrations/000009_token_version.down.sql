-- ============================================================================
-- DogPaw - Migration 000009 down: remove token_version
-- ============================================================================

ALTER TABLE users DROP COLUMN token_version;
