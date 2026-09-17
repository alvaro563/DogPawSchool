-- Reverses 000016. NOTE: if rows in terminal statuses have accumulated
-- duplicates for the same (activity_id, dog_id) between the up and the
-- down, the ADD CONSTRAINT will fail. The down is best-effort: in
-- development the typical workflow is to drop the database and let the
-- migrations seed fresh data; in production a manual audit-and-cleanup
-- of terminal duplicates is required before this down can succeed.

DROP INDEX IF EXISTS uniq_reservation_dog_active;

ALTER TABLE reservations
    ADD CONSTRAINT uniq_reservation_dog_per_activity UNIQUE (activity_id, dog_id);
