-- ============================================================================
-- DogPaw - Migration 000010 down: restore incompatibility kind
-- ============================================================================

BEGIN;

CREATE TYPE incompatibility_kind AS ENUM ('TRAIT', 'TRIGGER');

ALTER TABLE incompatibilities ADD COLUMN kind incompatibility_kind;

-- Reclassify: rows with code -> TRAIT, rows with target_trait_code -> TRIGGER
UPDATE incompatibilities
SET kind = CASE
    WHEN code IS NOT NULL THEN 'TRAIT'::incompatibility_kind
    ELSE 'TRIGGER'::incompatibility_kind
END;

ALTER TABLE incompatibilities ALTER COLUMN kind SET NOT NULL;

ALTER TABLE incompatibilities
    ADD CONSTRAINT incompatibilities_trait_has_code
        CHECK (kind <> 'TRAIT' OR (code IS NOT NULL AND LENGTH(TRIM(code)) > 0));

ALTER TABLE incompatibilities
    ADD CONSTRAINT incompatibilities_trigger_has_target
        CHECK (kind <> 'TRIGGER' OR (target_trait_code IS NOT NULL AND LENGTH(TRIM(target_trait_code)) > 0));

CREATE INDEX idx_incompatibilities_kind ON incompatibilities (kind);

COMMIT;
