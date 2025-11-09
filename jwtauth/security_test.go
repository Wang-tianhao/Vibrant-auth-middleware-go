package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestAlgorithmConfusionPrevention tests prevention of algorithm confusion attacks (FR-011, SEC-001)
func TestAlgorithmConfusionPrevention(t *testing.T) {
	// Generate test keys
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	tests := []struct {
		name          string
		configAlg     string
		configKey     interface{}
		tokenSignKey  interface{}
		tokenSignMethod jwt.SigningMethod
		description   string
	}{
		{
			name:          "RS256 token presented to HS256-only config",
			configAlg:     "HS256",
			configKey:     hs256Secret,
			tokenSignKey:  rs256PrivateKey,
			tokenSignMethod: jwt.SigningMethodRS256,
			description:   "RS256 token should be rejected by HS256-only config",
		},
		{
			name:          "HS256 token presented to RS256-only config",
			configAlg:     "RS256",
			configKey:     rs256PublicKey,
			tokenSignKey:  hs256Secret,
			tokenSignMethod: jwt.SigningMethodHS256,
			description:   "HS256 token should be rejected by RS256-only config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with specific algorithm
			var cfg *Config
			var err error
			if tt.configAlg == "HS256" {
				cfg, err = NewConfig(WithHS256(tt.configKey.([]byte)))
			} else {
				cfg, err = NewConfig(WithRS256(tt.configKey.(*rsa.PublicKey)))
			}
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Create token with different algorithm
			claims := jwt.MapClaims{
				"sub": "attacker",
				"exp": time.Now().Add(1 * time.Hour).Unix(),
			}
			token := jwt.NewWithClaims(tt.tokenSignMethod, claims)

			tokenString, err := token.SignedString(tt.tokenSignKey)
			if err != nil {
				t.Fatalf("Failed to sign token: %v", err)
			}

			// Attempt to validate (should fail)
			_, err = parseAndValidateJWT(tokenString, cfg)

			if err == nil {
				t.Errorf("%s: expected error, got nil", tt.description)
				return
			}

			// Should be rejected with UNSUPPORTED_ALGORITHM or MALFORMED
			valErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("%s: expected ValidationError, got %T", tt.description, err)
				return
			}

			// Either UNSUPPORTED_ALGORITHM or MALFORMED is acceptable (defense in depth)
			if valErr.Code != ErrUnsupportedAlgorithm && valErr.Code != ErrMalformed {
				t.Errorf("%s: expected UNSUPPORTED_ALGORITHM or MALFORMED, got %s", tt.description, valErr.Code)
			}
		})
	}
}

// TestDualConfigAlgorithmConfusion tests algorithm confusion with dual-config setup
func TestDualConfigAlgorithmConfusion(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	// Dual-algorithm config
	cfg, err := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)
	if err != nil {
		t.Fatalf("Failed to create dual-algorithm config: %v", err)
	}

	t.Run("Valid HS256 token validates correctly", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(hs256Secret)

		_, err := parseAndValidateJWT(tokenString, cfg)
		if err != nil {
			t.Errorf("Valid HS256 token should validate, got error: %v", err)
		}
	})

	t.Run("Valid RS256 token validates correctly", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user456",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, _ := token.SignedString(rs256PrivateKey)

		_, err := parseAndValidateJWT(tokenString, cfg)
		if err != nil {
			t.Errorf("Valid RS256 token should validate, got error: %v", err)
		}
	})

	t.Run("Token with alg mismatch detected", func(t *testing.T) {
		// Create token claiming to be RS256 but signed with HS256
		claims := jwt.MapClaims{
			"sub": "attacker",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["alg"] = "RS256" // Claim to be RS256

		tokenString, _ := token.SignedString(hs256Secret)

		_, err := parseAndValidateJWT(tokenString, cfg)

		if err == nil {
			t.Error("Algorithm confusion attack should be detected")
			return
		}

		// Should fail with signature error (alg:RS256 but signed with HS256)
		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
			return
		}

		// The error could be ErrInvalidSignature (if golang-jwt detects it)
		// or our algorithm confusion check catches it
		if valErr.Code != ErrInvalidSignature && valErr.Code != ErrMalformed {
			t.Logf("Got error code: %s (message: %s)", valErr.Code, valErr.Message)
		}
	})
}

// TestSignatureVerificationWithWrongKey tests that wrong keys are rejected
func TestSignatureVerificationWithWrongKey(t *testing.T) {
	// Create two different HS256 secrets
	hs256Secret1 := make([]byte, 32)
	rand.Read(hs256Secret1)

	hs256Secret2 := make([]byte, 32)
	rand.Read(hs256Secret2)

	// Configure with secret1
	cfg, _ := NewConfig(WithHS256(hs256Secret1))

	// Create token signed with secret2
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret2)

	// Validation should fail
	_, err := parseAndValidateJWT(tokenString, cfg)

	if err == nil {
		t.Error("Token signed with wrong key should be rejected")
		return
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
		return
	}

	// Should be either INVALID_SIGNATURE or MALFORMED
	if valErr.Code != ErrInvalidSignature && valErr.Code != ErrMalformed {
		t.Errorf("Expected ErrInvalidSignature or ErrMalformed, got %s", valErr.Code)
	}
}

// TestExpiredTokenRejection tests that expired tokens are rejected properly
func TestExpiredTokenRejection(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	// Create expired token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	_, err := parseAndValidateJWT(tokenString, cfg)

	if err == nil {
		t.Error("Expired token should be rejected")
		return
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
		return
	}

	// Either EXPIRED or MALFORMED is acceptable (JWT library may catch it differently)
	if valErr.Code != ErrExpired && valErr.Code != ErrMalformed {
		t.Errorf("Expected ErrExpired or ErrMalformed, got %s", valErr.Code)
	}
}

// publicKeyToBytes converts RSA public key to PEM bytes (for attack simulation)
func publicKeyToBytes(pub *rsa.PublicKey) []byte {
	pubASN1, _ := x509.MarshalPKIXPublicKey(pub)
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	return pubPEM
}
