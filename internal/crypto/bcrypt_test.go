package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"dogpaw/internal/crypto"
)

// The auth use case asserts only that it persists whatever the hasher
// returns; the guarantees about the hash itself belong here.

func TestBcryptHasher_Hash(t *testing.T) {
	t.Parallel()

	// bcrypt.MinCost keeps the test fast: the output format is
	// identical at every cost, and the cost factor itself is a
	// deployment decision, not a property under test.
	hasher := crypto.NewBcryptHasher(bcrypt.MinCost)

	t.Run("produces a verifiable 60-char bcrypt hash", func(t *testing.T) {
		t.Parallel()
		hash, err := hasher.Hash("correct horse battery staple")

		require.NoError(t, err)
		assert.Len(t, hash, 60, "bcrypt hashes are 60 chars")
		assert.Regexp(t, `^\$2[aby]\$`, hash, "bcrypt hashes carry a $2x$ prefix")
		assert.NoError(t, hasher.Compare(hash, "correct horse battery staple"))
	})

	t.Run("never returns the plaintext", func(t *testing.T) {
		t.Parallel()
		hash, err := hasher.Hash("hunter2")

		require.NoError(t, err)
		assert.NotContains(t, hash, "hunter2")
	})

	t.Run("is salted: same input yields different hashes", func(t *testing.T) {
		t.Parallel()
		first, err := hasher.Hash("same-password")
		require.NoError(t, err)
		second, err := hasher.Hash("same-password")
		require.NoError(t, err)

		assert.NotEqual(t, first, second, "a per-hash salt must make the outputs differ")
		assert.NoError(t, hasher.Compare(first, "same-password"))
		assert.NoError(t, hasher.Compare(second, "same-password"))
	})

	t.Run("satisfies the users_password_length DB constraint", func(t *testing.T) {
		t.Parallel()
		// migrations/000001: CHECK (LENGTH(password) >= 60)
		hash, err := hasher.Hash("x")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(hash), 60)
	})
}

func TestBcryptHasher_Compare(t *testing.T) {
	t.Parallel()
	hasher := crypto.NewBcryptHasher(bcrypt.MinCost)
	hash, err := hasher.Hash("right")
	require.NoError(t, err)

	t.Run("rejects a wrong password", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, hasher.Compare(hash, "wrong"))
	})

	t.Run("rejects a malformed hash", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, hasher.Compare("not-a-bcrypt-hash", "right"))
	})
}

func TestNewBcryptHasher_ClampsInvalidCost(t *testing.T) {
	t.Parallel()

	// An out-of-range cost must degrade to the library default rather
	// than fail at hash time, so a bad config cannot break registration.
	cases := []struct {
		name string
		cost int
	}{
		{"zero", 0},
		{"negative", -1},
		{"below min", bcrypt.MinCost - 1},
		{"above max", bcrypt.MaxCost + 1},
	}
	for _, tc := range cases {
		name, cost := tc.name, tc.cost
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			hash, err := crypto.NewBcryptHasher(cost).Hash("pw")

			require.NoError(t, err)
			actualCost, err := bcrypt.Cost([]byte(hash))
			require.NoError(t, err)
			assert.Equal(t, bcrypt.DefaultCost, actualCost)
		})
	}
}
