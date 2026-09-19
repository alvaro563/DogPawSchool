ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_individual_requires_dog;
DROP INDEX IF EXISTS idx_activities_dog_id;
ALTER TABLE activities DROP COLUMN IF EXISTS dog_id;
