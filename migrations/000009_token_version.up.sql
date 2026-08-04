-- ============================================================================
-- DogPaw - Migration 000009: token_version for JWT revocation
-- ============================================================================
-- Adds a token_version counter to the users table. Every time the user
-- changes their password the counter is incremented, and the JWT carries
-- the version it was minted with. On each authenticated request the
-- middleware compares the token's version against the user's current
-- version: a mismatch means the token was revoked (password changed after
-- the token was issued) and the request is rejected with 401.
-- ============================================================================

ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
