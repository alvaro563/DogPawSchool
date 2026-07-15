-- ============================================================================
-- DogPaw - Migration 000003: Seed test data
-- ============================================================================
-- Inserts 10 users (1 ADMIN + 9 REGULAR) and 30 dogs (3 per user) for
-- development, manual QA, and integration testing.
--
-- Distribution is designed to exercise every code path:
--   * age_bracket: 4 CHILDREN / 10 TEENAGER / 8 SEMI_ADULT / 8 ADULT
--   * size_bracket: 6 MINI / 11 MEDIUM / 13 LARGE
--   * sex: 15 MALE / 15 FEMALE
--   * neutered: 15 true / 15 false
--   * heat: 5 females in heat
--   * is_active: 27 true / 3 false
--
-- Idempotency: this migration is NOT idempotent. users.email and dogs.passport
-- have UNIQUE constraints; running it twice will fail. This is expected for a
-- seed migration that should run exactly once after the schema is in place.
--
-- To start fresh on a dev database, drop and re-create the DB; do not rely on
-- this DOWN (it is intentionally empty; see 000003_seed_test_data.down.sql).
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Users (10)
-- ============================================================================
-- Position index (assigned by INSERT order, referenced by dogs.user_position):
--   0: Carlos Admin     (ADMIN,   active)
--   1: María López      (REGULAR, active)
--   2: Javier Ruiz      (REGULAR, active)
--   3: Laura García     (REGULAR, active)
--   4: Diego Fernández  (REGULAR, active)
--   5: Ana Martínez     (REGULAR, active)
--   6: Pedro Sánchez    (REGULAR, active)
--   7: Sara Gómez       (REGULAR, active)
--   8: Miguel Torres    (REGULAR, INACTIVE — exercises the is_active filter)
--   9: Elena Navarro    (REGULAR, active)
--
-- Passwords are 60-char bcrypt-format placeholders. The schema only checks
-- LENGTH(password) >= 60, not the algorithm. None of these accounts are
-- meant to authenticate; they are inert seed data.
WITH new_users AS (
    INSERT INTO users (name, email, password, role, is_active)
    VALUES
        ('Carlos Admin',    'admin@dogpaw.com',            '$2a$12$seeduser01passwordplaceholder000000000000000000000000', 'ADMIN',   true),
        ('María López',     'maria.lopez@example.com',     '$2a$12$seeduser02passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Javier Ruiz',     'javier.ruiz@example.com',     '$2a$12$seeduser03passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Laura García',    'laura.garcia@example.com',    '$2a$12$seeduser04passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Diego Fernández', 'diego.fernandez@example.com', '$2a$12$seeduser05passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Ana Martínez',    'ana.martinez@example.com',    '$2a$12$seeduser06passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Pedro Sánchez',   'pedro.sanchez@example.com',   '$2a$12$seeduser07passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Sara Gómez',      'sara.gomez@example.com',      '$2a$12$seeduser08passwordplaceholder000000000000000000000000', 'REGULAR', true),
        ('Miguel Torres',   'miguel.torres@example.com',   '$2a$12$seeduser09passwordplaceholder000000000000000000000000', 'REGULAR', false),
        ('Elena Navarro',   'elena.navarro@example.com',   '$2a$12$seeduser10passwordplaceholder000000000000000000000000', 'REGULAR', true)
    RETURNING id
),
numbered AS (
    -- Map each freshly-inserted user to a stable 0-based position by insertion
    -- order. dogs.user_position below joins on this so the seed stays readable
    -- as "user 0 is Carlos, user 1 is María, ...".
    SELECT id, (ROW_NUMBER() OVER (ORDER BY id ASC)) - 1 AS position
    FROM new_users
)

-- ============================================================================
-- 2. Dogs (30) — 3 per user
-- ============================================================================
-- user_position maps to the numbered CTE above.
-- Passports are prefixed ES-SEED- to distinguish from any manually-created
-- test data (existing dogs use ES-Luna1, ES-Luna2, ES-Toby).
INSERT INTO dogs (
    user_id, name, breed, age_in_months, sex, neutered, heat, weight_kg,
    passport, is_active, medical_notes, educator_notes
)
SELECT
    n.id,
    d.name,
    d.breed,
    d.age_in_months,
    d.sex::dog_sex,
    d.neutered,
    d.heat,
    d.weight_kg,
    d.passport,
    d.is_active,
    d.medical_notes,
    d.educator_notes
FROM (VALUES
    -- Carlos Admin (position 0): 3 dogs covering SEMI_ADULT+LARGE, TEENAGER+MEDIUM, CHILDREN+MINI
    (0, 'Rex',  'Dalmatian',           60, 'MALE',   true,  false, 33.0, 'ES-SEED-001', true,  NULL,                        'Sociable con otros perros grandes'),

    (0, 'Leia', 'Samoyed',             18, 'FEMALE', true,  true,  12.0, 'ES-SEED-002', true,  'Alergia leve a polen',      NULL),

    (0, 'Nano', 'Chihuahua',            4, 'MALE',   false, false,  1.8, 'ES-SEED-003', false, NULL,                        'Necesita socialización'),

    -- María López (position 1)
    (1, 'Luna',  'Labrador',           24, 'FEMALE', true,  false, 22.5, 'ES-SEED-004', true,  NULL,                        NULL),

    (1, 'Max',   'German Shepherd',    48, 'MALE',   true,  false, 32.0, 'ES-SEED-005', true,  'Displasia cadera leve',    'Muy obediente, listo para grupo avanzado'),

    (1, 'Coco',  'Toy Poodle',          6, 'FEMALE', false, false,  4.5, 'ES-SEED-006', true,  NULL,                        'Cachorra, primera sesión'),

    -- Javier Ruiz (position 2)
    (2, 'Rocky', 'Boxer',              36, 'MALE',   false, false, 28.0, 'ES-SEED-007', true,  NULL,                        'Reactivo a correas'),

    (2, 'Nina',  'Beagle',             12, 'FEMALE', false, true,   6.0, 'ES-SEED-008', true,  NULL,                        'En celo, separar de machos'),

    (2, 'Toby',  'Cocker Spaniel',     18, 'MALE',   true,  false,  9.0, 'ES-SEED-009', true,  'Otitis recurrente',        NULL),

    -- Laura García (position 3)
    (3, 'Kira',  'Border Collie',      60, 'FEMALE', true,  false, 18.0, 'ES-SEED-010', true,  NULL,                        'Alta energía, necesita ejercicio previo'),

    (3, 'Thor',  'Husky',              30, 'MALE',   false, false, 35.0, 'ES-SEED-011', true,  NULL,                        'Tira mucho de la correa'),

    (3, 'Maya',  'Chihuahua',           4, 'FEMALE', false, false,  2.5, 'ES-SEED-012', true,  NULL,                        'Cachorra, muy miedosa'),

    -- Diego Fernández (position 4)
    (4, 'Bruno', 'Rottweiler',         84, 'MALE',   true,  false, 40.0, 'ES-SEED-013', true,  'Control articular anual',  NULL),

    (4, 'Lola',  'French Bulldog',     18, 'FEMALE', false, true,  14.0, 'ES-SEED-014', true,  'Problemas respiratorios',  'Evitar ejercicio intenso en calor'),

    (4, 'Taco',  'Jack Russell',        8, 'MALE',   false, false,  7.0, 'ES-SEED-015', true,  NULL,                        'Muy activo'),

    -- Ana Martínez (position 5)
    (5, 'Nala',  'Golden Retriever',   36, 'FEMALE', true,  false, 25.0, 'ES-SEED-016', true,  NULL,                        NULL),

    (5, 'Simba', 'Doberman',           24, 'MALE',   true,  false, 30.0, 'ES-SEED-017', true,  NULL,                        'Necesita socialización con extraños'),

    (5, 'Bimba', 'Dachshund',          12, 'FEMALE', false, false,  5.5, 'ES-SEED-018', true,  'Problemas de espalda',      'Evitar saltos'),

    -- Pedro Sánchez (position 6)
    (6, 'Koko',  'Standard Poodle',    48, 'MALE',   true,  false, 22.0, 'ES-SEED-019', true,  NULL,                        NULL),

    (6, 'Luna2', 'Cocker Spaniel',     18, 'FEMALE', true,  true,  16.0, 'ES-SEED-020', true,  NULL,                        'En celo'),

    (6, 'Rambo', 'Bernese Mountain',   60, 'MALE',   true,  false, 38.0, 'ES-SEED-021', true,  'Chequeos cardíacos',       NULL),

    -- Sara Gómez (position 7)
    (7, 'Mia',   'Yorkshire Terrier',   6, 'FEMALE', false, false,  3.8, 'ES-SEED-022', true,  NULL,                        'Cachorra'),

    (7, 'Odin',  'Akita',              30, 'MALE',   false, false, 28.0, 'ES-SEED-023', true,  NULL,                        'Territorial con otros machos'),

    (7, 'Laika', 'Siberian Husky',     48, 'FEMALE', true,  false, 20.0, 'ES-SEED-024', true,  NULL,                        NULL),

    -- Miguel Torres (position 8): owner is INACTIVE
    (8, 'Zeus',  'Cane Corso',         96, 'MALE',   true,  false, 45.0, 'ES-SEED-025', true,  NULL,                        'Necesita bozal'),

    (8, 'Kira2', 'Shiba Inu',          24, 'FEMALE', false, true,   8.0, 'ES-SEED-026', true,  NULL,                        'En celo'),

    (8, 'Pipo',  'Pomeranian',         12, 'MALE',   false, false,  4.0, 'ES-SEED-027', true,  NULL,                        NULL),

    -- Elena Navarro (position 9): 2 inactive dogs
    (9, 'Roxy',  'Beagle',             12, 'FEMALE', false, false, 11.0, 'ES-SEED-028', false, NULL,                        'Perro de baja por enfermedad'),

    (9, 'Duke',  'Weimaraner',         36, 'MALE',   true,  false, 26.0, 'ES-SEED-029', true,  NULL,                        NULL),

    (9, 'Flora', 'Maltese',             8, 'FEMALE', false, false,  4.2, 'ES-SEED-030', false, NULL,                        'Baja temporal')
) AS d(
    user_position, name, breed, age_in_months, sex, neutered, heat,
    weight_kg, passport, is_active, medical_notes, educator_notes
)
JOIN numbered n ON n.position = d.user_position;

COMMIT;
