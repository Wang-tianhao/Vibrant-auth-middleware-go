# Public API Contract: JWT Authentication Middleware

**Feature**: JWT Authentication Middleware
**Created**: 2025-11-09
**Version**: v1.0.0 (initial)
**Purpose**: Define the public API surface for the JWT authentication middleware library

## Package Declaration

**Package Name**: `jwtauth` (or `vibrantauth`, `authmw` - to be decided during implementation)
**Import Path**: `github.com/user/project/jwtauth`

---

## Public Functions

### Middleware Construction

#### Gin HTTP Middleware

```go
// JWTAuth creates a Gin middleware handler for JWT authentication.
// Returns gin.HandlerFunc that validates JWT from Authorization header or cookie.
// On success: injects claims into gin.Context and calls c.Next()
// On failure: aborts with 401 Unauthorized and JSON error response
//
// Example:
//   cfg := NewConfig(WithHS256([]byte("secret")))
//   r.Use(JWTAuth(cfg))
func JWTAuth(config *Config) gin.HandlerFunc
```

**Input**:
- `config *Config`: Immutable configuration (MUST be non-nil)

**Output**:
- `gin.HandlerFunc`: Middleware function compatible with Gin router

**Behavior**:
- Extracts JWT from `Authorization: Bearer <token>` header OR cookie (configurable name)
- Validates token signature, algorithm, expiration, not-before
- On success: Injects claims into request context, calls `c.Next()`
- On failure: Calls `c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized", "reason": reason})`
- Logs SecurityEvent on all outcomes (if logger configured)

**Error Responses** (HTTP 401 JSON):
```json
{
  "error": "unauthorized",
  "reason": "missing_token" | "expired" | "invalid_signature" | "malformed" | "algorithm_mismatch"
}
```

---

#### gRPC Unary Interceptor

```go
// UnaryServerInterceptor creates a gRPC unary server interceptor for JWT authentication.
// Validates JWT from incoming metadata (key: "authorization").
// On success: injects claims into context and calls handler
// On failure: returns codes.Unauthenticated error
//
// Example:
//   cfg := NewConfig(WithRS256(publicKey))
//   grpc.NewServer(grpc.UnaryInterceptor(UnaryServerInterceptor(cfg)))
func UnaryServerInterceptor(config *Config) grpc.UnaryServerInterceptor
```

**Input**:
- `config *Config`: Immutable configuration (MUST be non-nil)

**Output**:
- `grpc.UnaryServerInterceptor`: Interceptor function for gRPC server

**Behavior**:
- Extracts JWT from gRPC metadata key "authorization" (value: "Bearer <token>")
- Validates token signature, algorithm, expiration, not-before
- On success: Injects claims into context, calls `handler(ctx, req)`
- On failure: Returns `status.Error(codes.Unauthenticated, reason)`
- Logs SecurityEvent on all outcomes (if logger configured)

**Error Responses** (gRPC Status):
```
code: codes.Unauthenticated (16)
message: "missing_token" | "expired" | "invalid_signature" | "malformed" | "algorithm_mismatch"
```

---

### Configuration Construction

#### NewConfig

```go
// NewConfig creates a new immutable Config with functional options.
// Returns error if configuration is invalid (e.g., unsupported algorithm, nil signing key).
// Config is frozen after construction - no mutations allowed.
//
// Example:
//   cfg, err := NewConfig(
//       WithHS256(secret),
//       WithClockSkew(30*time.Second),
//       WithLogger(slog.Default()),
//   )
func NewConfig(opts ...ConfigOption) (*Config, error)
```

**Input**:
- `opts ...ConfigOption`: Variadic functional options

**Output**:
- `*Config`: Immutable configuration
- `error`: Validation error if config invalid

**Validation Errors**:
- `"algorithm must be HS256 or RS256"`: Algorithm not set or unsupported
- `"signing key required"`: SigningKey is nil
- `"HS256 key must be at least 32 bytes"`: HMAC key too short
- `"clock skew must be non-negative"`: ClockSkewLeeway < 0

---

### Functional Options

#### WithHS256

```go
// WithHS256 configures HS256 (HMAC-SHA256) algorithm with secret key.
// Key should be at least 32 bytes (256 bits) for security.
//
// Example:
//   WithHS256([]byte("your-256-bit-secret"))
func WithHS256(secret []byte) ConfigOption
```

**Input**: `secret []byte` (MUST be ≥ 32 bytes)

**Validation**: Key length checked in `NewConfig()`, returns error if < 32 bytes

---

#### WithRS256

```go
// WithRS256 configures RS256 (RSA-SHA256) algorithm with RSA public key.
// Key should be parsed from PEM format using ParseRSAPublicKeyFromPEM helper.
//
// Example:
//   pubKey, _ := ParseRSAPublicKeyFromPEM(pemBytes)
//   WithRS256(pubKey)
func WithRS256(publicKey *rsa.PublicKey) ConfigOption
```

**Input**: `publicKey *rsa.PublicKey` (MUST be non-nil)

**Validation**: Key presence checked in `NewConfig()`, returns error if nil

---

#### WithClockSkew

```go
// WithClockSkew sets tolerance for exp/nbf claim validation.
// Default: 60 seconds. Use 0 for strict validation (not recommended).
//
// Example:
//   WithClockSkew(30 * time.Second)
func WithClockSkew(leeway time.Duration) ConfigOption
```

**Input**: `leeway time.Duration` (MUST be ≥ 0)

**Default**: 60 seconds if not specified

**Validation**: Returns error if < 0

---

#### WithCookie

```go
// WithCookie enables JWT extraction from cookie in addition to Authorization header.
// If both present, Authorization header takes precedence.
// If name is empty, cookie extraction is disabled (header-only mode).
//
// Example:
//   WithCookie("auth_token")
func WithCookie(name string) ConfigOption
```

**Input**: `name string` (empty = disabled)

**Default**: Header-only mode (no cookie extraction)

---

#### WithLogger

```go
// WithLogger configures structured logging for authentication events.
// Pass nil to disable logging (not recommended for production).
//
// Example:
//   WithLogger(slog.Default())
func WithLogger(logger *slog.Logger) ConfigOption
```

**Input**: `logger *slog.Logger` (nil = silent mode)

**Default**: nil (no logging) if not specified

---

#### WithRequiredClaims

```go
// WithRequiredClaims specifies claims that MUST be present in JWT.
// Validation fails if any required claim is missing.
//
// Example:
//   WithRequiredClaims("sub", "email")
func WithRequiredClaims(claims ...string) ConfigOption
```

**Input**: `claims ...string` (claim names)

**Default**: No required claims beyond standard (exp)

---

### Context Helpers

#### GetClaims

```go
// GetClaims retrieves validated JWT claims from request context.
// Returns (nil, false) if no claims present (authentication failed or not executed).
//
// Example:
//   claims, ok := jwtauth.GetClaims(c.Request.Context())
//   if ok {
//       userID := claims.Subject
//   }
func GetClaims(ctx context.Context) (*Claims, bool)
```

**Input**: `ctx context.Context`

**Output**:
- `*Claims`: Validated claims (nil if not present)
- `bool`: true if claims present, false otherwise

**Usage**: Call in downstream handlers after middleware

---

#### MustGetClaims

```go
// MustGetClaims retrieves claims or panics if not present.
// Use only when authentication middleware guarantees claims presence.
//
// Example:
//   claims := jwtauth.MustGetClaims(ctx)
func MustGetClaims(ctx context.Context) *Claims
```

**Input**: `ctx context.Context`

**Output**: `*Claims` (panics if not present)

**Warning**: Only use in handlers protected by JWT middleware

---

### Key Parsing Helpers

#### ParseRSAPublicKeyFromPEM

```go
// ParseRSAPublicKeyFromPEM parses RSA public key from PEM-encoded bytes.
// Supports both PKCS#1 and PKIX formats.
//
// Example:
//   pubKey, err := jwtauth.ParseRSAPublicKeyFromPEM(pemBytes)
//   cfg, _ := NewConfig(WithRS256(pubKey))
func ParseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error)
```

**Input**: `pemBytes []byte` (PEM-encoded public key)

**Output**:
- `*rsa.PublicKey`: Parsed key
- `error`: Parse error if invalid PEM or not RSA key

**Supported Formats**:
- PKCS#1: `-----BEGIN RSA PUBLIC KEY-----`
- PKIX: `-----BEGIN PUBLIC KEY-----`

---

## Public Types

### Config

```go
// Config is immutable middleware configuration.
// Construct via NewConfig() with functional options.
type Config struct {
    // Unexported fields - access via methods only
}
```

**Methods**: None (immutable, no getters needed)

**Construction**: `NewConfig(...ConfigOption)` only

---

### Claims

```go
// Claims represents validated JWT claims injected into context.
type Claims struct {
    Subject   string                 // User identifier (sub)
    Issuer    string                 // Token issuer (iss)
    Audience  string                 // Intended audience (aud)
    ExpiresAt time.Time              // Expiration time (exp)
    NotBefore time.Time              // Not-before time (nbf)
    IssuedAt  time.Time              // Issue time (iat)
    JWTID     string                 // JWT ID (jti)
    Custom    map[string]interface{} // Application-specific claims
}
```

**All fields exported** (read-only by convention)

**Access**: Retrieved via `GetClaims(ctx)` or `MustGetClaims(ctx)`

---

### ConfigOption

```go
// ConfigOption is a functional option for NewConfig.
type ConfigOption func(*configBuilder) error
```

**Usage**: Passed to `NewConfig()`, not constructed directly

**Provided Options**: `WithHS256`, `WithRS256`, `WithClockSkew`, `WithCookie`, `WithLogger`, `WithRequiredClaims`

---

### ValidationError (Implements error interface)

```go
// ValidationError represents a JWT validation failure.
// Implements error interface with structured error codes.
type ValidationError struct {
    Code    string // Error code (EXPIRED, INVALID_SIGNATURE, etc.)
    Message string // Human-readable message
    // Internal field not exported
}

// Error returns formatted error string.
func (e *ValidationError) Error() string

// Unwrap returns underlying error (for errors.Is/As).
func (e *ValidationError) Unwrap() error
```

**Error Codes** (string constants):
- `"EXPIRED"`: Token expired (exp claim)
- `"INVALID_SIGNATURE"`: Signature verification failed
- `"MISSING_TOKEN"`: No JWT found in request
- `"MALFORMED"`: JWT syntax invalid
- `"ALGORITHM_MISMATCH"`: Algorithm doesn't match config
- `"NONE_ALGORITHM"`: "none" algorithm rejected
- `"CONFIG_ERROR"`: Invalid configuration

---

## Usage Examples

### Gin HTTP Server

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/user/project/jwtauth"
    "log/slog"
)

func main() {
    // Configure middleware
    cfg, err := jwtauth.NewConfig(
        jwtauth.WithHS256([]byte("your-256-bit-secret")),
        jwtauth.WithCookie("auth_token"),
        jwtauth.WithClockSkew(30*time.Second),
        jwtauth.WithLogger(slog.Default()),
    )
    if err != nil {
        panic(err)
    }

    // Create Gin router with JWT middleware
    r := gin.Default()
    r.Use(jwtauth.JWTAuth(cfg))

    // Protected endpoint
    r.GET("/profile", func(c *gin.Context) {
        claims, ok := jwtauth.GetClaims(c.Request.Context())
        if !ok {
            c.JSON(500, gin.H{"error": "claims not found"})
            return
        }
        c.JSON(200, gin.H{"user_id": claims.Subject})
    })

    r.Run(":8080")
}
```

---

### gRPC Server

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/user/project/jwtauth"
    "crypto/rsa"
)

func main() {
    // Parse RSA public key
    pubKey, err := jwtauth.ParseRSAPublicKeyFromPEM(pemBytes)
    if err != nil {
        panic(err)
    }

    // Configure middleware
    cfg, err := jwtauth.NewConfig(
        jwtauth.WithRS256(pubKey),
        jwtauth.WithLogger(slog.Default()),
    )
    if err != nil {
        panic(err)
    }

    // Create gRPC server with JWT interceptor
    srv := grpc.NewServer(
        grpc.UnaryInterceptor(jwtauth.UnaryServerInterceptor(cfg)),
    )

    // Register service
    pb.RegisterMyServiceServer(srv, &myService{})

    // In service handler:
    func (s *myService) GetUser(ctx context.Context, req *pb.Request) (*pb.Response, error) {
        claims := jwtauth.MustGetClaims(ctx)
        // Use claims.Subject, claims.Custom, etc.
    }
}
```

---

### RS256 with PEM File

```go
package main

import (
    "os"
    "github.com/user/project/jwtauth"
)

func main() {
    // Read RSA public key from file
    pemBytes, err := os.ReadFile("public_key.pem")
    if err != nil {
        panic(err)
    }

    // Parse key
    pubKey, err := jwtauth.ParseRSAPublicKeyFromPEM(pemBytes)
    if err != nil {
        panic(err)
    }

    // Configure with RS256
    cfg, err := jwtauth.NewConfig(
        jwtauth.WithRS256(pubKey),
    )
    if err != nil {
        panic(err)
    }

    // Use cfg with JWTAuth() or UnaryServerInterceptor()
}
```

---

## API Stability Guarantees

### v1.x Compatibility Promise

Once v1.0.0 is released:

✅ **Stable** (will not break in v1.x):
- Function signatures: `JWTAuth()`, `UnaryServerInterceptor()`, `NewConfig()`, `GetClaims()`
- Type definitions: `Config`, `Claims`, `ValidationError`
- Functional options: `WithHS256()`, `WithRS256()`, etc.
- Error codes: `"EXPIRED"`, `"INVALID_SIGNATURE"`, etc.

⚠️ **May Add** (backward-compatible additions in v1.x):
- New functional options (e.g., `WithES256()`)
- New Claims fields (e.g., `Scope string`)
- New helper functions (e.g., `GetClaimsOrDefault()`)

❌ **Breaking** (require v2.0.0):
- Removing exported functions or types
- Changing function signatures
- Removing or renaming Claims fields
- Changing error codes

---

## Versioning & Deprecation

### Semantic Versioning

- **v0.x.x**: Pre-release, API may change without notice
- **v1.0.0**: Stable release, API frozen per compatibility promise
- **v1.x.x**: Backward-compatible additions and bug fixes
- **v2.0.0**: Breaking changes (requires import path update: `/v2`)

### Deprecation Policy

- Deprecated features marked with `// Deprecated:` comment
- Deprecated features retained for 2 minor versions
- Migration path documented in CHANGELOG.md
- Example: Function deprecated in v1.5.0, removed in v1.7.0 (not v1.6.0)

---

## Security Considerations

### API Safety

✅ **Immutable Config**: No setters prevent runtime tampering
✅ **Typed Context Keys**: Unexported keys prevent collisions
✅ **Error Sanitization**: ValidationError messages safe for client exposure
✅ **No Panic**: All functions return errors (except `MustGetClaims`)
✅ **Constant-Time**: Signature validation via golang-jwt/jwt (crypto/subtle)

### Misuse Prevention

⚠️ **Claims.Custom**: Applications MUST validate custom claims (middleware does not)
⚠️ **MustGetClaims**: Only use in handlers guaranteed to have claims (protected by middleware)
⚠️ **Concurrent Config**: Config is read-only safe, but constructing multiple configs with same secret is discouraged

---

## Testing Support

### Test Helpers (Provided)

```go
// MockClaims creates test claims with sensible defaults.
func MockClaims(subject string, expiresIn time.Duration) *Claims

// WithMockClaims injects test claims into context (for testing handlers).
func WithMockClaims(ctx context.Context, claims *Claims) context.Context
```

### Test Token Generation (Not Provided)

Applications should use golang-jwt/jwt directly to generate test tokens:

```go
import "github.com/golang-jwt/jwt/v5"

func createTestToken(secret []byte) string {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": "user123",
        "exp": time.Now().Add(time.Hour).Unix(),
    })
    tokenString, _ := token.SignedString(secret)
    return tokenString
}
```

---

## API Surface Summary

**Total Public Functions**: 13
- Middleware: 2 (JWTAuth, UnaryServerInterceptor)
- Config: 1 (NewConfig)
- Options: 6 (WithHS256, WithRS256, WithClockSkew, WithCookie, WithLogger, WithRequiredClaims)
- Context: 2 (GetClaims, MustGetClaims)
- Helpers: 1 (ParseRSAPublicKeyFromPEM)
- Test: 2 (MockClaims, WithMockClaims - optional)

**Total Public Types**: 4
- Config (opaque)
- Claims (struct with exported fields)
- ConfigOption (function type)
- ValidationError (error type with exported fields)

**Dependencies Exposed in API**:
- `github.com/gin-gonic/gin` (JWTAuth return type only)
- `google.golang.org/grpc` (UnaryServerInterceptor return type only)
- `crypto/rsa` (WithRS256 parameter only)
- `log/slog` (WithLogger parameter only)

**Zero External Dependencies in Core API** (Config, Claims, ValidationError fully self-contained)
