# Quickstart: JWT Authentication Middleware

**Target Time**: <15 minutes from zero to protected endpoints
**Prerequisites**: Go 1.23+, basic familiarity with JWT concepts

---

## 5-Minute HTTP (Gin) Integration

### Step 1: Install (30 seconds)

```bash
# Initialize Go module (if new project)
go mod init myapp

# Install dependencies
go get github.com/golang-jwt/jwt/v5
go get github.com/gin-gonic/gin
go get github.com/user/project/jwtauth  # Replace with actual import path
```

### Step 2: Configure Middleware (2 minutes)

```go
package main

import (
    "log"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/user/project/jwtauth"
    "log/slog"
)

func main() {
    // Configure JWT middleware with HS256 (HMAC-SHA256)
    secret := []byte("your-256-bit-secret-key-min-32-bytes!")

    cfg, err := jwtauth.NewConfig(
        jwtauth.WithHS256(secret),
        jwtauth.WithCookie("auth_token"),      // Optional: also check cookies
        jwtauth.WithClockSkew(30*time.Second), // Optional: 30s tolerance (default 60s)
        jwtauth.WithLogger(slog.Default()),    // Optional: enable logging
    )
    if err != nil {
        log.Fatalf("Config error: %v", err)
    }

    // Create Gin router
    r := gin.Default()

    // Apply JWT middleware globally (all routes protected)
    r.Use(jwtauth.JWTAuth(cfg))

    // Or apply to specific routes only
    authorized := r.Group("/api")
    authorized.Use(jwtauth.JWTAuth(cfg))
    {
        authorized.GET("/profile", getProfile)
        authorized.POST("/data", postData)
    }

    r.Run(":8080")
}
```

### Step 3: Access Claims in Handlers (1 minute)

```go
func getProfile(c *gin.Context) {
    // Retrieve validated JWT claims
    claims, ok := jwtauth.GetClaims(c.Request.Context())
    if !ok {
        // Should never happen if middleware is applied
        c.JSON(500, gin.H{"error": "claims not found"})
        return
    }

    // Use claims
    userID := claims.Subject
    email := claims.Custom["email"].(string) // Custom claims

    c.JSON(200, gin.H{
        "user_id": userID,
        "email":   email,
        "expires": claims.ExpiresAt,
    })
}
```

### Step 4: Test (1.5 minutes)

```bash
# Start server
go run main.go

# Test without token (should fail with 401)
curl http://localhost:8080/api/profile

# Test with valid token (create token first - see Token Generation below)
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/api/profile

# Test with cookie
curl --cookie "auth_token=YOUR_JWT_TOKEN" http://localhost:8080/api/profile
```

---

## 10-Minute gRPC Integration

### Step 1: Install (30 seconds)

```bash
go get github.com/golang-jwt/jwt/v5
go get google.golang.org/grpc
go get github.com/user/project/jwtauth
```

### Step 2: Configure Interceptor (3 minutes)

```go
package main

import (
    "context"
    "crypto/rsa"
    "log"
    "os"
    "google.golang.org/grpc"
    "github.com/user/project/jwtauth"
    pb "myapp/proto"  // Your protobuf definitions
)

func main() {
    // Load RS256 public key (for production: separate signing service)
    pemBytes, err := os.ReadFile("public_key.pem")
    if err != nil {
        log.Fatalf("Failed to read public key: %v", err)
    }

    pubKey, err := jwtauth.ParseRSAPublicKeyFromPEM(pemBytes)
    if err != nil {
        log.Fatalf("Failed to parse public key: %v", err)
    }

    // Configure JWT middleware with RS256
    cfg, err := jwtauth.NewConfig(
        jwtauth.WithRS256(pubKey),
        jwtauth.WithLogger(slog.Default()),
    )
    if err != nil {
        log.Fatalf("Config error: %v", err)
    }

    // Create gRPC server with JWT interceptor
    srv := grpc.NewServer(
        grpc.UnaryInterceptor(jwtauth.UnaryServerInterceptor(cfg)),
    )

    // Register your service
    pb.RegisterUserServiceServer(srv, &userServiceImpl{})

    // Start server
    lis, _ := net.Listen("tcp", ":50051")
    log.Println("gRPC server listening on :50051")
    srv.Serve(lis)
}
```

### Step 3: Access Claims in Handlers (2 minutes)

```go
type userServiceImpl struct {
    pb.UnimplementedUserServiceServer
}

func (s *userServiceImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // Retrieve validated JWT claims
    claims := jwtauth.MustGetClaims(ctx) // Panics if not present (safe with middleware)

    // Use claims
    userID := claims.Subject
    role := claims.Custom["role"].(string)

    // Authorization check (beyond authentication)
    if role != "admin" {
        return nil, status.Errorf(codes.PermissionDenied, "admin access required")
    }

    // Fetch and return user data
    return &pb.User{
        Id:    userID,
        Email: claims.Custom["email"].(string),
    }, nil
}
```

### Step 4: Client Test (4.5 minutes)

```go
// Client code to test gRPC with JWT
func main() {
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Create context with JWT in metadata
    token := "YOUR_JWT_TOKEN"
    md := metadata.Pairs("authorization", "Bearer "+token)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    // Make authenticated RPC call
    resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: "123"})
    if err != nil {
        log.Fatalf("RPC failed: %v", err)
    }
    log.Printf("User: %v", resp)
}
```

---

## Token Generation (For Testing)

### Create Test Tokens with HS256

```go
package main

import (
    "fmt"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func main() {
    secret := []byte("your-256-bit-secret-key-min-32-bytes!")

    // Create claims
    claims := jwt.MapClaims{
        "sub":   "user123",                          // Subject (user ID)
        "email": "user@example.com",                 // Custom claim
        "role":  "admin",                            // Custom claim
        "exp":   time.Now().Add(time.Hour).Unix(),   // Expires in 1 hour
        "nbf":   time.Now().Unix(),                  // Not before now
        "iat":   time.Now().Unix(),                  // Issued at now
    }

    // Create token with HS256
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    // Sign token
    tokenString, err := token.SignedString(secret)
    if err != nil {
        panic(err)
    }

    fmt.Println("Token:", tokenString)
    // Use this token in Authorization header or cookie
}
```

### Generate RS256 Key Pair (Production)

```bash
# Generate private key (keep secret!)
openssl genrsa -out private_key.pem 2048

# Extract public key (distribute to services)
openssl rsa -in private_key.pem -pubout -out public_key.pem

# Sign tokens with private key (auth service)
# Validate tokens with public key (resource services)
```

```go
// Create RS256 token (auth service with private key)
func signRS256Token(privateKey *rsa.PrivateKey) string {
    claims := jwt.MapClaims{
        "sub": "user123",
        "exp": time.Now().Add(time.Hour).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    tokenString, _ := token.SignedString(privateKey)
    return tokenString
}
```

---

## Configuration Options Reference

### Algorithm Selection

```go
// HS256 (HMAC-SHA256) - Symmetric key, simpler setup
jwtauth.WithHS256([]byte("secret-min-32-bytes"))

// RS256 (RSA-SHA256) - Asymmetric key, better for distributed systems
pubKey, _ := jwtauth.ParseRSAPublicKeyFromPEM(pemBytes)
jwtauth.WithRS256(pubKey)
```

**When to use each**:
- **HS256**: Single service, simpler key management, faster (~10-20% faster)
- **RS256**: Microservices, zero-trust architecture, auth/resource service separation

---

### Token Extraction

```go
// Header only (default)
jwtauth.NewConfig(jwtauth.WithHS256(secret))

// Header + Cookie (fallback)
jwtauth.WithCookie("auth_token")  // Check cookie if header missing

// Cookie name customization
jwtauth.WithCookie("jwt")
jwtauth.WithCookie("session_token")
```

**Precedence**: Authorization header > Cookie (if both present, header wins)

---

### Clock Skew Tolerance

```go
// Strict validation (not recommended - NTP drift issues)
jwtauth.WithClockSkew(0)

// Balanced (recommended for most cases)
jwtauth.WithClockSkew(30 * time.Second)

// Lenient (default - tolerates minor time sync issues)
jwtauth.WithClockSkew(60 * time.Second)
```

**Why needed**: Server clocks may drift slightly, causing valid tokens to fail at exact expiration

---

### Structured Logging

```go
// Default logger
jwtauth.WithLogger(slog.Default())

// Custom logger with JSON output
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
jwtauth.WithLogger(slog.New(handler))

// Disable logging (not recommended)
jwtauth.NewConfig(jwtauth.WithHS256(secret))  // No WithLogger = silent
```

**Log Fields**:
- `event`: "auth_success" or "auth_failure"
- `timestamp`: RFC3339 timestamp
- `request_id`: Correlation ID
- `failure_reason`: "expired", "invalid_signature", etc. (on failure)
- `user_id`: Subject claim (on success)
- Tokens automatically redacted

---

### Required Claims

```go
// Enforce custom claims presence
jwtauth.WithRequiredClaims("sub", "email", "role")

// Validation fails if any required claim missing
```

**Note**: Standard `exp` claim always required (built-in)

---

## Common Patterns

### Public + Protected Routes

```go
func main() {
    r := gin.Default()
    cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

    // Public routes (no authentication)
    r.GET("/health", healthCheck)
    r.POST("/login", login)

    // Protected routes (authentication required)
    authorized := r.Group("/api")
    authorized.Use(jwtauth.JWTAuth(cfg))
    {
        authorized.GET("/profile", getProfile)
        authorized.POST("/data", postData)
    }

    r.Run(":8080")
}
```

---

### Custom Claims Validation

```go
func requireAdmin(c *gin.Context) {
    claims, _ := jwtauth.GetClaims(c.Request.Context())

    role, ok := claims.Custom["role"].(string)
    if !ok || role != "admin" {
        c.AbortWithStatusJSON(403, gin.H{"error": "admin access required"})
        return
    }

    c.Next()
}

// Usage
r.GET("/admin/users", jwtauth.JWTAuth(cfg), requireAdmin, listUsers)
```

---

### Error Handling

```go
// Gin automatically aborts with 401 on auth failure
// Custom error handler (optional)
r.Use(func(c *gin.Context) {
    c.Next()

    if c.Writer.Status() == 401 {
        // Custom 401 response
        c.JSON(401, gin.H{
            "error": "authentication_required",
            "message": "Please provide a valid JWT token",
        })
    }
})
```

---

### Testing with Mock Claims

```go
func TestProtectedEndpoint(t *testing.T) {
    // Create test claims
    claims := jwtauth.MockClaims("user123", time.Hour)

    // Inject into test context
    ctx := jwtauth.WithMockClaims(context.Background(), claims)

    // Test handler with mocked authentication
    req := httptest.NewRequest("GET", "/profile", nil).WithContext(ctx)
    w := httptest.NewRecorder()

    getProfile(gin.CreateTestContext(w).Request.Context())

    assert.Equal(t, 200, w.Code)
}
```

---

## Troubleshooting

### "unauthorized: missing_token"

**Problem**: JWT not found in request

**Solutions**:
```bash
# Verify Authorization header format
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/profile

# Check cookie name matches config
curl --cookie "auth_token=YOUR_TOKEN" http://localhost:8080/api/profile

# Verify token not expired (check exp claim)
```

---

### "unauthorized: invalid_signature"

**Problem**: Token signature doesn't match

**Solutions**:
- Verify same secret used for signing and validation (HS256)
- Verify public key matches private key used for signing (RS256)
- Check token not manually modified
- Ensure algorithm matches config (HS256 vs RS256)

---

### "unauthorized: expired"

**Problem**: Token exp claim past current time

**Solutions**:
- Generate new token with future expiration
- Increase clock skew tolerance: `WithClockSkew(60*time.Second)`
- Verify server clocks synchronized (NTP)

---

### "unauthorized: algorithm_mismatch"

**Problem**: Token algorithm doesn't match config

**Solutions**:
- Check token signed with HS256 but config uses RS256 (or vice versa)
- Ensure `alg` header in JWT matches `WithHS256`/`WithRS256` config
- **Security**: Never allow `alg: none` (middleware rejects automatically)

---

### Claims Not Found in Handler

**Problem**: `GetClaims()` returns `ok=false`

**Solutions**:
- Verify middleware applied to route: `r.Use(jwtauth.JWTAuth(cfg))`
- Check middleware runs before handler in chain
- Confirm authentication succeeded (no 401 response)

---

## Performance Tips

### Minimize Allocations

```go
// Reuse Config across requests (already done - Config immutable and reused)
cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

// Don't recreate middleware per request
r.Use(jwtauth.JWTAuth(cfg))  // ✅ Once at startup

// Avoid
r.GET("/route", func(c *gin.Context) {
    cfg, _ := jwtauth.NewConfig(...)  // ❌ Per request (huge overhead)
    jwtauth.JWTAuth(cfg)(c)
})
```

---

### Benchmark Your Setup

```go
// Create benchmark_test.go
func BenchmarkJWTAuth(b *testing.B) {
    cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))
    middleware := jwtauth.JWTAuth(cfg)

    token := createTestToken(secret)
    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        w := httptest.NewRecorder()
        c, _ := gin.CreateTestContext(w)
        c.Request = req
        middleware(c)
    }
}

// Run: go test -bench=. -benchmem
// Target: <1000 ns/op, <3 allocs/op
```

---

## Security Checklist

Before Production:

- [ ] Use strong secrets (HS256: ≥32 bytes random, RS256: ≥2048-bit keys)
- [ ] Enable logging: `WithLogger(slog.Default())`
- [ ] Set reasonable clock skew (30-60 seconds)
- [ ] Use RS256 for microservices (asymmetric keys)
- [ ] Store secrets in environment variables or secret managers (never hardcode)
- [ ] Set short token expiration (≤1 hour recommended)
- [ ] Validate custom claims in handlers (middleware only does authentication)
- [ ] Use HTTPS/TLS in production (JWT in plaintext HTTP = insecure)
- [ ] Monitor auth failure rates (unusual patterns = potential attack)
- [ ] Rotate signing keys periodically (support key rotation in auth service)

---

## Next Steps

- **Production Deployment**: Add rate limiting, HTTPS, key rotation
- **Authorization**: Implement role-based access control (RBAC) using `claims.Custom`
- **Token Refresh**: Implement refresh token flow (separate from JWT validation)
- **Monitoring**: Integrate with APM tools (Datadog, New Relic, Prometheus)
- **Testing**: Write integration tests for auth flows, security tests for attack scenarios

---

## Getting Help

- **API Reference**: See [contracts/public-api.md](./contracts/public-api.md)
- **Data Model**: See [data-model.md](./data-model.md)
- **Research**: See [research.md](./research.md) for library selection rationale
- **Issues**: GitHub issues for bug reports and feature requests
- **Security**: Email security@yourcompany.com for security vulnerabilities (private disclosure)
