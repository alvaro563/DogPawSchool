-- ============================================================================
-- DogPaw - Migration 000007: Traits & Triggers (ROLLBACK)
-- ============================================================================
-- Deletes the master TRAIT rows created by the UP migration (the
-- reclassified original incompatibilities keep their rows) and then drops
-- the added columns, constraints, indexes and the enum type.
-- ============================================================================

BEGIN;

-- The 5 master TRAITs created in UP were never attached to any dog, so the
-- RESTRICT FK from dog_incompatibilities does not block this delete.
DELETE FROM incompatibilities
WHERE kind = 'TRAIT'
  AND name IN ('Macho entero (no castrado)', 'Hembra en celo', 'Otros perros',
               'Alta energía', 'Tamaño grande');

DROP INDEX IF EXISTS idx_incompatibilities_target;
DROP INDEX IF EXISTS idx_incompatibilities_kind;
DROP INDEX IF EXISTS idx_incompatibilities_code;

ALTER TABLE incompatibilities
    DROP CONSTRAINT IF EXISTS incompatibilities_trigger_has_target;
ALTER TABLE incompatibilities
    DROP CONSTRAINT IF EXISTS incompatibilities_trait_has_code;

ALTER TABLE incompatibilities
    DROP COLUMN IF EXISTS target_trait_code,
    DROP COLUMN IF EXISTS code,
    DROP COLUMN IF EXISTS kind;

DROP TYPE IF EXISTS incompatibility_kind;

COMMIT;
