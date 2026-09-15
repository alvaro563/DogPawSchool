DROP INDEX IF EXISTS idx_activities_size_target;
ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_size_target_valid;
ALTER TABLE activities DROP COLUMN IF EXISTS size_target;
