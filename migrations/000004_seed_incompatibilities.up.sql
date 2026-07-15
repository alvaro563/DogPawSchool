-- ============================================================================
-- DogPaw - Migration 000004: Seed incompatibilities and dog associations
-- ============================================================================
-- Inserts 10 master incompatibilities and 33 dog_incompatibilities rows for
-- development, manual QA, and integration testing of the incompatibilities API.
--
-- Distribution is designed to exercise every code path:
--   * 9 dogs with 0 incompatibilities (test "add first incompatibility")
--   * 11 dogs with 1 (test "add a second", "remove the only one")
--   * 8 dogs with 2 (test "list multiple", "remove one among many")
--   * 2 dogs with 3 (test "list many", "remove and verify remaining")
--
-- Each incompatibility is assigned to 2-6 dogs, so DELETE /incompatibilities/:id
-- always returns 409 (incompatibility_in_use) — the FK is RESTRICT by 000002.
--
-- Incompatibility mix:
--   * 3 ABSOLUTA, 3 MEDIA, 2 BAJA, plus 2 MEDIA on reactivity categories
--   * 6 behavior categories: reactivity, fear, anxiety, aggression, prey, safety
--
-- Idempotency: this migration is NOT idempotent. The incompatibilities.name
-- UNIQUE index and the (dog_id, incompatibility_id) PRIMARY KEY both make a
-- re-run fail. Expected for a one-shot seed.
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Master incompatibilities (10)
-- ============================================================================
-- Position index (assigned by INSERT order, used by associations below):
--   0: Reactivo a machos enteros    (MEDIA)
--   1: Miedoso con extraños          (BAJA)
--   2: Ansiedad por separación       (BAJA)
--   3: Protección de recursos        (ABSOLUTA)
--   4: Agresividad alimentaria       (ABSOLUTA)
--   5: Reactivo a hembras en celo    (MEDIA)
--   6: Reactivo a bicicletas         (BAJA)
--   7: Selectivo con otros perros    (MEDIA)
--   8: Necesita bozal en grupo       (ABSOLUTA)
--   9: Agresivo con gatos            (ABSOLUTA)
WITH new_incompat AS (
    INSERT INTO incompatibilities (name, level_type)
    VALUES
        ('Reactivo a machos enteros',  'MEDIA'),
        ('Miedoso con extraños',        'BAJA'),
        ('Ansiedad por separación',     'BAJA'),
        ('Protección de recursos',      'ABSOLUTA'),
        ('Agresividad alimentaria',     'ABSOLUTA'),
        ('Reactivo a hembras en celo',  'MEDIA'),
        ('Reactivo a bicicletas',       'BAJA'),
        ('Selectivo con otros perros',  'MEDIA'),
        ('Necesita bozal en grupo',     'ABSOLUTA'),
        ('Agresivo con gatos',          'ABSOLUTA')
    RETURNING id
),
incompat_indexed AS (
    -- Map each freshly-inserted incompatibility to a stable 0-based position
    -- by insertion order. associations.incompat_position joins on this.
    SELECT id, (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM new_incompat
),
seed_users AS (
    -- The 10 seed users from migration 000003, identified by their unique
    -- emails. Filter is necessary because the demo "Demo Owner" user
    -- (and any other pre-existing data) must not be touched by this seed.
    SELECT id, name, email,
           (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM users
    WHERE email = 'admin@dogpaw.com'
       OR email LIKE '%@example.com'
)

-- ============================================================================
-- 2. dog_incompatibilities (33 associations)
-- ============================================================================
-- Each row references:
--   * owner_pos → seed_users.position (0=Carlos, 1=María, ..., 9=Elena)
--   * dog_name  → the dog's name (unique within a user's dogs; verified)
--   * incompat_pos → incompat_indexed.position (0-9)
--
-- Dogs with ZERO incompatibilities are intentionally absent (no row here):
--   Rex, Nano, Coco, Nina, Maya, Nala, Luna2, Mia, Pipo.
INSERT INTO dog_incompatibilities (dog_id, incompatibility_id)
SELECT
    d.id,
    ii.id
FROM (VALUES
    -- Carlos Admin (position 0): 1 of 3 dogs has incompat
    (0, 'Leia',  0),  -- Reactivo a machos enteros

    -- María López (position 1): 2 of 3 dogs, 4 associations
    (1, 'Luna',  1),  -- Miedoso con extraños
    (1, 'Luna',  3),  -- Protección de recursos
    (1, 'Max',   7),  -- Selectivo con otros perros
    (1, 'Max',   4),  -- Agresividad alimentaria

    -- Javier Ruiz (position 2): 2 of 3 dogs, 3 associations
    (2, 'Rocky', 1),  -- Miedoso con extraños
    (2, 'Rocky', 3),  -- Protección de recursos
    (2, 'Toby',  2),  -- Ansiedad por separación

    -- Laura García (position 3): 2 of 3 dogs, 3 associations
    (3, 'Kira',  1),  -- Miedoso con extraños
    (3, 'Kira',  7),  -- Selectivo con otros perros
    (3, 'Thor',  5),  -- Reactivo a hembras en celo

    -- Diego Fernández (position 4): 3 of 3 dogs, 4 associations
    (4, 'Bruno', 8),  -- Necesita bozal en grupo
    (4, 'Bruno', 3),  -- Protección de recursos
    (4, 'Lola',  0),  -- Reactivo a machos enteros
    (4, 'Taco',  1),  -- Miedoso con extraños
    (4, 'Taco',  6),  -- Reactivo a bicicletas

    -- Ana Martínez (position 5): 2 of 3 dogs, 3 associations
    (5, 'Simba', 1),  -- Miedoso con extraños
    (5, 'Simba', 7),  -- Selectivo con otros perros
    (5, 'Bimba', 2),  -- Ansiedad por separación

    -- Pedro Sánchez (position 6): 2 of 3 dogs, 4 associations
    (6, 'Koko',  9),  -- Agresivo con gatos
    (6, 'Rambo', 1),  -- Miedoso con extraños
    (6, 'Rambo', 7),  -- Selectivo con otros perros
    (6, 'Rambo', 4),  -- Agresividad alimentaria

    -- Sara Gómez (position 7): 2 of 3 dogs, 4 associations
    (7, 'Odin',  5),  -- Reactivo a hembras en celo
    (7, 'Odin',  3),  -- Protección de recursos
    (7, 'Odin',  7),  -- Selectivo con otros perros
    (7, 'Laika', 9),  -- Agresivo con gatos

    -- Miguel Torres (position 8, INACTIVE owner): 2 of 3 dogs, 3 associations
    (8, 'Zeus',  8),  -- Necesita bozal en grupo
    (8, 'Kira2', 0),  -- Reactivo a machos enteros
    (8, 'Kira2', 2),  -- Ansiedad por separación

    -- Elena Navarro (position 9): 3 of 3 dogs, 3 associations
    -- Roxy and Flora are is_active=false (incompatible dogs still persist)
    (9, 'Roxy',  2),  -- Ansiedad por separación
    (9, 'Duke',  6),  -- Reactivo a bicicletas
    (9, 'Flora', 9)   -- Agresivo con gatos
) AS a(owner_pos, dog_name, incompat_pos)
JOIN seed_users su       ON su.position        = a.owner_pos
JOIN dogs d              ON d.name             = a.dog_name
                        AND d.user_id         = su.id
JOIN incompat_indexed ii ON ii.position        = a.incompat_pos;

-- ============================================================================
-- 3. Refresh planner statistics
-- ============================================================================
ANALYZE incompatibilities;
ANALYZE dog_incompatibilities;

COMMIT;
