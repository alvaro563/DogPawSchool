-- ============================================================================
-- DogPaw - Migration 000010: remove incompatibility kind column
-- ============================================================================
-- The kind column is redundant: a row is a trait when `code IS NOT NULL`
-- and a trigger when `target_trait_code IS NOT NULL`. The application
-- now discriminates by which list the row is attached to on the dog
-- (dog.traits vs dog.incompatibilities) and by direct code comparison
-- (trigger.TargetTraitCode() == trait.Code()).
-- ============================================================================

BEGIN;

ALTER TABLE incompatibilities DROP CONSTRAINT IF EXISTS incompatibilities_trait_has_code;
ALTER TABLE incompatibilities DROP CONSTRAINT IF EXISTS incompatibilities_trigger_has_target;

ALTER TABLE incompatibilities DROP COLUMN kind;

DROP TYPE IF EXISTS incompatibility_kind;

DROP INDEX IF EXISTS idx_incompatibilities_kind;

COMMIT;
