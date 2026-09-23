BEGIN;

-- Reverse of 000017. Restores the case-sensitive unique index. Any data
-- that became ".duplicate.<id>" tombstone in the up migration is left
-- alone — restoring those would require manual reconciliation, which is
-- out of scope for a down migration.
DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email ON users (email);

DROP INDEX IF EXISTS idx_invitations_email_lower;
CREATE INDEX idx_invitations_email ON invitations (email);

COMMIT;
