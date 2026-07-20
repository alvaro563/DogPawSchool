-- ============================================================================
-- DogPaw - Migration 000005: Seed activities, passes, movements, reservations
-- ============================================================================
-- Inserts 14 activities, 10 passes, 18 pass_movements, and 15 reservations
-- for development, manual QA, and integration testing of the activity/pass/
-- reservation endpoints.
--
-- This migration is a "demo data" companion to 000003 (users + dogs) and
-- 000004 (incompatibilities). It depends on both of them being applied
-- because it references seed users by their position (0=Carlos, 1=María, ...)
-- and seed dogs by (owner_position, name).
--
-- Distribution is designed to exercise every code path of the
-- reservation use cases:
--   * Activities: 4 SOCIALIZATION_GROUP, 4 ROUTE, 4 INDIVIDUAL_CLASS, 2 EXTRA
--   * Activities: 13 in the future, 1 in the past (-30 days)
--   * Activities: 4 with max_capacity=1 (one-booking-only, easy to test activity_full)
--   * Passes: 10 (one per seed user, no "exhausted" pass — pass_exhausted is
--     covered by the unit tests)
--   * Passes: 2 with pass_type=ESPECIFICO, 8 with GENERICO
--   * Passes: 1 with expires_at in the future (Diego), 0 with expires_at in the
--     past (impossible due to CHECK passes_expiry_after_creation)
--   * Reservations: 6 status values covered
--     CONFIRMED=9, CANCELLED_IN_TIME=2, CANCELLED_LATE=1, FORGIVEN=1,
--     COMPLETED=1, NO_SHOW=1
--   * Movements: 15 consume (-1) + 3 refund (+1) for CANCELLED_IN_TIME×2
--     and FORGIVEN×1 = 18 total
--
-- Idempotency: this migration is NOT idempotent. The activities.name is
-- not UNIQUE, but the FK constraints on reservations (activity_id, dog_id,
-- pass_id) all make a re-run fail. Expected for a one-shot seed that
-- should run after 000003 and 000004.
-- ============================================================================

BEGIN;

-- ============================================================================
-- CTEs: locate the seed users and dogs from 000003
-- ============================================================================
-- This CTE pattern is the same one 000004 uses: identify the seed users by
-- their unique emails and assign each one a stable 0-based position by
-- insertion order. All downstream joins reference these positions, not
-- hard-coded user ids, so the seed is robust against any prior data the
-- dev environment may have (e.g., the demo user from 000001).
WITH seed_users AS (
    SELECT id, name, email,
           (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM users
    WHERE email = 'admin@dogpaw.com'
       OR email LIKE '%@example.com'
),
seed_dogs AS (
    -- Join dogs to their owners so callers can reference them by
    -- (owner_position, dog_name) — much more readable than id joins.
    SELECT d.id, d.name, d.user_id, su.position AS owner_position
    FROM dogs d
    JOIN seed_users su ON su.id = d.user_id
)

-- ============================================================================
-- Section 1: Activities (14)
-- ============================================================================
-- Mix of activity_type, max_capacity, and date to exercise every code
-- path of the register / cancel / list use cases.
--
-- Position index (assigned by INSERT order, used by reservations below):
--   0:  Paseo Social Matutino      (SOCIALIZATION_GROUP, cap 8,  +7d)
--   1:  Paseo Social Vespertino    (SOCIALIZATION_GROUP, cap 6,  +10d)
--   2:  Paseo Avanzado Grupo A     (SOCIALIZATION_GROUP, cap 4,  +14d)
--   3:  Paseo Avanzado Grupo B     (SOCIALIZATION_GROUP, cap 4,  +21d)
--   4:  Ruta Río                   (ROUTE,                 cap 6,  +5d)
--   5:  Ruta Montaña               (ROUTE,                 cap 4,  +12d)
--   6:  Ruta Playa                 (ROUTE,                 cap 8,  +30d)
--   7:  Ruta Corta                 (ROUTE,                 cap 2,  +3d)
--   8:  Clase Individual Luna      (INDIVIDUAL_CLASS,      cap 1,  +2d)
--   9:  Clase Individual Max       (INDIVIDUAL_CLASS,      cap 1,  +9d)
--  10:  Clase Individual Toby      (INDIVIDUAL_CLASS,      cap 1,  +15d)
--  11:  Clase Individual Kira      (INDIVIDUAL_CLASS,      cap 1,  +18d)
--  12:  Evento Solidario           (EXTRA,                 cap 12, +45d)
--  13:  Evento Pasado (-30d)       (SOCIALIZATION_GROUP,  cap 8,  -30d) ← test activity_in_past
, new_activities AS (
    INSERT INTO activities (name, activity_type, max_capacity, location, duration_in_hours, date)
    VALUES
        ('SEED: Paseo Social Matutino',   'SOCIALIZATION_GROUP',  8, 'Parque Central',     2, NOW() + INTERVAL '7 days'),
        ('SEED: Paseo Social Vespertino', 'SOCIALIZATION_GROUP',  6, 'Parque Norte',       2, NOW() + INTERVAL '10 days'),
        ('SEED: Paseo Avanzado Grupo A',  'SOCIALIZATION_GROUP',  4, 'Sierra',             3, NOW() + INTERVAL '14 days'),
        ('SEED: Paseo Avanzado Grupo B',  'SOCIALIZATION_GROUP',  4, 'Sierra',             3, NOW() + INTERVAL '21 days'),
        ('SEED: Ruta Río',                'ROUTE',                6, 'Río',                2, NOW() + INTERVAL '5 days'),
        ('SEED: Ruta Montaña',            'ROUTE',                4, 'Montaña',            4, NOW() + INTERVAL '12 days'),
        ('SEED: Ruta Playa',              'ROUTE',                8, 'Playa',              5, NOW() + INTERVAL '30 days'),
        ('SEED: Ruta Corta',              'ROUTE',                2, 'Barrio',             1, NOW() + INTERVAL '3 days'),
        ('SEED: Clase Individual Luna',   'INDIVIDUAL_CLASS',     1, 'Centro',             1, NOW() + INTERVAL '2 days'),
        ('SEED: Clase Individual Max',    'INDIVIDUAL_CLASS',     1, 'Centro',             1, NOW() + INTERVAL '9 days'),
        ('SEED: Clase Individual Toby',   'INDIVIDUAL_CLASS',     1, 'Centro',             1, NOW() + INTERVAL '15 days'),
        ('SEED: Clase Individual Kira',   'INDIVIDUAL_CLASS',     1, 'Centro',             1, NOW() + INTERVAL '18 days'),
        ('SEED: Evento Solidario',        'EXTRA',               12, 'Plaza Mayor',        3, NOW() + INTERVAL '45 days'),
        ('SEED: Evento Pasado',           'SOCIALIZATION_GROUP',  8, 'Centro',             2, NOW() - INTERVAL '30 days')
    RETURNING id
),
numbered_activities AS (
    -- Map each freshly-inserted activity to a stable 0-based position
    -- by insertion order. reservations below join on this.
    SELECT id, (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM new_activities
)

-- ============================================================================
-- Section 2: Passes (10)
-- ============================================================================
-- One pass per seed user. The remaining_sessions column is set equal to
-- num_of_sessions (every pass starts fully available); the movements in
-- Section 4 are the audit log of what each reservation consumed, and the
-- register use case decrements remaining_sessions in memory on every new
-- booking. The pass_exhausted use case is covered by the unit tests; it
-- is not part of the seed because the seed positions would shift if a
-- second pass for one user were inserted, making the reservation join
-- brittle.
--
-- Position index (assigned by ROW_NUMBER OVER id ASC):
--   0: Carlos  (num=10, GENERICO,  no expiry)
--   1: María   (num=10, GENERICO,  no expiry)
--   2: Javier  (num=10, GENERICO,  no expiry)
--   3: Laura   (num=5,  ESPECIFICO, no expiry)
--   4: Diego   (num=10, GENERICO,  +1 year)
--   5: Ana     (num=10, GENERICO,  no expiry)
--   6: Pedro   (num=10, GENERICO,  no expiry)
--   7: Sara    (num=5,  ESPECIFICO, no expiry)
--   8: Miguel  (num=10, GENERICO,  no expiry) — owner is INACTIVE
--   9: Elena   (num=10, GENERICO,  no expiry)
, new_passes AS (
    INSERT INTO passes (user_id, num_of_sessions, remaining_sessions, price, pass_type, expires_at)
    SELECT
        su.id,
        p.num_of_sessions,
        p.remaining_sessions,
        p.price,
        p.pass_type::pass_type,
        p.expires_at
    FROM (VALUES
        (0,  10, 10, 12000, 'GENERICO',  NULL::TIMESTAMPTZ),
        (1,  10, 10, 12000, 'GENERICO',  NULL),
        (2,  10, 10, 12000, 'GENERICO',  NULL),
        (3,   5,  5,  8000, 'ESPECIFICO', NULL),
        (4,  10, 10, 12000, 'GENERICO',  NOW() + INTERVAL '1 year'),
        (5,  10, 10, 12000, 'GENERICO',  NULL),
        (6,  10, 10, 12000, 'GENERICO',  NULL),
        (7,   5,  5,  8000, 'ESPECIFICO', NULL),
        (8,  10, 10, 12000, 'GENERICO',  NULL),
        (9,  10, 10, 12000, 'GENERICO',  NULL)
    ) AS p(owner_position, num_of_sessions, remaining_sessions, price, pass_type, expires_at)
    JOIN seed_users su ON su.position = p.owner_position
    RETURNING id
),
numbered_passes AS (
    SELECT id, (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM new_passes
)

-- ============================================================================
-- Section 3: Reservations (15)
-- ============================================================================
-- 6 status values covered. The position columns below join the four
-- CTEs above (activities, passes, dogs) and resolve to real ids in a
-- single INSERT. No application code is involved.
--
-- Columns per row:
--   activity_pos → numbered_activities.position
--   owner_pos    → seed_users.position
--   dog_name     → seed_dogs.name (unique within an owner)
--   pass_pos     → numbered_passes.position
--   status       → 'CONFIRMED' | 'CANCELLED_IN_TIME' | 'CANCELLED_LATE'
--                | 'FORGIVEN' | 'COMPLETED' | 'NO_SHOW'
, new_reservations AS (
    INSERT INTO reservations (activity_id, dog_id, pass_id, status, created_at)
    SELECT na.id, sd.id, np.id, r.status::reservation_status,
           NOW() - ((15 - r.position)::int * INTERVAL '1 hour')
    FROM (VALUES
        -- (position, activity_pos, owner_pos, dog_name, pass_pos, status)
        -- 1-2: Carlos (0) — same pass, two CONFIRMED bookings into Paseo Social Matutino
        ( 1,  0,  0, 'Rex',   0, 'CONFIRMED'),
        ( 2,  0,  0, 'Leia',  0, 'CONFIRMED'),
        -- 3-4: María (1) — Luna CONFIRMED + Max CANCELLED_IN_TIME (refund visible)
        ( 3,  4,  1, 'Luna',  1, 'CONFIRMED'),
        ( 4,  4,  1, 'Max',   1, 'CANCELLED_IN_TIME'),
        -- 5-6: Javier (2) — 2 CONFIRMED into Ruta Corta + Clase Individual Luna
        --     (Clase Individual Luna has max_capacity=1 so this is the "full" scenario)
        ( 5,  7,  2, 'Rocky', 2, 'CONFIRMED'),
        ( 6,  8,  2, 'Toby',  2, 'CONFIRMED'),
        -- 7-8: Laura (3) — Thor CANCELLED_IN_TIME (refund) + Kira CONFIRMED
        ( 7,  9,  3, 'Thor',  3, 'CANCELLED_IN_TIME'),
        ( 8,  2,  3, 'Kira',  3, 'CONFIRMED'),
        -- 9: Diego (4) — Bruno CONFIRMED into Ruta Montaña
        ( 9,  5,  4, 'Bruno', 4, 'CONFIRMED'),
        -- 10-12: Ana (5) — 3 reservations: 1 CONFIRMED, 1 COMPLETED (past), 1 CANCELLED_LATE
        (10,  3,  5, 'Nala',  5, 'CONFIRMED'),
        (11, 13,  5, 'Simba', 5, 'COMPLETED'),
        (12,  1,  5, 'Bimba', 5, 'CANCELLED_LATE'),
        -- 13: Pedro (6) — Koko FORGIVEN into the past activity (admin override)
        (13, 13,  6, 'Koko',  6, 'FORGIVEN'),
        -- 14: Pedro (6) — Rambo NO_SHOW into Ruta Playa
        (14,  6,  6, 'Rambo', 6, 'NO_SHOW'),
        -- 15: Sara (7) — Laika CONFIRMED into Ruta Playa
        (15,  6,  7, 'Laika', 7, 'CONFIRMED')
    ) AS r(position, activity_pos, owner_pos, dog_name, pass_pos, status)
    JOIN numbered_activities na ON na.position = r.activity_pos
    JOIN seed_users su         ON su.position  = r.owner_pos
    JOIN seed_dogs sd          ON sd.name       = r.dog_name AND sd.user_id = su.id
    JOIN numbered_passes np    ON np.position   = r.pass_pos
    RETURNING id, activity_id, dog_id, pass_id, status, created_at
)

-- ============================================================================
-- Section 4: Pass movements (18, derived from the reservations above)
-- ============================================================================
-- Every reservation that consumed a session produces a -1 movement.
-- Every reservation in CANCELLED_IN_TIME or FORGIVEN status also produces
-- a +1 movement (the refund). The reason string mirrors what the
-- application would write so the audit log looks identical to a live
-- one. created_at is offset by 1 microsecond from the reservation's
-- created_at to guarantee chronological order in the audit log.
, reservation_movements AS (
    -- 1) -1 for every reservation that consumed a session.
    --    Includes CONFIRMED, CANCELLED_IN_TIME, CANCELLED_LATE, FORGIVEN,
    --    COMPLETED, NO_SHOW (every status that implies a consume).
    SELECT pass_id, -1 AS amount,
           'Reservation: activity ' || activity_id || ', dog ' || dog_id AS reason,
           created_at
    FROM new_reservations

    UNION ALL

    -- 2) +1 refund for every CANCELLED_IN_TIME and FORGIVEN reservation.
    --    CANCELLED_LATE is intentionally excluded (the seed documents the
    --    "no refund on late cancel" policy).
    SELECT pass_id, 1 AS amount,
           'Reservation ' || id || ' cancelled in time' AS reason,
           created_at + INTERVAL '1 microsecond'
    FROM new_reservations
    WHERE status IN ('CANCELLED_IN_TIME', 'FORGIVEN')
)
INSERT INTO pass_movements (pass_id, amount, reason, created_at)
SELECT pass_id, amount, reason, created_at
FROM reservation_movements
ORDER BY pass_id, created_at;

-- ============================================================================
-- Section 5: Refresh planner statistics
-- ============================================================================
ANALYZE activities;
ANALYZE passes;
ANALYZE pass_movements;
ANALYZE reservations;

COMMIT;
