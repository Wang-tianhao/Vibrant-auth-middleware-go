package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user/vibrant-auth-middleware-go/jwtauth"
)

func main() {
	// Choose example: single-algorithm or dual-algorithm
	// Uncomment one of the following:

	runSingleAlgorithmExample()
	// runDualAlgorithmExample()
}

func runSingleAlgorithmExample() {
	// Configure JWT middleware with HS256
	secret := []byte("your-256-bit-secret-key-min-32-bytes-here-for-demo!")

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := jwtauth.NewConfig(
		jwtauth.WithHS256(secret),
		jwtauth.WithCookie("auth_token"),      // Optional: also check cookies
		jwtauth.WithClockSkew(30*time.Second), // Optional: 30s tolerance
		jwtauth.WithLogger(logger),            // Enable structured logging
	)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	startServer(cfg, "Single Algorithm (HS256)")
}

// runDualAlgorithmExample demonstrates dual-algorithm configuration (HS256 + RS256)
// Use case: Accept tokens from multiple issuers using different signing methods
func runDualAlgorithmExample() {
	// HS256 secret for internal tokens
	hs256Secret := []byte("your-256-bit-secret-key-min-32-bytes-here-for-demo!")

	// RS256 public key for external partner tokens
	// In production, load from file or environment variable
	rs256PublicKey := loadRS256PublicKey()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Configure middleware to accept BOTH HS256 and RS256 tokens
	cfg, err := jwtauth.NewConfig(
		jwtauth.WithHS256(hs256Secret),      // Accept internal HS256 tokens
		jwtauth.WithRS256(rs256PublicKey),   // Accept external RS256 tokens
		jwtauth.WithCookie("auth_token"),    // Optional: also check cookies
		jwtauth.WithClockSkew(30*time.Second), // Optional: 30s tolerance
		jwtauth.WithLogger(logger),          // Enable structured logging
	)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	log.Println("Dual-algorithm mode enabled!")
	log.Printf("Configured algorithms: %v\n", cfg.AvailableAlgorithms())

	startServer(cfg, "Dual Algorithm (HS256 + RS256)")
}

// loadRS256PublicKey loads an RSA public key for RS256 validation
// In production, load from file, environment variable, or JWKS endpoint
func loadRS256PublicKey() *rsa.PublicKey {
	// Example PEM-encoded RSA public key (2048-bit)
	// In production, replace with your actual public key
	publicKeyPEM := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2p7VWW8L3FXvJyq3Oi0V
bR5hGH7YdD5FxqH5tJoKJN8ZyMQqWJvKP3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb
3L5pQ8vZxMQqWJvKP3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb3L5pQ8vZxMQqWJvK
P3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb3L5pQ8vZxMQqWJvKP3wKqRB0cF9VxS3L
2JH5PqR7X8KfQfNb3L5pQ8vZxMQqWJvKP3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb
3L5pQ8vZxMQqWJvKP3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb3L5pQ8vZxMQqWJvK
P3wKqRB0cF9VxS3L2JH5PqR7X8KfQfNb3L5pQ8vZxMQqWJvKP3wKqRB0cF9VxS3L
2wIDAQAB
-----END PUBLIC KEY-----`

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		log.Fatal("Failed to parse PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		log.Fatal("Not an RSA public key")
	}

	return rsaPub
}

func startServer(cfg *jwtauth.Config, mode string) {
	// Create Gin router
	r := gin.Default()

	// Public routes (no authentication required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to the JWT Auth Demo API",
			"mode":    mode,
			"endpoints": gin.H{
				"GET /health":      "Health check (public)",
				"GET /api/profile": "Get user profile (protected)",
				"GET /api/data":    "Get user data (protected)",
			},
			"auth": "Add 'Authorization: Bearer <token>' header to access protected routes",
		})
	})

	// Protected routes (authentication required)
	authorized := r.Group("/api")
	authorized.Use(jwtauth.JWTAuth(cfg))
	{
		authorized.GET("/profile", getProfile)
		authorized.GET("/data", getData)
	}

	log.Printf("Starting Gin server on :8080 (%s)\n", mode)
	log.Println("Public endpoints: GET /health, GET /")
	log.Println("Protected endpoints: GET /api/profile, GET /api/data")
	log.Println("Use: curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/profile")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getProfile(c *gin.Context) {
	// Retrieve validated JWT claims
	claims, ok := jwtauth.GetClaims(c.Request.Context())
	if !ok {
		c.JSON(500, gin.H{"error": "claims not found"})
		return
	}

	// Access standard claims
	userID := claims.Subject

	// Access custom claims (if present)
	email := ""
	if emailVal, ok := claims.Custom["email"]; ok {
		if emailStr, ok := emailVal.(string); ok {
			email = emailStr
		}
	}

	c.JSON(200, gin.H{
		"user_id":    userID,
		"email":      email,
		"expires_at": claims.ExpiresAt.Format(time.RFC3339),
		"issuer":     claims.Issuer,
	})
}

func getData(c *gin.Context) {
	// Retrieve validated JWT claims
	claims, ok := jwtauth.GetClaims(c.Request.Context())
	if !ok {
		c.JSON(500, gin.H{"error": "claims not found"})
		return
	}

	// Example: Role-based access control
	role := ""
	if roleVal, ok := claims.Custom["role"]; ok {
		if roleStr, ok := roleVal.(string); ok {
			role = roleStr
		}
	}

	if role != "admin" {
		c.JSON(403, gin.H{
			"error":   "forbidden",
			"message": "admin role required",
		})
		return
	}

	// Return sensitive data (only for admins)
	c.JSON(200, gin.H{
		"data": []string{"sensitive", "information", "here"},
		"user": claims.Subject,
	})
}
