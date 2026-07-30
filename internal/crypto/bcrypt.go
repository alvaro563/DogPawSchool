// Package crypto holds the concrete password-hashing implementations
// used by the application layer. It is infrastructure: the use cases
// depend on the auth.PasswordHasher interface, and only the composition
// root (cmd/api) knows this package exists.
package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher hashes passwords with bcrypt at the configured cost.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a hasher at the given cost. A cost outside
// bcrypt's accepted range falls back to bcrypt.DefaultCost rather than
// failing at hash time, so a misconfiguration degrades to the library
// default instead of breaking registration.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// NewDefaultBcryptHasher returns a hasher at bcrypt.DefaultCost.
func NewDefaultBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

// Hash implements auth.PasswordHasher.
func (hasher *BcryptHasher) Hash(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), hasher.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hashed), nil
}

// Compare reports whether plain matches the stored hash. Returns nil on
// a match, a non-nil error otherwise. Kept here alongside Hash so both
// halves of the algorithm choice live in one place.
func (hasher *BcryptHasher) Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return fmt.Errorf("bcrypt compare: %w", err)
	}
	return nil
}

// Verify implements auth.PasswordVerifier by delegating to Compare.
// The two methods exist so that BcryptHasher satisfies both the
// PasswordVerifier interface (Verify) and the existing Compare API
// used by internal/crypto tests.
func (hasher *BcryptHasher) Verify(hash, plain string) error {
	return hasher.Compare(hash, plain)
}
