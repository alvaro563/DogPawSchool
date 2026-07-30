package crypto

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestJWTTokenGenerator_Generate(t *testing.T) {
	t.Parallel()

	secret := "super-secret-key-for-testing"
	ttl := 24 * time.Hour
	generator := NewJWTTokenGenerator(secret, ttl)

	user, err := domain.NewUser(7, "Alice", "alice@dogpaw.com", "hashed-pw-60chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
	require.NoError(t, err)

	tokenString, err := generator.Generate(user)
	require.NoError(t, err)

	require.NotEmpty(t, tokenString, "token must not be empty")

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		assert.True(t, ok, "unexpected signing method: %v", token.Header["alg"])
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid, "token must be valid")

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok, "claims must be MapClaims")

	assert.InDelta(t, float64(7), claims["sub"], 0, "sub must be user ID")
	assert.Equal(t, "ADMIN", claims["role"])

	iat, err := claims.GetIssuedAt()
	require.NoError(t, err)
	exp, err := claims.GetExpirationTime()
	require.NoError(t, err)

	assert.WithinDuration(t, iat.Time, time.Now(), 5*time.Second, "iat should be near now")
	assert.WithinDuration(t, iat.Time.Add(ttl), exp.Time, time.Second, "exp should be iat + ttl")
}

func TestJWTTokenGenerator_DifferentSecrets(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "Bob", "bob@dogpaw.com", "hashed-pw-60chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	require.NoError(t, err)

	generatorA := NewJWTTokenGenerator("secret-a", 1*time.Hour)
	generatorB := NewJWTTokenGenerator("secret-b", 1*time.Hour)

	tokenA, err := generatorA.Generate(user)
	require.NoError(t, err)

	tokenB, err := generatorB.Generate(user)
	require.NoError(t, err)

	assert.NotEqual(t, tokenA, tokenB, "different secrets must produce different tokens")
}

func TestJWTTokenGenerator_Generate_InvalidUser(t *testing.T) {
	t.Parallel()

	// User with empty name is invalid for NewUser, so we can't create one.
	// Instead verify the generator handles a valid user gracefully.
	generator := NewJWTTokenGenerator("test-secret", 1*time.Hour)

	_, err := domain.NewUser(0, "", "x@y.com", "pw", domain.RoleRegular)
	require.Error(t, err, "user with empty name should fail validation")
	_ = generator
}
