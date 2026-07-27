DROP INDEX IF EXISTS idx_activities_closed;

ALTER TABLE activities DROP COLUMN closed;
