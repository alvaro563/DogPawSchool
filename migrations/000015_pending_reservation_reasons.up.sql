BEGIN;

CREATE TABLE reservation_pending_reasons (
    reservation_id BIGINT     NOT NULL,
    ordinal        SMALLINT   NOT NULL,
    reason_code    TEXT       NOT NULL,
    dog_ids        BIGINT[]   NOT NULL,

    PRIMARY KEY (reservation_id, ordinal),
    CONSTRAINT fk_rpr_reservation
        FOREIGN KEY (reservation_id)
        REFERENCES reservations (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT rpr_ordinal_nonneg
        CHECK (ordinal >= 0),
    CONSTRAINT rpr_reason_code_nonempty
        CHECK (length(reason_code) > 0)
);

CREATE INDEX idx_rpr_reservation
    ON reservation_pending_reasons (reservation_id);

COMMENT ON TABLE  reservation_pending_reasons
    IS 'Audit trail of why each PENDING_TO_CONFIRM reservation was held. One row per reason; ordinal preserves evaluation order (sex/neutered before special-condition).';
COMMENT ON COLUMN reservation_pending_reasons.reason_code
    IS 'Stable, language-neutral identifier (e.g. "sex_neutered:intact_vs_castrated", "has_special_condition"). Translated to Spanish by the handler.';
COMMENT ON COLUMN reservation_pending_reasons.dog_ids
    IS 'Dog IDs that triggered this reason (incoming candidate + existing conflict pair).';
COMMENT ON COLUMN reservation_pending_reasons.ordinal
    IS '0-based position in the original pending_reasons slice.';

COMMIT;
