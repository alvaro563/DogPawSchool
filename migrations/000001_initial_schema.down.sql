-- ============================================================================
-- DogPaw - Initial Schema ROLLBACK (Migration 000001 DOWN)
-- ============================================================================
-- Drops every object created in 000001_initial_schema.up.sql in REVERSE
-- dependency order. Run via: migrate -path migrations -database $DATABASE_URL down 1
-- ============================================================================

BEGIN;

-- Tables: reverse topological order. CASCADE removes triggers + indexes
-- attached to each table automatically.
DROP TABLE IF EXISTS reservations           CASCADE;
DROP TABLE IF EXISTS pass_movements         CASCADE;
DROP TABLE IF EXISTS passes                 CASCADE;
DROP TABLE IF EXISTS activities             CASCADE;
DROP TABLE IF EXISTS dog_incompatibilities  CASCADE;
DROP TABLE IF EXISTS dogs                   CASCADE;
DROP TABLE IF EXISTS incompatibilities      CASCADE;
DROP TABLE IF EXISTS users                  CASCADE;

-- Trigger function (orphaned once all triggers are gone)
DROP FUNCTION IF EXISTS set_updated_at()    CASCADE;

-- Enum types (no inter-dependencies, order is irrelevant here)
DROP TYPE IF EXISTS reservation_status      CASCADE;
DROP TYPE IF EXISTS activity_type           CASCADE;
DROP TYPE IF EXISTS pass_type               CASCADE;
DROP TYPE IF EXISTS size_bracket            CASCADE;
DROP TYPE IF EXISTS age_bracket             CASCADE;
DROP TYPE IF EXISTS dog_sex                 CASCADE;
DROP TYPE IF EXISTS incompatibility_level   CASCADE;
DROP TYPE IF EXISTS user_role               CASCADE;

COMMIT;
