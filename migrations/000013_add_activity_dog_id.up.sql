-- Target dog for INDIVIDUAL_CLASS activities. NULL for group
-- classes (SOCIALIZATION_GROUP, ROUTE) so existing rows and
-- group-class registrations are unaffected.
ALTER TABLE activities
    ADD COLUMN dog_id BIGINT REFERENCES dogs (id) ON DELETE SET NULL;

COMMENT ON COLUMN activities.dog_id
    IS 'Target dog for INDIVIDUAL_CLASS; NULL for group classes';

CREATE INDEX idx_activities_dog_id ON activities (dog_id);

-- Backfill: seed rows created before this migration assigned an
-- INDIVIDUAL_CLASS without a dog_id. Look up the dog by the same
-- passport that the seed used; activities whose seed dog no longer
-- exists are left NULL. (In practice the seed is deterministic, but
-- the WHERE NOT NULL guard makes the migration safe on production
-- DBs with arbitrary existing data.)
UPDATE activities SET dog_id = (SELECT id FROM dogs WHERE passport = 'ES-SEED-004' LIMIT 1)
 WHERE name = 'SEED: Clase Individual Luna' AND activity_type = 'INDIVIDUAL_CLASS' AND dog_id IS NULL;
UPDATE activities SET dog_id = (SELECT id FROM dogs WHERE passport = 'ES-SEED-005' LIMIT 1)
 WHERE name = 'SEED: Clase Individual Max'  AND activity_type = 'INDIVIDUAL_CLASS' AND dog_id IS NULL;
UPDATE activities SET dog_id = (SELECT id FROM dogs WHERE passport = 'ES-SEED-009' LIMIT 1)
 WHERE name = 'SEED: Clase Individual Toby' AND activity_type = 'INDIVIDUAL_CLASS' AND dog_id IS NULL;
UPDATE activities SET dog_id = (SELECT id FROM dogs WHERE passport = 'ES-SEED-010' LIMIT 1)
 WHERE name = 'SEED: Clase Individual Kira' AND activity_type = 'INDIVIDUAL_CLASS' AND dog_id IS NULL;

-- Defence-in-depth: enforce at the DB level that INDIVIDUAL_CLASS
-- always has a dog. Application code validates first, but the
-- CHECK guarantees no row can ever violate this invariant, even if
-- a future migration / script tries to insert directly.
ALTER TABLE activities
    ADD CONSTRAINT activities_individual_requires_dog
    CHECK (
        (activity_type = 'INDIVIDUAL_CLASS' AND dog_id IS NOT NULL)
        OR (activity_type <> 'INDIVIDUAL_CLASS')
    );
