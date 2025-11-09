package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// Set Gin to test mode to suppress logs
	gin.SetMode(gin.TestMode)
}

// TestGinMiddlewareDualAlgorithm tests Gin middleware with dual-algorithm config (FR-007)
func TestGinMiddlewareDualAlgorithm(t *testing.T) {
	// Generate test keys
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	rs256PrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rs256PublicKey := &rs256PrivateKey.PublicKey

	// Create dual-algorithm config
	cfg, err := NewConfig(
		WithHS256(hs256Secret),
		WithRS256(rs256PublicKey),
	)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create test router with middleware
	router := gin.New()
	router.Use(JWTAuth(cfg))
	router.GET("/protected", func(c *gin.Context) {
		claims, _ := GetClaims(c.Request.Context())
		c.JSON(200, gin.H{
			"message": "success",
			"user_id": claims.Subject,
		})
	})

	tests := []struct {
		name           string
		tokenAlg       string
		signingKey     interface{}
		signingMethod  jwt.SigningMethod
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "HS256 token validates successfully",
			tokenAlg:       "HS256",
			signingKey:     hs256Secret,
			signingMethod:  jwt.SigningMethodHS256,
			expectedStatus: 200,
			description:    "Valid HS256 token should return 200 OK",
		},
		{
			name:           "RS256 token validates successfully",
			tokenAlg:       "RS256",
			signingKey:     rs256PrivateKey,
			signingMethod:  jwt.SigningMethodRS256,
			expectedStatus: 200,
			description:    "Valid RS256 token should return 200 OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create token
			claims := jwt.MapClaims{
				"sub": "user123",
				"exp": time.Now().Add(1 * time.Hour).Unix(),
			}
			token := jwt.NewWithClaims(tt.signingMethod, claims)
			token.Header["alg"] = tt.tokenAlg

			tokenString, err := token.SignedString(tt.signingKey)
			if err != nil {
				t.Fatalf("Failed to sign token: %v", err)
			}

			// Create request
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}

			// For 401 responses, verify error reason is included
			if tt.expectedStatus == 401 && tt.expectedError != "" {
				body := w.Body.String()
				if !contains(body, tt.expectedError) {
					t.Errorf("%s: expected error reason %q in response, got: %s", tt.description, tt.expectedError, body)
				}
			}
		})
	}
}

// TestGinMiddlewareBackwardCompatibility tests backward compatibility with single-algorithm config
func TestGinMiddlewareBackwardCompatibility(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	tests := []struct {
		name        string
		config      *Config
		tokenAlg    string
		signingKey  interface{}
		signingMethod jwt.SigningMethod
		shouldPass  bool
		description string
	}{
		{
			name:        "Single HS256 config - HS256 token passes",
			config:      mustCreateConfig(WithHS256(hs256Secret)),
			tokenAlg:    "HS256",
			signingKey:  hs256Secret,
			signingMethod: jwt.SigningMethodHS256,
			shouldPass:  true,
			description: "Legacy HS256-only config should still work",
		},
		{
			name:        "Single HS256 config - RS256 token rejected",
			config:      mustCreateConfig(WithHS256(hs256Secret)),
			tokenAlg:    "RS256",
			signingKey:  mustGenerateRSAKey(),
			signingMethod: jwt.SigningMethodRS256,
			shouldPass:  false,
			description: "HS256-only config should reject RS256 tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(JWTAuth(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Create token
			claims := jwt.MapClaims{
				"sub": "user123",
				"exp": time.Now().Add(1 * time.Hour).Unix(),
			}
			token := jwt.NewWithClaims(tt.signingMethod, claims)
			token.Header["alg"] = tt.tokenAlg

			tokenString, _ := token.SignedString(tt.signingKey)

			// Make request
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.shouldPass {
				if w.Code != 200 {
					t.Errorf("%s: expected 200, got %d", tt.description, w.Code)
				}
			} else {
				if w.Code != 401 {
					t.Errorf("%s: expected 401, got %d", tt.description, w.Code)
				}
			}
		})
	}
}

// TestGinMiddlewareMissingToken tests missing token handling
func TestGinMiddlewareMissingToken(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	router := gin.New()
	router.Use(JWTAuth(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Request without Authorization header
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for missing token, got %d", w.Code)
	}

	body := w.Body.String()
	if !contains(body, "MISSING_TOKEN") {
		t.Errorf("Expected MISSING_TOKEN error, got: %s", body)
	}
}

// TestGinMiddlewareExpiredToken tests expired token handling
func TestGinMiddlewareExpiredToken(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	router := gin.New()
	router.Use(JWTAuth(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create expired token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for expired token, got %d", w.Code)
	}

	body := w.Body.String()
	// Either EXPIRED or MALFORMED is acceptable (JWT library may catch it differently)
	if !contains(body, "EXPIRED") && !contains(body, "MALFORMED") {
		t.Errorf("Expected EXPIRED or MALFORMED error, got: %s", body)
	}
}

// TestGinMiddlewareClaimsInjection tests that claims are properly injected into context
func TestGinMiddlewareClaimsInjection(t *testing.T) {
	hs256Secret := make([]byte, 32)
	rand.Read(hs256Secret)

	cfg, _ := NewConfig(WithHS256(hs256Secret))

	var extractedSubject string
	router := gin.New()
	router.Use(JWTAuth(cfg))
	router.GET("/protected", func(c *gin.Context) {
		claims, _ := GetClaims(c.Request.Context())
		extractedSubject = claims.Subject
		c.JSON(200, gin.H{"user_id": claims.Subject})
	})

	// Create token
	claims := jwt.MapClaims{
		"sub": "testuser456",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(hs256Secret)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	if extractedSubject != "testuser456" {
		t.Errorf("Expected subject 'testuser456', got %q", extractedSubject)
	}
}

// Helper functions

func mustCreateConfig(opts ...ConfigOption) *Config {
	cfg, err := NewConfig(opts...)
	if err != nil {
		panic(err)
	}
	return cfg
}

func mustGenerateRSAKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}
