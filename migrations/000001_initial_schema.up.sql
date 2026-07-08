-- ============================================================================
-- DogPaw - Initial Schema (Migration 000001)
-- PostgreSQL 15+
-- ============================================================================
-- Design principles:
--   * BIGINT IDENTITY for every primary key (future-proof, 9.2e18 rows).
--   * Native ENUM types for type safety and storage efficiency.
--   * TIMESTAMPTZ for every timestamp (timezone-aware, never TIMESTAMP).
--   * NUMERIC for decimals, INTEGER for money (in cents).
--   * Generated columns for age/size brackets to allow indexed filters.
--   * ON DELETE CASCADE on every FK (clean user deletion semantics).
--   * Indexes aligned 1:1 with repository list filters.
--   * Trigger-set updated_at on mutable tables (impossible to forget).
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. ENUM types (created first; every table below references them)
-- ============================================================================

CREATE TYPE user_role AS ENUM (
    'ADMIN',
    'REGULAR'
);

CREATE TYPE dog_sex AS ENUM (
    'MALE',
    'FEMALE'
);

CREATE TYPE age_bracket AS ENUM (
    'CHILDREN',
    'TEENAGER',
    'SEMI_ADULT',
    'ADULT',
    'UNKNOWN'
);

CREATE TYPE size_bracket AS ENUM (
    'MINI',
    'MEDIUM',
    'LARGE',
    'UNKNOWN'
);

CREATE TYPE pass_type AS ENUM (
    'GENERICO',
    'ESPECIFICO'
);

CREATE TYPE activity_type AS ENUM (
    'SOCIALIZATION_GROUP',
    'ROUTE',
    'INDIVIDUAL_CLASS',
    'EXTRA'
);

CREATE TYPE incompatibility_level AS ENUM (
    'ABSOLUTA',
    'MEDIA',
    'BAJA'
);

CREATE TYPE reservation_status AS ENUM (
    'CONFIRMED',
    'COMPLETED',
    'CANCELLED_IN_TIME',
    'CANCELLED_LATE',
    'FORGIVEN',
    'NO_SHOW'
);

-- ============================================================================
-- 2. Helper: trigger function to auto-update updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 3. users
-- ============================================================================
CREATE TABLE users (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL,
    password    TEXT        NOT NULL,
    role        user_role   NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_name_not_empty  CHECK (LENGTH(TRIM(name))     > 0),
    CONSTRAINT users_email_not_empty CHECK (LENGTH(TRIM(email))    > 0),
    CONSTRAINT users_email_format    CHECK (email ~* '^[^@\s]+@[^@\s]+\.[^@\s]+$'),
    CONSTRAINT users_password_length CHECK (LENGTH(password) >= 60)
);

CREATE UNIQUE INDEX idx_users_email       ON users (email);
CREATE INDEX        idx_users_role_active ON users (role, is_active);

CREATE TRIGGER trg_users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 4. incompatibilities
-- ============================================================================
CREATE TABLE incompatibilities (
    id          BIGINT                 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT                   NOT NULL,
    level_type  incompatibility_level  NOT NULL,
    created_at  TIMESTAMPTZ            NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ            NOT NULL DEFAULT NOW(),

    CONSTRAINT incompatibilities_name_not_empty CHECK (LENGTH(TRIM(name)) > 0)
);

CREATE UNIQUE INDEX idx_incompatibilities_name  ON incompatibilities (LOWER(name));
CREATE INDEX        idx_incompatibilities_level ON incompatibilities (level_type);

CREATE TRIGGER trg_incompatibilities_set_updated_at
    BEFORE UPDATE ON incompatibilities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 5. dogs
-- ============================================================================
CREATE TABLE dogs (
    id              BIGINT        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT        NOT NULL,
    name            TEXT          NOT NULL,
    breed           TEXT          NOT NULL,
    age_in_months   INTEGER       NOT NULL DEFAULT 0,
    sex             dog_sex       NOT NULL,
    neutered        BOOLEAN       NOT NULL DEFAULT FALSE,
    heat            BOOLEAN       NOT NULL DEFAULT FALSE,
    weight_kg       NUMERIC(6,2)  NOT NULL DEFAULT 0,
    photo_url       TEXT,
    medical_notes   TEXT,
    educator_notes  TEXT,
    passport        TEXT          NOT NULL,
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    age_bracket     age_bracket   GENERATED ALWAYS AS (
        CASE
            WHEN age_in_months <  0   THEN 'UNKNOWN'::age_bracket
            WHEN age_in_months <= 6   THEN 'CHILDREN'::age_bracket
            WHEN age_in_months <= 18  THEN 'TEENAGER'::age_bracket
            WHEN age_in_months <= 36  THEN 'SEMI_ADULT'::age_bracket
            ELSE                           'ADULT'::age_bracket
        END
    ) STORED,
    size_bracket    size_bracket  GENERATED ALWAYS AS (
        CASE
            WHEN weight_kg <= 0   THEN 'UNKNOWN'::size_bracket
            WHEN weight_kg <= 5   THEN 'MINI'::size_bracket
            WHEN weight_kg <= 20  THEN 'MEDIUM'::size_bracket
            ELSE                       'LARGE'::size_bracket
        END
    ) STORED,

    CONSTRAINT dogs_user_id_positive     CHECK (user_id > 0),
    CONSTRAINT dogs_name_not_empty      CHECK (LENGTH(TRIM(name))    > 0),
    CONSTRAINT dogs_breed_not_empty     CHECK (LENGTH(TRIM(breed))   > 0),
    CONSTRAINT dogs_age_valid           CHECK (age_in_months >= 0),
    CONSTRAINT dogs_weight_kg_valid     CHECK (weight_kg >= 0),
    CONSTRAINT dogs_passport_not_empty  CHECK (LENGTH(TRIM(passport)) > 0),

    CONSTRAINT fk_dogs_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE UNIQUE INDEX idx_dogs_passport     ON dogs (passport);
CREATE INDEX        idx_dogs_user_id      ON dogs (user_id);
CREATE INDEX        idx_dogs_breed        ON dogs (breed);
CREATE INDEX        idx_dogs_sex          ON dogs (sex);
CREATE INDEX        idx_dogs_neutered     ON dogs (neutered);
CREATE INDEX        idx_dogs_heat         ON dogs (heat);
CREATE INDEX        idx_dogs_is_active    ON dogs (is_active);
CREATE INDEX        idx_dogs_age_bracket  ON dogs (age_bracket);
CREATE INDEX        idx_dogs_size_bracket ON dogs (size_bracket);
CREATE INDEX        idx_dogs_user_active  ON dogs (user_id, is_active);

CREATE TRIGGER trg_dogs_set_updated_at
    BEFORE UPDATE ON dogs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 6. dog_incompatibilities
-- ============================================================================
CREATE TABLE dog_incompatibilities (
    dog_id             BIGINT      NOT NULL,
    incompatibility_id BIGINT      NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (dog_id, incompatibility_id),

    CONSTRAINT fk_dog_incompat_dog
        FOREIGN KEY (dog_id)
        REFERENCES dogs (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT fk_dog_incompat_incompat
        FOREIGN KEY (incompatibility_id)
        REFERENCES incompatibilities (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE INDEX idx_dog_incompat_incompat_id
    ON dog_incompatibilities (incompatibility_id);

-- ============================================================================
-- 7. passes
-- ============================================================================
-- price is stored in cents (INTEGER). Example: 49.95€ -> 4995.
CREATE TABLE passes (
    id                  BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id             BIGINT      NOT NULL,
    num_of_sessions     INTEGER     NOT NULL,
    remaining_sessions  INTEGER     NOT NULL,
    price               INTEGER     NOT NULL,
    pass_type           pass_type   NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT passes_user_id_positive       CHECK (user_id > 0),
    CONSTRAINT passes_num_of_sessions_pos    CHECK (num_of_sessions > 0),
    CONSTRAINT passes_remaining_non_negative CHECK (remaining_sessions >= 0),
    CONSTRAINT passes_remaining_le_total     CHECK (remaining_sessions <= num_of_sessions),
    CONSTRAINT passes_price_non_negative     CHECK (price >= 0),
    CONSTRAINT passes_expiry_after_creation  CHECK (expires_at IS NULL OR expires_at >= created_at),

    CONSTRAINT fk_passes_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE INDEX idx_passes_user_id     ON passes (user_id);
CREATE INDEX idx_passes_user_expiry ON passes (user_id, expires_at);

CREATE TRIGGER trg_passes_set_updated_at
    BEFORE UPDATE ON passes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 8. pass_movements (append-only audit log)
-- ============================================================================
-- amount = -1 for consume, +1 for refund. Immutable, no updated_at.
CREATE TABLE pass_movements (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pass_id     BIGINT      NOT NULL,
    amount      INTEGER     NOT NULL,
    reason      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pass_movements_pass_id_positive  CHECK (pass_id > 0),
    CONSTRAINT pass_movements_amount_nonzero    CHECK (amount <> 0),
    CONSTRAINT pass_movements_reason_not_empty  CHECK (LENGTH(TRIM(reason)) > 0),

    CONSTRAINT fk_pass_movements_pass
        FOREIGN KEY (pass_id)
        REFERENCES passes (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE INDEX idx_pass_movements_pass_id ON pass_movements (pass_id);
CREATE INDEX idx_pass_movements_recent  ON pass_movements (pass_id, created_at DESC);

-- ============================================================================
-- 9. activities
-- ============================================================================
CREATE TABLE activities (
    id                BIGINT          GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name              TEXT            NOT NULL,
    activity_type     activity_type   NOT NULL,
    max_capacity      INTEGER         NOT NULL,
    location          TEXT            NOT NULL,
    duration_in_hours INTEGER         NOT NULL,
    date              TIMESTAMPTZ     NOT NULL,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT activities_name_not_empty        CHECK (LENGTH(TRIM(name))     > 0),
    CONSTRAINT activities_location_not_empty    CHECK (LENGTH(TRIM(location)) > 0),
    CONSTRAINT activities_max_capacity_positive CHECK (max_capacity > 0),
    CONSTRAINT activities_duration_positive     CHECK (duration_in_hours > 0)
);

CREATE INDEX idx_activities_date      ON activities (date);
CREATE INDEX idx_activities_type_date ON activities (activity_type, date);

CREATE TRIGGER trg_activities_set_updated_at
    BEFORE UPDATE ON activities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 10. reservations
-- ============================================================================
CREATE TABLE reservations (
    id           BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    activity_id  BIGINT             NOT NULL,
    dog_id       BIGINT             NOT NULL,
    pass_id      BIGINT             NOT NULL,
    status       reservation_status NOT NULL DEFAULT 'CONFIRMED',
    created_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW(),

    CONSTRAINT reservations_activity_id_positive CHECK (activity_id > 0),
    CONSTRAINT reservations_dog_id_positive      CHECK (dog_id > 0),
    CONSTRAINT reservations_pass_id_positive     CHECK (pass_id > 0),

    CONSTRAINT fk_reservations_activity
        FOREIGN KEY (activity_id)
        REFERENCES activities (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT fk_reservations_dog
        FOREIGN KEY (dog_id)
        REFERENCES dogs (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT fk_reservations_pass
        FOREIGN KEY (pass_id)
        REFERENCES passes (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT uniq_reservation_dog_per_activity UNIQUE (activity_id, dog_id)
);

CREATE INDEX idx_reservations_activity_id     ON reservations (activity_id);
CREATE INDEX idx_reservations_dog_id          ON reservations (dog_id);
CREATE INDEX idx_reservations_pass_id         ON reservations (pass_id);
CREATE INDEX idx_reservations_activity_status ON reservations (activity_id, status);

CREATE TRIGGER trg_reservations_set_updated_at
    BEFORE UPDATE ON reservations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- 11. Table & column comments (database documentation)
-- ============================================================================
COMMENT ON TABLE users                 IS 'Application users (dog owners and admins)';
COMMENT ON TABLE incompatibilities     IS 'Master list of incompatibility categories dogs may have';
COMMENT ON TABLE dogs                  IS 'Dogs enrolled in the school, owned by a user';
COMMENT ON TABLE dog_incompatibilities IS 'Many-to-many: which dogs have which incompatibilities';
COMMENT ON TABLE passes                IS 'Prepaid session packs owned by users; price in cents';
COMMENT ON TABLE pass_movements        IS 'Append-only audit log of session consume (-1) and refund (+1)';
COMMENT ON TABLE activities            IS 'Scheduled classes/routes/individual sessions';
COMMENT ON TABLE reservations          IS 'A dog booked into an activity, paid from a pass';

COMMENT ON COLUMN users.password       IS 'bcrypt hash, minimum 60 chars';
COMMENT ON COLUMN passes.price         IS 'Price in cents/centimos (e.g. 49.95€ = 4995)';
COMMENT ON COLUMN dogs.weight_kg       IS 'Weight in kilograms, NUMERIC(6,2) for precision';
COMMENT ON COLUMN dogs.age_bracket     IS 'GENERATED: derived from age_in_months using domain rules';
COMMENT ON COLUMN dogs.size_bracket    IS 'GENERATED: derived from weight_kg using domain rules';

-- ============================================================================
-- 12. Refresh planner statistics
-- ============================================================================
ANALYZE;

INSERT INTO users (name, email, password, role, is_active)
VALUES ('Demo Owner', 'demo@dogpaw.com', '$2a$12$V7bXmS7D6v9f9rG7gH8zOue1n2m3o4p5q6r7s8t9u0v1w2x3y4z56', 'REGULAR', true);

COMMIT;
