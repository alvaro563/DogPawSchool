-- Optional size target for SOCIALIZATION_GROUP and ROUTE activities.
-- NULL = "all sizes" (no filter at reservation time). Only MINI,
-- MEDIUM or LARGE are accepted as targets; UNKNOWN is intentionally
-- not a valid target — it represents a dog with an unknown weight
-- which should never book into a size-restricted class.
ALTER TABLE activities
    ADD COLUMN size_target TEXT;

COMMENT ON COLUMN activities.size_target IS
    'Optional target size for SOCIALIZATION_GROUP / ROUTE. NULL = all sizes.';

ALTER TABLE activities
    ADD CONSTRAINT activities_size_target_valid
    CHECK (size_target IS NULL OR size_target IN ('MINI', 'MEDIUM', 'LARGE'));

CREATE INDEX idx_activities_size_target ON activities (size_target);
