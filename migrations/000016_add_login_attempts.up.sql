BEGIN;

CREATE TABLE login_attempts (
    id           BIGSERIAL PRIMARY KEY,
    email        TEXT        NOT NULL,
    remote_ip    TEXT        NOT NULL,
    success      BOOLEAN     NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookup: how many failed attempts against this email in the
-- last N minutes? The DESC ordering matches the query plan:
-- WHERE email=$1 AND success=false AND attempted_at >= $2
CREATE INDEX idx_login_attempts_email_time
    ON login_attempts (email, attempted_at DESC);

-- Same lookup but keyed by source IP. Used as a second line of
-- defense: a single IP hammering the login endpoint with
-- rotating emails is still detectable.
CREATE INDEX idx_login_attempts_ip_time
    ON login_attempts (remote_ip, attempted_at DESC);

-- Cleanup job (every 15 min) deletes rows older than 1 hour.
-- The index makes that scan a sequential tail-cut.
CREATE INDEX idx_login_attempts_attempted_at
    ON login_attempts (attempted_at);

COMMENT ON TABLE  login_attempts
    IS 'Auth attempt audit. Powers the account lockout (per email) and the IP volumetric defense (per remote_ip). One row per attempt; cleaned up by background job after 1 hour.';
COMMENT ON COLUMN login_attempts.email
    IS 'Plaintext email as submitted. The lockout is keyed on the IDENTIFIER, so hashing defeats the lookup. Email is already public (login form), so no extra disclosure.';
COMMENT ON COLUMN login_attempts.remote_ip
    IS 'TCP peer (RemoteAddr). Behind a proxy this is the proxy IP. The upstream proxy/WAF is responsible for per-real-client limits.';
COMMENT ON COLUMN login_attempts.success
    IS 'False = failed login (bcrypt mismatch, unknown email, inactive user). True = successful authentication.';

COMMIT;
