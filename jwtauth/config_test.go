package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// TestDualAlgorithmConfiguration tests dual-algorithm configuration scenarios (FR-001, FR-002)
func TestDualAlgorithmConfiguration(t *testing.T) {
	// Generate test keys
	hs256Secret := make([]byte, 32)
	if _, err := rand.Read(hs256Secret); err != nil {
		t.Fatalf("Failed to generate HS256 secret: %v", err)
	}

	rs256PrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RS256 key pair: %v", err)
	}
	rs256PublicKey := &rs256PrivateKey.PublicKey

	tests := []struct {
		name          string
		options       []ConfigOption
		wantErr       bool
		errContains   string
		expectedAlgs  []string
		description   string
	}{
		{
			name:         "Both HS256 and RS256 configured",
			options:      []ConfigOption{WithHS256(hs256Secret), WithRS256(rs256PublicKey)},
			wantErr:      false,
			expectedAlgs: []string{"HS256", "RS256"},
			description:  "Should successfully configure both algorithms",
		},
		{
			name:         "Only HS256 configured",
			options:      []ConfigOption{WithHS256(hs256Secret)},
			wantErr:      false,
			expectedAlgs: []string{"HS256"},
			description:  "Should successfully configure single HS256 algorithm",
		},
		{
			name:         "Only RS256 configured",
			options:      []ConfigOption{WithRS256(rs256PublicKey)},
			wantErr:      false,
			expectedAlgs: []string{"RS256"},
			description:  "Should successfully configure single RS256 algorithm",
		},
		{
			name:        "Neither configured",
			options:     []ConfigOption{},
			wantErr:     true,
			errContains: "at least one algorithm must be configured",
			description: "Should reject config with no algorithms",
		},
		{
			name:        "Invalid HS256 secret (< 32 bytes)",
			options:     []ConfigOption{WithHS256([]byte("short"))},
			wantErr:     true,
			errContains: "at least 32 bytes",
			description: "Should reject weak HS256 secret",
		},
		{
			name:        "Nil RS256 public key",
			options:     []ConfigOption{WithRS256(nil)},
			wantErr:     true,
			errContains: "cannot be nil",
			description: "Should reject nil RS256 public key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.options...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: expected error containing %q, got nil", tt.description, tt.errContains)
					return
				}
				if tt.errContains != "" {
					valErr, ok := err.(*ValidationError)
					if !ok {
						t.Errorf("%s: expected ValidationError, got %T", tt.description, err)
						return
					}
					if !contains(valErr.Message, tt.errContains) && !contains(valErr.Error(), tt.errContains) {
						t.Errorf("%s: error %q does not contain %q", tt.description, valErr.Message, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.description, err)
			}

			if cfg == nil {
				t.Fatalf("%s: config is nil", tt.description)
			}

			// Verify expected algorithms
			availableAlgs := cfg.AvailableAlgorithms()
			if len(availableAlgs) != len(tt.expectedAlgs) {
				t.Errorf("%s: expected %d algorithms, got %d", tt.description, len(tt.expectedAlgs), len(availableAlgs))
			}

			for _, expectedAlg := range tt.expectedAlgs {
				found := false
				for _, alg := range availableAlgs {
					if alg == expectedAlg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected algorithm %s not found in %v", tt.description, expectedAlg, availableAlgs)
				}
			}
		})
	}
}

// TestAvailableAlgorithms tests the AvailableAlgorithms method returns sorted list
func TestAvailableAlgorithms(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, err := NewConfig(
		WithRS256(rs256PublicKey), // Add RS256 first
		WithHS256(hs256Secret),    // Add HS256 second
	)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	algs := cfg.AvailableAlgorithms()

	// Should be sorted alphabetically
	if len(algs) != 2 {
		t.Errorf("Expected 2 algorithms, got %d", len(algs))
	}
	if algs[0] != "HS256" {
		t.Errorf("Expected first algorithm to be HS256, got %s", algs[0])
	}
	if algs[1] != "RS256" {
		t.Errorf("Expected second algorithm to be RS256, got %s", algs[1])
	}
}

// TestBackwardCompatibility tests deprecated Algorithm() and SigningKey() methods
func TestBackwardCompatibility(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	// Single-algorithm config (legacy usage)
	cfg, err := NewConfig(WithHS256(hs256Secret))
	if err != nil {
		t.Fatalf("Failed to create single-algorithm config: %v", err)
	}

	// Deprecated Algorithm() should return the single configured algorithm
	if cfg.Algorithm() != "HS256" {
		t.Errorf("Expected Algorithm() to return HS256, got %s", cfg.Algorithm())
	}

	// Deprecated SigningKey() should return the signing key
	if cfg.SigningKey() == nil {
		t.Error("Expected SigningKey() to return non-nil key")
	}
}

// TestConfigValidatorIntegrity tests that validators are properly populated
func TestConfigValidatorIntegrity(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, err := NewConfig(WithHS256(hs256Secret))
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test internal getValidator method
	validator, exists := cfg.getValidator("HS256")
	if !exists {
		t.Error("Expected HS256 validator to exist")
	}

	if validator.signingKey == nil {
		t.Error("Expected signing key to be non-nil")
	}

	if validator.signingMethod == nil {
		t.Error("Expected signing method to be non-nil")
	}

	// Test non-existent algorithm
	_, exists = cfg.getValidator("ES256")
	if exists {
		t.Error("Expected ES256 validator to not exist")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
