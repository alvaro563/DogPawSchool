BEGIN;

-- ============================================================================
-- Normalize email to lowercase + functional unique index.
--
-- Background: previously the UNIQUE index on users(email) was case-sensitive,
-- so "Ana@x.com" and "ana@x.com" were two distinct accounts. That violates
-- the user expectation that email is a single identifier regardless of case,
-- and it weakens login (an attacker can register Ana@x.com to shadow a
-- legitimate ana@x.com).
--
-- This migration:
--   1. Renames any duplicate (case-insensitive) rows to .duplicate.<id>
--      tombstones, KEEPING the OLDEST row (smallest id) as the canonical one.
--      The tombstone rows are kept (not deleted) so the operator can review
--      and merge or contact them manually.
--   2. Lower-cases every existing email in users and invitations.
--   3. Replaces the case-sensitive unique index with a UNIQUE INDEX on the
--      functional expression lower(email).
-- ============================================================================

-- 1a. Tombstone duplicates in users (keeping the oldest by id).
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT id, email
        FROM users u
        WHERE EXISTS (
            SELECT 1 FROM users u2
            WHERE lower(u2.email) = lower(u.email)
              AND u2.id < u.id
        )
    LOOP
        -- Append the id to make the row's lower(email) unique without
        -- losing the original data.
        UPDATE users
        SET email = lower(email) || '.duplicate.' || id
        WHERE id = r.id;
    END LOOP;
END $$;

-- 1b. Lower-case every remaining email in users.
UPDATE users SET email = lower(email) WHERE email <> lower(email);

-- 1c. Lower-case every email in invitations (no UNIQUE constraint there,
--     just a plain index for lookups; case-insensitive matching will now
--     succeed on lower(email)).
UPDATE invitations SET email = lower(email) WHERE email <> lower(email);

-- 2. Drop the case-sensitive unique index on users.email.
DROP INDEX IF EXISTS idx_users_email;

-- 3. Create the functional UNIQUE index. From now on, two rows cannot
--    differ only in case; the second insert/update fails with a unique
--    violation (mapped to domain.ErrDuplicateEmail by the repo).
CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));

-- 4. Replace the case-sensitive lookup index on invitations with a
--    functional one so ListByEmail(lower(email)) hits an index.
DROP INDEX IF EXISTS idx_invitations_email;
CREATE INDEX idx_invitations_email_lower ON invitations (lower(email));

-- 5. CHECK constraints continue to apply (case-insensitive by virtue of
--    ~*). No changes needed.

COMMENT ON INDEX idx_users_email_lower
    IS 'Functional UNIQUE: enforces case-insensitive uniqueness of email.';
COMMENT ON INDEX idx_invitations_email_lower
    IS 'Functional index: speeds up case-insensitive ListByEmail(lower(email)).';

COMMIT;
