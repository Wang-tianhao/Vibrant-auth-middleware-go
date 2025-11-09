package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// BenchmarkAlgorithmRouting measures the overhead of algorithm routing (target <10μs per SC-004)
func BenchmarkAlgorithmRouting(b *testing.B) {
	// Setup
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, _ := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)

	// Create test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Measure just the algorithm routing (map lookup)
		alg := token.Header["alg"].(string)
		_, _ = cfg.getValidator(alg)
	}
}

// BenchmarkHS256Validation measures full HS256 token validation with dual-algorithm config
func BenchmarkHS256Validation(b *testing.B) {
	// Setup
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, _ := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)

	// Create HS256 token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parseAndValidateJWT(tokenString, cfg)
	}
}

// BenchmarkRS256Validation measures full RS256 token validation with dual-algorithm config
func BenchmarkRS256Validation(b *testing.B) {
	// Setup
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, _ := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)

	// Create RS256 token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(rs256PrivateKey)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parseAndValidateJWT(tokenString, cfg)
	}
}

// BenchmarkSingleAlgorithmConfig ensures no regression vs existing single-algorithm performance
func BenchmarkSingleAlgorithmConfig(b *testing.B) {
	// Setup
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	// Create token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parseAndValidateJWT(tokenString, cfg)
	}
}

// BenchmarkDualAlgorithmConfigCreation measures the overhead of creating a dual-algorithm config
func BenchmarkDualAlgorithmConfigCreation(b *testing.B) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = NewConfig(
			WithHS256(hs256Secret),
			WithRS256(rs256PublicKey),
		)
	}
}

// BenchmarkAvailableAlgorithms measures the performance of listing available algorithms
func BenchmarkAvailableAlgorithms(b *testing.B) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, _ := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cfg.AvailableAlgorithms()
	}
}

// BenchmarkUnsupportedAlgorithmError measures the performance of unsupported algorithm handling
func BenchmarkUnsupportedAlgorithmError(b *testing.B) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	// Create RS256 token (unsupported)
	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(rs256PrivateKey)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parseAndValidateJWT(tokenString, cfg)
	}
}

// BenchmarkValidationErrorCreation measures the overhead of creating validation errors
func BenchmarkValidationErrorCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewValidationError(ErrUnsupportedAlgorithm, "algorithm ES256 not supported (available: HS256, RS256)", nil)
	}
}
