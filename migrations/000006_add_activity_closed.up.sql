ALTER TABLE activities ADD COLUMN closed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_activities_closed ON activities (closed);
