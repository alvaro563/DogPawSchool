package auth

// PasswordHasher turns a plaintext password into a storage-safe hash.
// The use case depends on this contract, never on a concrete algorithm:
// bcrypt, argon2 and scrypt are interchangeable infrastructure details,
// and keeping the choice out of this layer is what lets the use case be
// tested without paying the cost of a real key-derivation function.
//
// Implemented by crypto.BcryptHasher (internal/crypto).
type PasswordHasher interface {
	Hash(plain string) (string, error)
}

// PasswordVerifier checks a plaintext password against a previously
// stored hash. Separated from PasswordHasher because RegisterWith-
// InvitationUseCase only needs to hash, not verify; adding a Verify
// method to PasswordHasher would force every stub and fake to implement
// it for no reason (Interface Segregation Principle).
type PasswordVerifier interface {
	Verify(hash, plain string) error
}
