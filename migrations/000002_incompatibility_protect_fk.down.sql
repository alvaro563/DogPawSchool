BEGIN;

ALTER TABLE dog_incompatibilities
    DROP CONSTRAINT fk_dog_incompat_incompat;

ALTER TABLE dog_incompatibilities
    ADD CONSTRAINT fk_dog_incompat_incompat
        FOREIGN KEY (incompatibility_id)
        REFERENCES incompatibilities (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;

COMMIT;
