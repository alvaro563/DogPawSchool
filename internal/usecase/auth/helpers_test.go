package auth

// stubHasher is a deterministic PasswordHasher for use case tests. It
// prefixes the plaintext instead of running a real key-derivation
// function, which keeps these tests fast and independent of the bcrypt
// cost factor. The bcrypt output format itself is covered by
// internal/crypto's own tests.
type stubHasher struct {
	hash func(plain string) (string, error)
}

func (s *stubHasher) Hash(plain string) (string, error) {
	if s.hash != nil {
		return s.hash(plain)
	}
	return "hashed:" + plain, nil
}
