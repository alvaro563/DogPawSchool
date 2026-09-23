package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

// cleanLoginAttempts truncates the login_attempts table for test
// isolation. Mirrors the helper used by the integration suites
// (TRUNCATE ... RESTART IDENTITY CASCADE) but only the one table
// so the lockout suite stays focused.
func cleanLoginAttempts(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`TRUNCATE TABLE login_attempts RESTART IDENTITY`)
	require.NoError(t, err)
}

func TestLoginAttemptRepository_RecordAndCount(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanLoginAttempts(t, testDB)

	repo := NewLoginAttemptRepository(testDB, 3, 15*time.Minute, 10, 5*time.Minute)
	ctx := context.Background()

	// Threshold semantics: count >= max → locked. So with
	// max=3, the 3rd failure triggers the lockout. Verify the
	// 2-failure state is still allowed.
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	require.NoError(t, repo.Check(ctx, "alice@dogpaw.com", "1.2.3.4"),
		"2 failures within window must NOT lock (threshold is 3)")

	// 3rd failure crosses the threshold → lockout.
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	err := repo.Check(ctx, "alice@dogpaw.com", "1.2.3.4")
	require.Error(t, err, "3rd failure must trigger email lockout")
	var lockoutErr *domain.ErrAccountLockout
	require.True(t, errors.As(err, &lockoutErr))
	assert.Equal(t, domain.LockoutReasonEmail, lockoutErr.Reason)
}

func TestLoginAttemptRepository_PerIPIsolation(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanLoginAttempts(t, testDB)

	// Tight IP limit: 2 failures per 5 minutes.
	repo := NewLoginAttemptRepository(testDB, 100, 15*time.Minute, 2, 5*time.Minute)
	ctx := context.Background()

	// Two different emails from the same IP: both count toward
	// the IP bucket.
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	require.NoError(t, repo.Record(ctx, "bob@dogpaw.com", "1.2.3.4", false))

	// A request for a NEW email from this IP should still be
	// blocked because the IP bucket is exhausted.
	err := repo.Check(ctx, "carol@dogpaw.com", "1.2.3.4")
	require.Error(t, err, "IP bucket lockout must apply even to fresh emails")

	// A request from a different IP for the same email must
	// still be allowed (email bucket independent of IP).
	require.NoError(t, repo.Check(ctx, "alice@dogpaw.com", "9.9.9.9"))
}

func TestLoginAttemptRepository_ResetByEmail(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanLoginAttempts(t, testDB)

	repo := NewLoginAttemptRepository(testDB, 2, 15*time.Minute, 100, 5*time.Minute)
	ctx := context.Background()

	// Two failures: at threshold.
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	require.NoError(t, repo.Record(ctx, "alice@dogpaw.com", "1.2.3.4", false))
	require.Error(t, repo.Check(ctx, "alice@dogpaw.com", "1.2.3.4"))

	// Reset wipes the email history.
	require.NoError(t, repo.ResetByEmail(ctx, "alice@dogpaw.com"))
	require.NoError(t, repo.Check(ctx, "alice@dogpaw.com", "1.2.3.4"),
		"after reset, the email bucket must be empty")
}

func TestLoginAttemptRepository_DeleteOlderThan(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanLoginAttempts(t, testDB)

	repo := NewLoginAttemptRepository(testDB, 3, 15*time.Minute, 100, 5*time.Minute)
	ctx := context.Background()

	// Insert one old row by hand.
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO login_attempts (email, remote_ip, success, attempted_at)
		 VALUES ($1, $2, $3, now() - interval '2 hours')`,
		"old@dogpaw.com", "1.2.3.4", false)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx,
		`INSERT INTO login_attempts (email, remote_ip, success, attempted_at)
		 VALUES ($1, $2, $3, now())`,
		"new@dogpaw.com", "1.2.3.4", false)
	require.NoError(t, err)

	deleted, err := repo.DeleteOlderThan(ctx, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted, "must delete only the row older than 1h")

	var remaining int
	require.NoError(t, testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM login_attempts`).Scan(&remaining))
	assert.Equal(t, 1, remaining)
}
