# Vibrant Auth Middleware (Go)

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-98%2B%20passing-brightgreen.svg)](jwtauth/)

High-performance JWT authentication middleware for Go web frameworks with dual-algorithm support.

## Features

- ✅ **Dual-Algorithm Support** (v2.0+): Accept both HS256 and RS256 tokens simultaneously
- ✅ **Framework Support**: Gin and gRPC interceptors included
- ✅ **Zero-Allocation Routing**: <10ns algorithm routing overhead
- ✅ **Structured Logging**: Built-in security event logging with algorithm metadata
- ✅ **Type-Safe**: Strong typing with custom claims support
- ✅ **Production-Ready**: <1ms p99 latency, race-condition tested
- ✅ **100% Backward Compatible**: No breaking changes from v1.x

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration-options)
- [Usage Examples](#usage-examples)
- [Error Handling](#error-handling)
- [Performance](#performance)
- [Security](#security)
- [Testing](#testing)
- [Migration Guide](#migration-from-v1x-to-v20)
- [Contributing](#contributing)

## Quick Start

### Single Algorithm (HS256)

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/Wang-tianhao/Vibrant-auth-middleware-go/jwtauth"
)

func main() {
    secret := []byte("your-256-bit-secret-min-32-bytes!")
    cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

    router := gin.Default()

    // Apply JWT authentication to protected routes
    authorized := router.Group("/api")
    authorized.Use(jwtauth.JWTAuth(cfg))
    {
        authorized.GET("/profile", getProfile)
    }

    router.Run(":8080")
}

func getProfile(c *gin.Context) {
    // Get validated claims from context
    claims, _ := jwtauth.GetClaims(c.Request.Context())

    c.JSON(200, gin.H{
        "user_id": claims.Subject,
        "email": claims.Custom["email"],
    })
}
```

### Dual Algorithm (HS256 + RS256) - v2.0+

Accept tokens from multiple issuers using different signing methods:

```go
import (
    "crypto/rsa"
    "github.com/Wang-tianhao/Vibrant-auth-middleware-go/jwtauth"
)

func main() {
    // HS256 for internal tokens
    hs256Secret := []byte("internal-secret-min-32-bytes!")

    // RS256 for external partner tokens
    rs256PublicKey := loadRSAPublicKey() // *rsa.PublicKey

    cfg, _ := jwtauth.NewConfig(
        jwtauth.WithHS256(hs256Secret),    // Accept internal HS256 tokens
        jwtauth.WithRS256(rs256PublicKey), // Accept external RS256 tokens
    )

    // Middleware now accepts BOTH HS256 and RS256 tokens!
    router.Use(jwtauth.JWTAuth(cfg))

    // Check which algorithms are configured
    log.Printf("Configured algorithms: %v", cfg.AvailableAlgorithms())
    // Output: Configured algorithms: [HS256 RS256]
}
```

## Installation

```bash
go get github.com/Wang-tianhao/Vibrant-auth-middleware-go
```

**Requirements:**
- Go 1.23+ (recommended: 1.24+)
- github.com/golang-jwt/jwt/v5 v5.3.0+
- github.com/gin-gonic/gin v1.11.0+ (for Gin middleware)
- google.golang.org/grpc v1.76.0+ (for gRPC interceptors)

## Configuration Options

### All Available Options

```go
cfg, err := jwtauth.NewConfig(
    // Algorithm support (at least one required)
    jwtauth.WithHS256(secret),              // Add HS256 (HMAC-SHA256) support
    jwtauth.WithRS256(publicKey),           // Add RS256 (RSA-SHA256) support

    // Optional: Token extraction
    jwtauth.WithCookie("auth_token"),       // Also check cookies for token

    // Optional: Validation settings
    jwtauth.WithClockSkew(30*time.Second),  // Clock skew tolerance (default: 0)
    jwtauth.WithRequiredClaims("sub", "iss"), // Require specific claims

    // Optional: Logging
    jwtauth.WithLogger(logger),             // Structured logging (slog.Logger)
)
```

### Configuration Methods

| Method | Description | Example |
|--------|-------------|---------|
| `WithHS256(secret []byte)` | Add HS256 algorithm support | `WithHS256([]byte("secret"))` |
| `WithRS256(publicKey *rsa.PublicKey)` | Add RS256 algorithm support | `WithRS256(pubKey)` |
| `WithCookie(name string)` | Check cookies for token | `WithCookie("auth_token")` |
| `WithClockSkew(duration time.Duration)` | Set clock skew tolerance | `WithClockSkew(30*time.Second)` |
| `WithRequiredClaims(claims ...string)` | Require specific claims | `WithRequiredClaims("sub", "iss")` |
| `WithLogger(logger *slog.Logger)` | Enable structured logging | `WithLogger(slog.Default())` |

## Usage Examples

### Gin HTTP Server

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/Wang-tianhao/Vibrant-auth-middleware-go/jwtauth"
    "time"
)

func main() {
    secret := []byte("your-secret-key-min-32-bytes-required!")
    cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

    router := gin.Default()

    // Public routes
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })

    // Protected routes
    authorized := router.Group("/api")
    authorized.Use(jwtauth.JWTAuth(cfg))
    {
        authorized.GET("/profile", getProfile)
        authorized.GET("/data", getData)
    }

    router.Run(":8080")
}

func getProfile(c *gin.Context) {
    claims, ok := jwtauth.GetClaims(c.Request.Context())
    if !ok {
        c.JSON(500, gin.H{"error": "claims not found"})
        return
    }

    c.JSON(200, gin.H{
        "user_id": claims.Subject,
        "issuer": claims.Issuer,
        "expires_at": claims.ExpiresAt.Format(time.RFC3339),
    })
}
```

### gRPC Interceptor

```go
import (
    "google.golang.org/grpc"
    "github.com/Wang-tianhao/Vibrant-auth-middleware-go/jwtauth"
)

func main() {
    secret := []byte("your-secret-key-min-32-bytes!")
    cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

    server := grpc.NewServer(
        grpc.UnaryInterceptor(jwtauth.UnaryServerInterceptor(cfg)),
    )

    // Register your services...
}
```

### Accessing Claims

```go
// In your handler
claims, ok := jwtauth.GetClaims(ctx)
if !ok {
    return errors.New("unauthorized")
}

// Standard claims
userID := claims.Subject       // "sub" claim
issuer := claims.Issuer        // "iss" claim
audience := claims.Audience    // "aud" claim
expiresAt := claims.ExpiresAt  // "exp" claim
notBefore := claims.NotBefore  // "nbf" claim
issuedAt := claims.IssuedAt    // "iat" claim
jwtID := claims.JWTID          // "jti" claim

// Custom claims
email := claims.Custom["email"].(string)
role := claims.Custom["role"].(string)
```

## Error Handling

The middleware returns clear, distinct error codes with helpful messages:

### Error Response Format

```json
{
  "error": "unauthorized",
  "reason": "UNSUPPORTED_ALGORITHM",
  "message": "algorithm ES256 not supported (available: HS256, RS256)"
}
```

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `UNSUPPORTED_ALGORITHM` | Token uses an algorithm not configured | 401 |
| `INVALID_SIGNATURE` | Signature verification failed | 401 |
| `EXPIRED` | Token has expired | 401 |
| `MISSING_TOKEN` | No token provided in request | 401 |
| `MALFORMED` | Token structure is invalid | 401 |
| `MALFORMED_ALGORITHM_HEADER` | Algorithm header is malformed | 401 |
| `NONE_ALGORITHM` | "none" algorithm explicitly rejected | 401 |

### Example: Handling Different Error Types

```go
// The middleware automatically returns 401 with error details
// Your frontend can handle specific error codes:

if (response.reason === "EXPIRED") {
    // Redirect to login with "session expired" message
} else if (response.reason === "UNSUPPORTED_ALGORITHM") {
    // Log security event - unexpected token algorithm
    console.error("Available algorithms:", response.message);
}
```

## Performance

Benchmarked on Apple M4 Pro:

| Operation | Latency | Allocations | Target |
|-----------|---------|-------------|--------|
| **Algorithm Routing** | 8.6 ns | 0 B/op | <10 μs ✅ |
| **HS256 Validation** | 1.7 μs | 2,648 B/op | <1 ms ✅ |
| **RS256 Validation** | 22 μs | 3,896 B/op | <1 ms ✅ |
| **Single vs Dual Config** | No difference | Same | No regression ✅ |

### Run Benchmarks

```bash
go test -bench=. -benchmem ./jwtauth/
```

See `jwtauth/benchmark_test.go` for detailed benchmark suite.

## Security

### Security Features

- ✅ **Algorithm Confusion Prevention**: Explicit validation prevents algorithm substitution attacks
- ✅ **"none" Algorithm Rejection**: All variants (none, None, NONE) are explicitly rejected
- ✅ **Case-Sensitive Matching**: Algorithm names are case-sensitive per RFC 7519
- ✅ **Comprehensive Testing**: 98+ tests including security attack scenarios
- ✅ **Audit Logging**: All authentication events logged with algorithm metadata
- ✅ **No Secret Leakage**: Tokens and secrets never logged

### Security Audit

```bash
# Run security scanner
gosec ./jwtauth/...

# Result: 0 issues found ✅
```

### Reported Vulnerabilities

None. If you discover a security issue, please email: tianhao.wang@vibrant-america.com

## Testing

### Run Tests

```bash
# All tests
go test ./jwtauth/...

# With coverage
go test -cover ./jwtauth/...

# With race detector
go test -race ./jwtauth/...

# Verbose output
go test -v ./jwtauth/...

# Specific test
go test -run TestDualAlgorithm ./jwtauth/...
```

### Test Coverage

- **Overall**: 62.2%
- **Security-Critical Code**: >85%
- **Total Tests**: 98+ test functions
- **All Tests**: ✅ Passing

## Examples

Complete working examples are available in the `examples/` directory:

- **Gin HTTP Server**: `examples/gin/main.go` - Full HTTP server with dual-algorithm support
- **gRPC Server**: `examples/grpc/main.go` - gRPC server with interceptor
- **Token Generator**: `cmd/tokengen/` - CLI tool to generate test tokens

### Run Examples

```bash
# Gin example
cd examples/gin && go run main.go

# gRPC example
cd examples/grpc && go run main.go

# Generate test token
go run cmd/tokengen/main.go
```

## Migration from v1.x to v2.0

v2.0 is **100% backward compatible**. Existing code works unchanged.

### No Changes Required

```go
// ✅ v1.x code works perfectly in v2.0
cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))
router.Use(jwtauth.JWTAuth(cfg))
```

### Optional: Add Dual-Algorithm Support

```go
// ✨ v2.0 enhancement - add RS256 alongside existing HS256
cfg, _ := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),      // Keep existing HS256
    jwtauth.WithRS256(publicKey),   // Add RS256 support
)
router.Use(jwtauth.JWTAuth(cfg))
```

### What's Backward Compatible

- ✅ All middleware signatures unchanged
- ✅ Single-algorithm configs work identically
- ✅ Error codes unchanged for existing scenarios
- ✅ Claims extraction unchanged
- ✅ Performance maintained (no regression)
- ✅ No breaking changes in public API

### What's New (Optional)

- ✨ Dual-algorithm support (use both HS256 and RS256)
- ✨ Enhanced error messages with available algorithms
- ✨ Algorithm metadata in security logs
- ✨ `Config.AvailableAlgorithms()` method

## Documentation

- [Feature Specification](specs/002-dual-algorithm-validation/spec.md)
- [Implementation Plan](specs/002-dual-algorithm-validation/plan.md)
- [Data Model](specs/002-dual-algorithm-validation/data-model.md)
- [Quick Start Guide](specs/002-dual-algorithm-validation/quickstart.md)
- [API Contracts](specs/002-dual-algorithm-validation/contracts/)

## Contributing

Contributions are welcome! Please ensure:

1. ✅ All tests pass: `go test ./...`
2. ✅ Code is formatted: `gofmt -w .`
3. ✅ No race conditions: `go test -race ./...`
4. ✅ Benchmarks meet targets: `go test -bench=. ./jwtauth/`
5. ✅ Security scan passes: `gosec ./...`

### Development Workflow

```bash
# 1. Fork and clone
git clone https://github.com/Wang-tianhao/Vibrant-auth-middleware-go.git

# 2. Create feature branch
git checkout -b feature/my-feature

# 3. Make changes and test
go test ./jwtauth/...
go test -race ./jwtauth/...

# 4. Format code
gofmt -w .

# 5. Commit and push
git commit -m "Add feature: description"
git push origin feature/my-feature

# 6. Create pull request
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [golang-jwt/jwt](https://github.com/golang-jwt/jwt) v5
- Supports [Gin](https://github.com/gin-gonic/gin) and [gRPC](https://grpc.io/)
- Follows [RFC 7519](https://tools.ietf.org/html/rfc7519) JWT standard

---

**Author**: Tianhao Wang (tianhao.wang@vibrant-america.com)
**Repository**: https://github.com/Wang-tianhao/Vibrant-auth-middleware-go
**Version**: 2.0.0

This middleware is designed for **high-performance production use** with **security-first principles**.
