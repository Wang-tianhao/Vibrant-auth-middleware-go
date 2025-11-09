# Configuration API Contract

**Feature**: Dual Algorithm JWT Validation
**Version**: 2.0.0
**Language**: Go
**Package**: `github.com/user/vibrant-auth-middleware-go/jwtauth`

---

## Overview

This document defines the public API contract for configuring the JWT authentication middleware with support for multiple algorithms (HS256 and RS256).

**Backward Compatibility**: All existing single-algorithm configurations remain valid. The dual-algorithm feature is **additive** and does not break existing code.

---

## Configuration API

### NewConfig

Creates a new immutable middleware configuration.

**Signature**:
```go
func NewConfig(opts ...ConfigOption) (*Config, error)
```

**Parameters**:
- `opts`: Variable number of `ConfigOption` functions (functional options pattern)

**Returns**:
- `*Config`: Immutable configuration object (on success)
- `error`: `*ValidationError` with code `CONFIG_ERROR` if validation fails

**Validation Rules**:
1. At least one algorithm MUST be configured (via `WithHS256` or `WithRS256`)
2. `none` algorithm MUST NOT be configured
3. All algorithm-specific keys MUST be valid (see algorithm options below)

**Example**:
```go
// Single algorithm (backward compatible)
cfg, err := jwtauth.NewConfig(jwtauth.WithHS256(secret))

// Dual algorithm (NEW in v2.0)
cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(hs256Secret),
    jwtauth.WithRS256(rs256PublicKey),
)
```

---

## Algorithm Configuration Options

### WithHS256

Configures HMAC-SHA256 (symmetric) algorithm validation.

**Signature**:
```go
func WithHS256(secret []byte) ConfigOption
```

**Parameters**:
- `secret`: HMAC secret key (MUST be at least 32 bytes / 256 bits)

**Returns**:
- `ConfigOption`: Functional option to be passed to `NewConfig`

**Validation**:
- `secret` MUST NOT be nil
- `len(secret) >= 32` (enforces 256-bit minimum key size)

**Error**:
- Returns error during `NewConfig` if validation fails

**Example**:
```go
secret := []byte("your-256-bit-secret-key-min-32-bytes")
cfg, err := jwtauth.NewConfig(jwtauth.WithHS256(secret))
```

**Security Notes**:
- Secret MUST be cryptographically random (use `crypto/rand`)
- Secret SHOULD be at least 32 bytes (256 bits) per JWT best practices
- Secret MUST be stored securely (environment variable, secret manager, NOT hardcoded)

---

### WithRS256

Configures RSA-SHA256 (asymmetric) algorithm validation.

**Signature**:
```go
func WithRS256(publicKey *rsa.PublicKey) ConfigOption
```

**Parameters**:
- `publicKey`: RSA public key for signature verification (MUST NOT be nil)

**Returns**:
- `ConfigOption`: Functional option to be passed to `NewConfig`

**Validation**:
- `publicKey` MUST NOT be nil
- Key size SHOULD be at least 2048 bits (not enforced, but recommended)

**Error**:
- Returns error during `NewConfig` if `publicKey` is nil

**Example**:
```go
publicKey, err := jwtauth.LoadRSAPublicKey("path/to/public.pem")
if err != nil {
    log.Fatal(err)
}

cfg, err := jwtauth.NewConfig(jwtauth.WithRS256(publicKey))
```

**Security Notes**:
- Public key SHOULD be obtained from trusted source (JWKS endpoint, secure distribution)
- Key size SHOULD be at least 2048 bits per current security standards
- Private key MUST NEVER be provided to middleware (only public key for verification)

---

## Other Configuration Options

### WithClockSkew

Sets the clock skew tolerance for `exp` (expiration) and `nbf` (not-before) validation.

**Signature**:
```go
func WithClockSkew(skew time.Duration) ConfigOption
```

**Parameters**:
- `skew`: Maximum allowed time difference between server and token issuer clocks

**Default**: 60 seconds

**Validation**:
- `skew` MUST be non-negative (`>= 0`)

**Example**:
```go
cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),
    jwtauth.WithClockSkew(30 * time.Second), // 30 seconds tolerance
)
```

---

### WithCookie

Enables token extraction from an HTTP cookie (in addition to `Authorization` header).

**Signature**:
```go
func WithCookie(cookieName string) ConfigOption
```

**Parameters**:
- `cookieName`: Name of the cookie containing the JWT token

**Default**: Token only extracted from `Authorization: Bearer <token>` header

**Example**:
```go
cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),
    jwtauth.WithCookie("auth_token"), // Also check "auth_token" cookie
)
```

---

### WithLogger

Configures structured logging for security events.

**Signature**:
```go
func WithLogger(logger *slog.Logger) ConfigOption
```

**Parameters**:
- `logger`: Standard library `log/slog.Logger` instance

**Default**: No logging (security events are silently dropped)

**Example**:
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),
    jwtauth.WithLogger(logger),
)
```

**Logged SecurityEvent Fields** (NEW in v2.0: includes `algorithm`):
```json
{
  "event_type": "success",
  "timestamp": "2025-11-09T10:30:00Z",
  "request_id": "abc-123",
  "user_id": "user-456",
  "algorithm": "HS256",
  "latency_ms": 0.8
}
```

---

### WithRequiredClaims

Specifies custom claim names that MUST be present in the JWT payload.

**Signature**:
```go
func WithRequiredClaims(claims ...string) ConfigOption
```

**Parameters**:
- `claims`: Variable number of claim names (e.g., `"email"`, `"role"`)

**Default**: No required claims (only validates standard claims like `exp`, `nbf`)

**Example**:
```go
cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),
    jwtauth.WithRequiredClaims("sub", "email", "role"),
)
```

---

## Config Methods (Public API)

### AvailableAlgorithms (NEW in v2.0)

Returns a sorted list of configured algorithm names.

**Signature**:
```go
func (c *Config) AvailableAlgorithms() []string
```

**Returns**:
- `[]string`: Sorted list of algorithm names (e.g., `["HS256", "RS256"]`)

**Example**:
```go
algs := cfg.AvailableAlgorithms()
fmt.Println(algs) // Output: [HS256 RS256]
```

**Use Case**: Generating error messages that list supported algorithms

---

### Algorithm (Deprecated)

Returns the first configured algorithm name (for backward compatibility).

**Signature**:
```go
func (c *Config) Algorithm() string
```

**Returns**:
- `string`: First algorithm name in sorted order

**Deprecation Notice**: Use `AvailableAlgorithms()` for dual-algorithm configs. This method retained for backward compatibility with single-algorithm code.

**Example**:
```go
alg := cfg.Algorithm()
fmt.Println(alg) // Output: "HS256" (first in sorted order)
```

---

### SigningKey (Deprecated)

Returns the signing key for the first configured algorithm (for backward compatibility).

**Signature**:
```go
func (c *Config) SigningKey() interface{}
```

**Returns**:
- `interface{}`: Signing key (`[]byte` for HS256, `*rsa.PublicKey` for RS256)

**Deprecation Notice**: Retained for backward compatibility. Internal use only.

---

## Usage Examples

### Example 1: Dual-Algorithm Configuration (Primary Use Case)

**Scenario**: Microservice accepts tokens from two issuers - internal service (HS256) and external OAuth provider (RS256).

```go
package main

import (
    "crypto/rsa"
    "log"
    "log/slog"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/user/vibrant-auth-middleware-go/jwtauth"
)

func main() {
    // Load HS256 secret from environment
    hs256Secret := []byte(os.Getenv("JWT_HS256_SECRET"))

    // Load RS256 public key from file
    rs256PublicKey, err := jwtauth.LoadRSAPublicKey("oauth_provider_public.pem")
    if err != nil {
        log.Fatal(err)
    }

    // Configure structured logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Create dual-algorithm config
    cfg, err := jwtauth.NewConfig(
        jwtauth.WithHS256(hs256Secret),        // Accept HS256 tokens
        jwtauth.WithRS256(rs256PublicKey),     // Accept RS256 tokens
        jwtauth.WithClockSkew(60 * time.Second),
        jwtauth.WithLogger(logger),
        jwtauth.WithRequiredClaims("sub"),
    )
    if err != nil {
        log.Fatalf("Config error: %v", err)
    }

    // Apply middleware to Gin router
    router := gin.Default()
    router.Use(jwtauth.JWTAuth(cfg))

    router.GET("/protected", func(c *gin.Context) {
        claims := jwtauth.GetClaims(c.Request.Context())
        c.JSON(200, gin.H{"user_id": claims.Subject})
    })

    router.Run(":8080")
}
```

---

### Example 2: Backward Compatible Single-Algorithm Configuration

**Scenario**: Existing code using HS256 only - no changes required.

```go
package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/user/vibrant-auth-middleware-go/jwtauth"
)

func main() {
    secret := []byte(os.Getenv("JWT_SECRET"))

    // Existing single-algorithm config - still works!
    cfg, err := jwtauth.NewConfig(jwtauth.WithHS256(secret))
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    router.Use(jwtauth.JWTAuth(cfg))

    router.GET("/protected", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "authenticated"})
    })

    router.Run(":8080")
}
```

---

### Example 3: RS256 Only Configuration

**Scenario**: Public API using only asymmetric cryptography (no shared secrets).

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "github.com/user/vibrant-auth-middleware-go/jwtauth"
)

func main() {
    publicKey, err := jwtauth.LoadRSAPublicKey("public.pem")
    if err != nil {
        log.Fatal(err)
    }

    cfg, err := jwtauth.NewConfig(jwtauth.WithRS256(publicKey))
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    router.Use(jwtauth.JWTAuth(cfg))

    router.GET("/public-api", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "RS256 validated"})
    })

    router.Run(":8080")
}
```

---

### Example 4: Error Handling

**Scenario**: Handling configuration errors during startup.

```go
package main

import (
    "fmt"
    "log"

    "github.com/user/vibrant-auth-middleware-go/jwtauth"
)

func main() {
    // Invalid config: secret too short
    shortSecret := []byte("short")
    cfg, err := jwtauth.NewConfig(jwtauth.WithHS256(shortSecret))

    if err != nil {
        if valErr, ok := err.(*jwtauth.ValidationError); ok {
            fmt.Printf("Config error [%s]: %s\n", valErr.Code, valErr.Message)
            // Output: Config error [CONFIG_ERROR]: HS256 secret must be at least 32 bytes
        }
        log.Fatal(err)
    }

    _ = cfg
}
```

---

## Algorithm Routing Behavior

### Token Validation Flow (NEW in v2.0)

When a token is received, the middleware:

1. **Extracts** `alg` field from JWT header
2. **Rejects** if `alg` is `"none"`, `"None"`, or `"NONE"` → `NONE_ALGORITHM` error
3. **Looks up** algorithm in configured validators map
4. **Rejects** if algorithm not found → `UNSUPPORTED_ALGORITHM` error (lists available algorithms)
5. **Verifies** signature using the appropriate validator (HS256 or RS256)
6. **Validates** claims (exp, nbf, required claims)
7. **Logs** security event with algorithm name

### Example: Token with Unsupported Algorithm

**Request**:
```
GET /protected HTTP/1.1
Authorization: Bearer eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Token Header** (decoded):
```json
{
  "alg": "ES256",
  "typ": "JWT"
}
```

**Middleware Response** (assuming only HS256 and RS256 configured):
```json
{
  "error": "unauthorized",
  "reason": "UNSUPPORTED_ALGORITHM"
}
```

**Log Entry**:
```json
{
  "event_type": "failure",
  "timestamp": "2025-11-09T10:30:00Z",
  "request_id": "abc-123",
  "algorithm": "ES256",
  "failure_reason": "UNSUPPORTED_ALGORITHM",
  "latency_ms": 0.3
}
```

---

## Backward Compatibility Guarantees

### What Remains Unchanged (v1.x → v2.0)

✅ Middleware signature: `JWTAuth(cfg *Config) gin.HandlerFunc`
✅ Single-algorithm configuration: `WithHS256` and `WithRS256` work as before
✅ Error response structure: `{error, reason}` fields unchanged
✅ Existing error codes: `EXPIRED`, `INVALID_SIGNATURE`, etc. retain same semantics
✅ Context injection: `GetClaims(ctx)` API unchanged
✅ Configuration options: `WithClockSkew`, `WithLogger`, etc. unchanged

### What's New in v2.0

🆕 Multiple algorithm support: Can configure both HS256 and RS256 simultaneously
🆕 New error codes: `UNSUPPORTED_ALGORITHM`, `MALFORMED_ALGORITHM_HEADER`
🆕 Enhanced logging: `SecurityEvent` includes `Algorithm` field
🆕 New method: `AvailableAlgorithms()` returns list of configured algorithms

### Migration Path

**No migration required** for existing single-algorithm deployments. To adopt dual-algorithm support:

1. Update `NewConfig` call to include both `WithHS256` and `WithRS256` options
2. Optionally handle new `UNSUPPORTED_ALGORITHM` error code in client code
3. Optionally use `AvailableAlgorithms()` method for introspection

---

## Thread Safety

All `Config` methods are **safe for concurrent use** by multiple goroutines. The `Config` struct is **immutable** after creation by `NewConfig()`.

---

## Performance Characteristics

- **Algorithm routing overhead**: <10 microseconds (O(1) map lookup)
- **Token validation latency**: <1ms p99 (dominated by cryptographic signature verification)
- **Memory allocations**: 0 additional allocations for algorithm routing (compared to single-algorithm case)

---

## Security Considerations

### Algorithm Confusion Prevention

The middleware prevents algorithm confusion attacks (CVE-2015-9235) by:

1. Verifying `token.Header["alg"]` matches the configured algorithm before using validator
2. Checking that `token.Method` type matches expected signing method (HMAC vs RSA)
3. Explicitly rejecting `none` algorithm regardless of configuration

**Example**: If token header says `alg: HS256` but token was actually signed with RSA private key, validation fails with `INVALID_SIGNATURE` (signature verification fails).

### Information Disclosure

Error messages listing available algorithms (e.g., "available: HS256, RS256") are safe because:

- Algorithm availability is public information (equivalent to JWKS `alg` field)
- No secret keys or sensitive configuration is exposed
- Helps developers debug integration issues quickly

### Timing Attacks

Algorithm routing uses standard Go map lookup (not constant-time). This is acceptable because:

- Algorithm names are public information (not secrets)
- Timing differences (<1ns) are negligible compared to network latency
- Industry-standard practice (Auth0, Spring Security use similar patterns)

---

## Versioning

**Current Version**: 2.0.0

**Semantic Versioning**:
- **MAJOR** (2.x.x): Breaking API changes (none in this release)
- **MINOR** (x.1.x): New features, backward compatible (dual-algorithm support)
- **PATCH** (x.x.1): Bug fixes, no API changes

**Compatibility**: v2.0.0 is **backward compatible** with v1.x configurations.
