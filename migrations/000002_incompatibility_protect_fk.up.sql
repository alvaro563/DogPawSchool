-- ============================================================================
-- DogPaw - Migration 000002: Protect Incompatibility FK from DELETE
-- ============================================================================
-- Changes the FK constraint on dog_incompatibilities.incompatibility_id from
-- CASCADE to RESTRICT, so that deleting an incompatibility that is still
-- referenced by at least one dog fails at the DB level with a foreign key
-- violation. The application translates pg 23503 into ErrIncompatibilityInUse
-- which maps to HTTP 409.
-- ============================================================================

BEGIN;

ALTER TABLE dog_incompatibilities
    DROP CONSTRAINT fk_dog_incompat_incompat;

ALTER TABLE dog_incompatibilities
    ADD CONSTRAINT fk_dog_incompat_incompat
        FOREIGN KEY (incompatibility_id)
        REFERENCES incompatibilities (id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE;

COMMIT;
