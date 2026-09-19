-- ============================================================================
-- DogPaw - Migration 000007: Traits & Triggers (incompatibility model)
-- ============================================================================
-- Extends the existing `incompatibilities` table with a Traits & Triggers
-- model, WITHOUT self-referencing foreign keys:
--
--   * kind: TRAIT | TRIGGER
--       - TRAIT   = a characteristic/tag the dog presents (MACHO_ENTERO,
--                   ALTA_ENERGIA, TAMANO_GRANDE...). Stored in `code`.
--       - TRIGGER = something the dog does not tolerate in class. Stored
--                   with a `target_trait_code` that points to the CODE of
--                   the trait it reacts to.
--   * The link trigger->trait is by code (a plain string), validated at the
--     application layer (the repo checks the target exists and is a TRAIT).
--   * dog_incompatibilities keeps being the single join: a dog's attached
--     TRAIT rows are its traits, its attached TRIGGER rows are its triggers.
--
-- Existing rows keep their identity and dog associations: informational
-- categories are reclassified as TRAITs, relational ones as TRIGGERs.
--
-- The CHECK constraints are added AFTER the reclassification: the new
-- column defaults existing rows to kind='TRIGGER', so enforcing
-- "trigger must have a target" before the backfill would fail on every
-- pre-existing row.
-- ============================================================================

BEGIN;

CREATE TYPE incompatibility_kind AS ENUM ('TRAIT', 'TRIGGER');

ALTER TABLE incompatibilities
    ADD COLUMN kind              incompatibility_kind NOT NULL DEFAULT 'TRIGGER',
    ADD COLUMN code              TEXT,
    ADD COLUMN target_trait_code TEXT;

-- ============================================================================
-- Seed: master TRAITs (new rows)
-- ============================================================================
INSERT INTO incompatibilities (kind, code, name, level_type)
VALUES
    ('TRAIT', 'MACHO_ENTERO',     'Macho entero (no castrado)', 'BAJA'),
    ('TRAIT', 'HEMBRA_EN_CELO',   'Hembra en celo',             'BAJA'),
    ('TRAIT', 'OTRO_PERRO',       'Otros perros',               'BAJA'),
    ('TRAIT', 'ALTA_ENERGIA',     'Alta energía',               'BAJA'),
    ('TRAIT', 'TAMANO_GRANDE',    'Tamaño grande',              'BAJA');

-- ============================================================================
-- Reclassify the seeded incompatibilities from 000004
-- ============================================================================
-- Informational categories -> TRAIT (they only block if a TRIGGER targets them).
UPDATE incompatibilities
SET kind = 'TRAIT', code = 'MIEDOSO'
WHERE name = 'Miedoso con extraños';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'ANSIOSO'
WHERE name = 'Ansiedad por separación';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'NECESITA_BOZAL'
WHERE name = 'Necesita bozal en grupo';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'PROTEGE_RECURSOS'
WHERE name = 'Protección de recursos';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'AGRESIVO_COMIDA'
WHERE name = 'Agresividad alimentaria';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'AGRESIVO_GATOS'
WHERE name = 'Agresivo con gatos';

UPDATE incompatibilities
SET kind = 'TRAIT', code = 'REACTIVO_BICIS'
WHERE name = 'Reactivo a bicicletas';

-- Relational categories -> TRIGGER pointing at the trait code they react to.
UPDATE incompatibilities
SET kind = 'TRIGGER', target_trait_code = 'MACHO_ENTERO'
WHERE name = 'Reactivo a machos enteros';

UPDATE incompatibilities
SET kind = 'TRIGGER', target_trait_code = 'HEMBRA_EN_CELO'
WHERE name = 'Reactivo a hembras en celo';

UPDATE incompatibilities
SET kind = 'TRIGGER', target_trait_code = 'OTRO_PERRO'
WHERE name = 'Selectivo con otros perros';

-- ============================================================================
-- Invariant checks + indexes (added now that the backfill is complete)
-- ============================================================================
-- A TRAIT always carries its identifying code.
ALTER TABLE incompatibilities
    ADD CONSTRAINT incompatibilities_trait_has_code
        CHECK (kind <> 'TRAIT' OR (code IS NOT NULL AND LENGTH(TRIM(code)) > 0));

-- A TRIGGER always points to the code of the trait it detests.
ALTER TABLE incompatibilities
    ADD CONSTRAINT incompatibilities_trigger_has_target
        CHECK (kind <> 'TRIGGER' OR (target_trait_code IS NOT NULL AND LENGTH(TRIM(target_trait_code)) > 0));

CREATE UNIQUE INDEX idx_incompatibilities_code   ON incompatibilities (code) WHERE code IS NOT NULL;
CREATE INDEX        idx_incompatibilities_kind   ON incompatibilities (kind);
CREATE INDEX        idx_incompatibilities_target ON incompatibilities (target_trait_code);

ANALYZE incompatibilities;

COMMIT;
