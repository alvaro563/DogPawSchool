package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// LoginAttemptRepository is the postgres-backed implementation of
// domain.AccountLoginLimiter. One row per attempt in login_attempts;
// rolling-window counts are done in SQL with an index hit on
// (email, attempted_at) or (remote_ip, attempted_at). The caller
// (LoginUseCase) drives the policy: thresholds and windows come
// from configuration.
//
// Concurrency: a single attempt is a single INSERT, so two
// parallel calls will each get their own row. The race between
// Check and Record inside ONE request is acceptable: a legitimate
// user pressing Enter twice within the same second can push the
// counter up by 2, but the next Check still rejects because the
// attempt already counted. The check / record sequence in the
// use case is not strictly atomic; we accept the small window of
// "one extra attempt above threshold" in exchange for not
// holding a serialisable transaction across a bcrypt verify
// (which costs ~250ms and would serialise the entire endpoint).
type LoginAttemptRepository struct {
	db                *sql.DB
	emailMaxFailures  int
	emailWindow       time.Duration
	ipMaxFailures     int
	ipWindow          time.Duration
}

// NewLoginAttemptRepository constructs the limiter. Thresholds
// must be > 0; windows must be > 0. Caller (router) supplies
// these from configuration so they are tunable per environment.
func NewLoginAttemptRepository(
	db *sql.DB,
	emailMaxFailures int,
	emailWindow time.Duration,
	ipMaxFailures int,
	ipWindow time.Duration,
) *LoginAttemptRepository {
	return &LoginAttemptRepository{
		db:               db,
		emailMaxFailures: emailMaxFailures,
		emailWindow:      emailWindow,
		ipMaxFailures:    ipMaxFailures,
		ipWindow:         ipWindow,
	}
}

// Check returns domain.ErrAccountLockout if the email OR the IP
// has exceeded its failure threshold inside its rolling window.
// The RetryAfter on the returned error is the time until the
// OLDEST counted failure ages out of the window — the soonest the
// caller could legitimately retry. Order of evaluation: email
// first (the more specific limit), then IP (the volumetric
// fallback). If both trigger, the email axis wins because it
// carries the more actionable diagnosis.
func (r *LoginAttemptRepository) Check(ctx context.Context, email, remoteIP string) error {
	now := time.Now().UTC()

	emailCount, emailOldest, err := r.countFailuresSince(ctx,
		"email = $2", []any{email}, now.Add(-r.emailWindow))
	if err != nil {
		return fmt.Errorf("check email lockout: %w", err)
	}
	if emailCount >= r.emailMaxFailures {
		retry := time.Until(emailOldest.Add(r.emailWindow))
		if retry < 0 {
			retry = 0
		}
		return &domain.ErrAccountLockout{
			RetryAfter: retry,
			Reason:     domain.LockoutReasonEmail,
		}
	}

	ipCount, ipOldest, err := r.countFailuresSince(ctx,
		"remote_ip = $2", []any{remoteIP}, now.Add(-r.ipWindow))
	if err != nil {
		return fmt.Errorf("check ip lockout: %w", err)
	}
	if ipCount >= r.ipMaxFailures {
		retry := time.Until(ipOldest.Add(r.ipWindow))
		if retry < 0 {
			retry = 0
		}
		return &domain.ErrAccountLockout{
			RetryAfter: retry,
			Reason:     domain.LockoutReasonIP,
		}
	}

	return nil
}

// countFailuresSince returns (count, oldestFailureTimestamp, err)
// for failures matching the predicate inside the window. The
// MIN(attempted_at) query uses the (col, time DESC) index for a
// index-only seek + the rows after the predicate filter.
//
// Parameter numbering: $1 is always the since timestamp (cast to
// timestamptz to defeat the planner's text-type fallback), $2 is
// the predicate value (email or remote_ip). This split avoids the
// "one parameter, two types" error that pgx returns when the
// same $N is used with conflicting casts in one statement.
func (r *LoginAttemptRepository) countFailuresSince(
	ctx context.Context,
	predicate string,
	args []any,
	since time.Time,
) (int, time.Time, error) {
	const query = `
		SELECT COUNT(*), COALESCE(MIN(attempted_at), now())
		FROM login_attempts
		WHERE success = false
		  AND attempted_at >= $1::timestamptz
		  AND ` // predicate appended by caller
	fullArgs := append([]any{since}, args...)

	var count int
	var oldest time.Time
	if err := runner(ctx, r.db).QueryRowContext(ctx, query+predicate, fullArgs...).
		Scan(&count, &oldest); err != nil {
		return 0, time.Time{}, err
	}
	return count, oldest, nil
}

// Record appends a row to login_attempts. Called AFTER every
// login attempt (success OR failure) so the counter reflects
// reality, not just the bad-faith attempts.
func (r *LoginAttemptRepository) Record(ctx context.Context, email, remoteIP string, success bool) error {
	const query = `
		INSERT INTO login_attempts (email, remote_ip, success)
		VALUES ($1, $2, $3)
	`
	if _, err := runner(ctx, r.db).ExecContext(ctx, query, email, remoteIP, success); err != nil {
		return fmt.Errorf("record login attempt: %w", err)
	}
	return nil
}

// ResetByEmail deletes the failure history for this email so the
// counter starts fresh after a successful login. The IP history
// is preserved — a successful login by a legitimate user does
// not exonerate other accounts being attacked from the same IP.
// Intended for use only on successful authentication.
func (r *LoginAttemptRepository) ResetByEmail(ctx context.Context, email string) error {
	const query = `
		DELETE FROM login_attempts
		WHERE email = $1
		  AND success = false
	`
	if _, err := runner(ctx, r.db).ExecContext(ctx, query, email); err != nil {
		return fmt.Errorf("reset login attempts by email: %w", err)
	}
	return nil
}

// DeleteOlderThan removes rows whose attempted_at is before the
// cutoff. Called by the background cleanup goroutine started in
// server.go (every 15 min, 1h cutoff). One DELETE per cycle; the
// index on attempted_at makes it a sequential tail cut.
func (r *LoginAttemptRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	const query = `DELETE FROM login_attempts WHERE attempted_at < $1::timestamptz`
	res, err := runner(ctx, r.db).ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old login attempts: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return rows, nil
}

// Compile-time assertion that *LoginAttemptRepository satisfies
// the domain contract. Catches drift if the interface is
// extended.
var _ domain.AccountLoginLimiter = (*LoginAttemptRepository)(nil)

// silence unused import warning if sql package only referenced via *sql.DB
var _ = sql.ErrNoRows
