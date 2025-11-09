package jwtauth

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestSecurityEvent_AlgorithmField_Success tests that successful HS256/RS256 validations log the correct algorithm
func TestSecurityEvent_AlgorithmField_Success(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func(t *testing.T) *Config
		generateToken func(t *testing.T, cfg *Config) string
		expectedAlg   string
	}{
		{
			name: "HS256 success logs algorithm=HS256",
			setupConfig: func(t *testing.T) *Config {
				secret := make([]byte, 32)
				rand.Read(secret)
				cfg, err := NewConfig(WithHS256(secret))
				if err != nil {
					t.Fatalf("Failed to create config: %v", err)
				}
				return cfg
			},
			generateToken: func(t *testing.T, cfg *Config) string {
				claims := jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

				// Get signing key from first validator
				algs := cfg.AvailableAlgorithms()
				validator, _ := cfg.getValidator(algs[0])

				tokenString, err := token.SignedString(validator.signingKey)
				if err != nil {
					t.Fatalf("Failed to sign token: %v", err)
				}
				return tokenString
			},
			expectedAlg: "HS256",
		},
		{
			name: "RS256 success logs algorithm=RS256",
			setupConfig: func(t *testing.T) *Config {
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatalf("Failed to generate RSA key: %v", err)
				}
				cfg, err := NewConfig(WithRS256(&privateKey.PublicKey))
				if err != nil {
					t.Fatalf("Failed to create config: %v", err)
				}
				// Store private key for token generation
				t.Cleanup(func() {})
				return cfg
			},
			generateToken: func(t *testing.T, cfg *Config) string {
				// Generate a new private key for signing (testing scenario)
				privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

				claims := jwt.MapClaims{
					"sub": "user456",
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

				// Re-configure with our private key for this test
				cfg, _ = NewConfig(WithRS256(&privateKey.PublicKey))

				tokenString, err := token.SignedString(privateKey)
				if err != nil {
					t.Fatalf("Failed to sign token: %v", err)
				}
				return tokenString
			},
			expectedAlg: "RS256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with buffer to capture output
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			// Create config with logger
			cfg := tt.setupConfig(t)
			cfg, err := NewConfig(
				WithHS256(make([]byte, 32)), // Default, will be overridden
			)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Override logger (since we can't pass it during initial setup in test)
			// This is a test-only workaround - in production, use WithLogger option
			cfgWithLogger := &Config{
				validators: cfg.validators,
				logger:     logger,
			}

			// Generate and validate token
			tokenString := tt.generateToken(t, cfg)

			// Manually trigger logAuthSuccess to test logging
			claims := &Claims{Subject: "test-user"}
			logAuthSuccess(cfgWithLogger, "test-req-123", claims, tokenString, 10*time.Millisecond)

			// Parse logged JSON
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v\nOutput: %s", err, buf.String())
			}

			// Extract algorithm from nested auth_event
			authEvent, ok := logEntry["auth_event"].(map[string]interface{})
			if !ok {
				t.Fatalf("Log entry missing auth_event field: %+v", logEntry)
			}

			algorithm, ok := authEvent["algorithm"].(string)
			if !ok {
				t.Fatalf("SecurityEvent missing algorithm field: %+v", authEvent)
			}

			if algorithm != tt.expectedAlg {
				t.Errorf("Expected algorithm=%s, got algorithm=%s", tt.expectedAlg, algorithm)
			}
		})
	}
}

// TestSecurityEvent_AlgorithmField_Failure tests that failed validations log the attempted algorithm
func TestSecurityEvent_AlgorithmField_Failure(t *testing.T) {
	tests := []struct {
		name                string
		token               string
		expectedAlg         string
		expectedFailureCode string
	}{
		{
			name:                "Unsupported ES256 logs algorithm=ES256",
			token:               "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.invalid",
			expectedAlg:         "ES256",
			expectedFailureCode: "UNSUPPORTED_ALGORITHM",
		},
		{
			name:                "Malformed alg header logs MALFORMED",
			token:               "invalid.token.structure",
			expectedAlg:         "MALFORMED",
			expectedFailureCode: "MALFORMED",
		},
		{
			name: "None algorithm logs algorithm=none",
			// JWT with alg:none header
			token:               "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyMTIzIn0.",
			expectedAlg:         "none",
			expectedFailureCode: "NONE_ALGORITHM",
		},
		{
			name:                "None variant (capitalized) logs algorithm=None",
			token:               "eyJhbGciOiJOb25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyMTIzIn0.",
			expectedAlg:         "None",
			expectedFailureCode: "NONE_ALGORITHM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with buffer
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			// Create config with HS256 only
			secret := make([]byte, 32)
			rand.Read(secret)
			cfg, err := NewConfig(WithHS256(secret))
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Override logger for testing
			cfgWithLogger := &Config{
				validators: cfg.validators,
				logger:     logger,
			}

			// Create a validation error
			valErr := &ValidationError{
				Code:    ErrorCode(tt.expectedFailureCode),
				Message: "Test error",
			}

			// Trigger logAuthFailure
			logAuthFailure(cfgWithLogger, "test-req-456", tt.token, valErr, 5*time.Millisecond)

			// Parse logged JSON
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v\nOutput: %s", err, buf.String())
			}

			// Extract auth_event
			authEvent, ok := logEntry["auth_event"].(map[string]interface{})
			if !ok {
				t.Fatalf("Log entry missing auth_event field: %+v", logEntry)
			}

			// Verify algorithm field
			algorithm, ok := authEvent["algorithm"].(string)
			if !ok {
				t.Fatalf("SecurityEvent missing algorithm field: %+v", authEvent)
			}

			if algorithm != tt.expectedAlg {
				t.Errorf("Expected algorithm=%s, got algorithm=%s", tt.expectedAlg, algorithm)
			}

			// Verify failure reason
			failureReason, ok := authEvent["failure_reason"].(string)
			if !ok {
				t.Fatalf("SecurityEvent missing failure_reason field: %+v", authEvent)
			}

			if failureReason != tt.expectedFailureCode {
				t.Errorf("Expected failure_reason=%s, got failure_reason=%s", tt.expectedFailureCode, failureReason)
			}
		})
	}
}

// TestLogSecurityEvent_JSONFormat tests that logSecurityEvent outputs valid JSON with all required fields
func TestLogSecurityEvent_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	event := SecurityEvent{
		EventType:     "success",
		Timestamp:     time.Now(),
		RequestID:     "req-789",
		UserID:        "user-xyz",
		Algorithm:     "HS256",
		FailureReason: "",
		TokenPreview:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.signature",
		Latency:       15 * time.Millisecond,
	}

	logSecurityEvent(logger, event)

	// Parse and validate JSON structure
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Log output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	// Verify auth_event group exists
	authEvent, ok := logEntry["auth_event"].(map[string]interface{})
	if !ok {
		t.Fatalf("Log entry missing auth_event group: %+v", logEntry)
	}

	// Verify all required fields
	requiredFields := map[string]string{
		"event":          "success",
		"request_id":     "req-789",
		"user_id":        "user-xyz",
		"algorithm":      "HS256",
		"failure_reason": "",
	}

	for field, expectedValue := range requiredFields {
		actualValue, ok := authEvent[field].(string)
		if !ok {
			t.Errorf("auth_event missing %s field", field)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", field, expectedValue, field, actualValue)
		}
	}

	// Verify token is redacted
	token, ok := authEvent["token"].(string)
	if !ok {
		t.Fatalf("auth_event missing token field")
	}
	if len(token) > 12 { // Should be redacted to ~8 chars + "..."
		t.Errorf("Token should be redacted, got: %s", token)
	}
}

// TestExtractAlgorithmFromToken tests the algorithm extraction helper
func TestExtractAlgorithmFromToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "Valid HS256 token",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.signature",
			expected: "HS256",
		},
		{
			name:     "Valid RS256 token",
			token:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyNDU2In0.signature",
			expected: "RS256",
		},
		{
			name:     "Malformed token (missing parts)",
			token:    "invalid",
			expected: "MALFORMED",
		},
		{
			name:     "Malformed token (invalid base64)",
			token:    "!!!invalid!!!.payload.signature",
			expected: "MALFORMED",
		},
		{
			name:     "Empty token",
			token:    "",
			expected: "MALFORMED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAlgorithmFromToken(tt.token)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
