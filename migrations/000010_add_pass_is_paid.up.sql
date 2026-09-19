BEGIN;
ALTER TABLE passes ADD COLUMN is_paid BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN passes.is_paid IS 'Whether the pass has been paid by the user';
COMMIT;
