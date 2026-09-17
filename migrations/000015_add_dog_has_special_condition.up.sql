ALTER TABLE dogs
    ADD COLUMN has_special_condition BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN dogs.has_special_condition IS
    'Marks dogs that need special attention (reactive, recovering, etc).';

CREATE INDEX idx_dogs_has_special_condition ON dogs (has_special_condition) WHERE has_special_condition = TRUE;
