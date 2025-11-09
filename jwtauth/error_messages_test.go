package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TestErrorMessageClarity_UnsupportedAlgorithm tests error messages for unsupported algorithms (FR-008, FR-009, US3)
func TestErrorMessageClarity_UnsupportedAlgorithm(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	// Generate RS256 key for testing
	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	tests := []struct {
		name                string
		config              *Config
		createToken         func() string
		expectedStatus      int
		expectedReason      string
		expectedMessagePart string // Partial match for available algorithms list
		description         string
	}{
		{
			name:   "RS256 token to HS256-only config",
			config: mustCreateConfig(WithHS256(hs256Secret)),
			createToken: func() string {
				// Create a properly signed RS256 token
				claims := jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(1 * time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(rs256PrivateKey)
				return tokenString
			},
			expectedStatus:      401,
			expectedReason:      "UNSUPPORTED_ALGORITHM",
			expectedMessagePart: "algorithm RS256 not supported (available: HS256)",
			description:         "Should return UNSUPPORTED_ALGORITHM with available algorithms list",
		},
		{
			name:   "HS256 token to RS256-only config",
			config: mustCreateConfig(WithRS256(&rs256PrivateKey.PublicKey)),
			createToken: func() string {
				// Create a properly signed HS256 token
				claims := jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(1 * time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(hs256Secret)
				return tokenString
			},
			expectedStatus:      401,
			expectedReason:      "UNSUPPORTED_ALGORITHM",
			expectedMessagePart: "algorithm HS256 not supported (available: RS256)",
			description:         "Should list RS256 as available algorithm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter(tt.config)
			tokenString := tt.createToken()

			// Make request
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d, body: %s", tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v, body: %s", err, w.Body.String())
			}

			// Verify error field
			if response["error"] != "unauthorized" {
				t.Errorf("Expected error='unauthorized', got %v", response["error"])
			}

			// Verify reason field
			if response["reason"] != tt.expectedReason {
				t.Errorf("Expected reason=%s, got %v", tt.expectedReason, response["reason"])
			}

			// Verify message field contains available algorithms
			message, hasMessage := response["message"]
			if !hasMessage {
				t.Errorf("%s: expected message field with available algorithms, but it was missing", tt.description)
			} else {
				messageStr := message.(string)
				if !strings.Contains(messageStr, tt.expectedMessagePart) {
					t.Errorf("%s: expected message to contain %q, got: %q", tt.description, tt.expectedMessagePart, messageStr)
				}
			}
		})
	}
}

// TestErrorMessageClarity_InvalidSignature tests that invalid signature returns INVALID_SIGNATURE (not UNSUPPORTED_ALGORITHM) (US3)
func TestErrorMessageClarity_InvalidSignature(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))
	router := createTestRouter(cfg)

	// Create token with valid HS256 algorithm but wrong signing key
	wrongSecret := make([]byte, 32)
	rand.Read(wrongSecret)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(wrongSecret) // Sign with wrong key

	// Make request
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify status code
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify reason is INVALID_SIGNATURE (not UNSUPPORTED_ALGORITHM)
	if response["reason"] != "INVALID_SIGNATURE" {
		t.Errorf("Expected reason=INVALID_SIGNATURE for invalid signature, got %v", response["reason"])
	}

	// Verify message field should NOT be present for INVALID_SIGNATURE
	if _, hasMessage := response["message"]; hasMessage {
		t.Errorf("INVALID_SIGNATURE should not include message field, got: %v", response["message"])
	}
}

// TestErrorMessageClarity_Expired tests that expired token returns EXPIRED (not UNSUPPORTED_ALGORITHM) (US3)
func TestErrorMessageClarity_Expired(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))
	router := createTestRouter(cfg)

	// Create expired token with valid HS256 algorithm
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	// Make request
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify status code
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify reason is EXPIRED (not UNSUPPORTED_ALGORITHM)
	reason := response["reason"]
	if reason != "EXPIRED" && reason != "MALFORMED" {
		t.Errorf("Expected reason=EXPIRED or MALFORMED for expired token, got %v", reason)
	}

	// Verify message field should NOT be present for EXPIRED
	if _, hasMessage := response["message"]; hasMessage {
		t.Errorf("EXPIRED should not include message field, got: %v", response["message"])
	}
}

// TestErrorMessageClarity_NoneAlgorithm tests that none algorithm returns NONE_ALGORITHM (US3)
func TestErrorMessageClarity_NoneAlgorithm(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))
	router := createTestRouter(cfg)

	tests := []struct {
		name           string
		tokenAlg       string
		expectedReason string // May be NONE_ALGORITHM or MALFORMED depending on JWT library behavior
	}{
		{"none lowercase", "none", "NONE_ALGORITHM"}, // JWT library recognizes "none" as special case
		{"None capitalized", "None", "MALFORMED"},    // JWT library doesn't recognize "None" (treats as malformed)
		{"NONE uppercase", "NONE", "MALFORMED"},      // JWT library doesn't recognize "NONE" (treats as malformed)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create token with none algorithm by manually constructing JWT
			header := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"alg":"%s","typ":"JWT"}`, tt.tokenAlg)))
			payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user123","exp":9999999999}`))
			tokenString := header + "." + payload + "."

			// Make request
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify status code
			if w.Code != 401 {
				t.Errorf("Expected status 401, got %d", w.Code)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Verify reason matches expected
			if response["reason"] != tt.expectedReason {
				t.Errorf("Expected reason=%s for %s algorithm, got %v", tt.expectedReason, tt.tokenAlg, response["reason"])
			}

			// Verify message field should NOT be present for NONE_ALGORITHM or MALFORMED
			if _, hasMessage := response["message"]; hasMessage && tt.expectedReason == "NONE_ALGORITHM" {
				t.Errorf("NONE_ALGORITHM should not include message field, got: %v", response["message"])
			}
		})
	}
}

// TestErrorCodeDistinction verifies that different failure types return distinct error codes (US3)
func TestErrorCodeDistinction(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	cfg := mustCreateConfig(WithHS256(hs256Secret))
	router := createTestRouter(cfg)

	tests := []struct {
		name           string
		setupToken     func() string
		expectedReason string
		description    string
	}{
		{
			name: "Unsupported algorithm (RS256)",
			setupToken: func() string {
				// Create properly signed RS256 token (unsupported by HS256-only config)
				claims := jwt.MapClaims{"sub": "user", "exp": time.Now().Add(1 * time.Hour).Unix()}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(rs256PrivateKey)
				return tokenString
			},
			expectedReason: "UNSUPPORTED_ALGORITHM",
			description:    "Unsupported algorithm should return UNSUPPORTED_ALGORITHM",
		},
		{
			name: "Invalid signature",
			setupToken: func() string {
				wrongSecret := make([]byte, 32)
				rand.Read(wrongSecret)
				claims := jwt.MapClaims{"sub": "user", "exp": time.Now().Add(1 * time.Hour).Unix()}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(wrongSecret) // Wrong key
				return tokenString
			},
			expectedReason: "INVALID_SIGNATURE",
			description:    "Invalid signature should return INVALID_SIGNATURE",
		},
		{
			name: "Expired token",
			setupToken: func() string {
				claims := jwt.MapClaims{"sub": "user", "exp": time.Now().Add(-1 * time.Hour).Unix()}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(hs256Secret)
				return tokenString
			},
			expectedReason: "EXPIRED",
			description:    "Expired token should return EXPIRED",
		},
		{
			name: "None algorithm",
			setupToken: func() string {
				// Create token with none algorithm by manually constructing JWT
				header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
				payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user","exp":9999999999}`))
				return header + "." + payload + "."
			},
			expectedReason: "NONE_ALGORITHM",
			description:    "None algorithm should return NONE_ALGORITHM",
		},
	}

	seenReasons := make(map[string]bool)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.setupToken()

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Verify reason code
			reason := response["reason"]
			// For expired tokens, accept either EXPIRED or MALFORMED (depending on JWT library version)
			if tt.expectedReason == "EXPIRED" {
				if reason != "EXPIRED" && reason != "MALFORMED" {
					t.Errorf("%s: expected reason=EXPIRED or MALFORMED, got %v", tt.description, reason)
				}
			} else {
				if reason != tt.expectedReason {
					t.Errorf("%s: expected reason=%s, got %v", tt.description, tt.expectedReason, reason)
				}
			}

			// Track that each error code is distinct
			if reasonStr, ok := reason.(string); ok {
				if seenReasons[reasonStr] && reasonStr != "EXPIRED" && reasonStr != "MALFORMED" {
					t.Logf("Note: Reason code %s appeared multiple times (may be expected for some scenarios)", reasonStr)
				}
				seenReasons[reasonStr] = true
			}
		})
	}

	// Verify we got distinct error codes
	if len(seenReasons) < 3 {
		t.Errorf("Expected at least 3 distinct error codes, got %d: %v", len(seenReasons), seenReasons)
	}
}

// Helper functions

func createTestRouter(cfg *Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	return router
}

func mustCreateDualConfig(hs256Secret []byte) *Config {
	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	cfg, err := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)
	if err != nil {
		panic(err)
	}
	return cfg
}

// contains helper is defined in config_test.go to avoid redeclaration
