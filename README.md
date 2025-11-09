# Vibrant Auth Middleware (Go)

High-performance JWT authentication middleware for Go web frameworks with dual-algorithm support.

## Features

- ✅ **Dual-Algorithm Support** (v2.0+): Accept both HS256 and RS256 tokens simultaneously
- ✅ **Framework Support**: Gin and gRPC interceptors included
- ✅ **Zero-Allocation Routing**: <10ns algorithm routing overhead
- ✅ **Structured Logging**: Built-in security event logging with algorithm metadata
- ✅ **Type-Safe**: Strong typing with custom claims support
- ✅ **Production-Ready**: <1ms p99 latency, race-condition tested

## Quick Start

### Single Algorithm (HS256)

```go
import "github.com/user/vibrant-auth-middleware-go/jwtauth"

secret := []byte("your-256-bit-secret")
cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

// Gin
router.Use(jwtauth.JWTAuth(cfg))

// gRPC  
grpc.NewServer(grpc.UnaryInterceptor(jwtauth.UnaryServerInterceptor(cfg)))
```

### Dual Algorithm (HS256 + RS256) - v2.0+

Accept tokens from multiple issuers using different signing methods:

```go
import "github.com/user/vibrant-auth-middleware-go/jwtauth"

hs256Secret := []byte("internal-secret")
rs256PublicKey := loadPublicKey() // *rsa.PublicKey

cfg, _ := jwtauth.NewConfig(
    jwtauth.WithHS256(hs256Secret),    // Internal tokens
    jwtauth.WithRS256(rs256PublicKey), // Partner tokens
)

// Middleware now accepts BOTH HS256 and RS256 tokens!
router.Use(jwtauth.JWTAuth(cfg))
```

## Installation

```bash
go get github.com/user/vibrant-auth-middleware-go
```

## Configuration Options

```go
cfg, err := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),              // Add HS256 support
    jwtauth.WithRS256(publicKey),           // Add RS256 support (can use both!)
    jwtauth.WithCookie("auth_token"),       // Also check cookies
    jwtauth.WithClockSkew(30*time.Second),  // Clock skew tolerance
    jwtauth.WithLogger(logger),             // Structured logging
)
```

## Error Handling

The middleware returns clear, distinct error codes:

```json
{
  "error": "unauthorized",
  "reason": "UNSUPPORTED_ALGORITHM",
  "message": "algorithm ES256 not supported (available: HS256, RS256)"
}
```

**Error Codes**:
- `UNSUPPORTED_ALGORITHM`: Token uses an algorithm not configured
- `INVALID_SIGNATURE`: Signature verification failed
- `EXPIRED`: Token has expired
- `MISSING_TOKEN`: No token provided
- `MALFORMED`: Token structure is invalid
- `NONE_ALGORITHM`: "none" algorithm explicitly rejected

## Performance

- **Algorithm Routing**: <10ns (zero allocations)
- **HS256 Validation**: ~1.7μs
- **RS256 Validation**: ~22μs
- **No Regression**: Dual-algorithm config performs identically to single-algorithm

See `jwtauth/benchmark_test.go` for detailed benchmarks.

## Examples

- **Gin**: `examples/gin/main.go` - HTTP server with dual-algorithm support
- **gRPC**: `examples/grpc/main.go` - gRPC server example
- **Token Generator**: `cmd/tokengen/` - CLI tool to generate test tokens

Run examples:
```bash
cd examples/gin && go run main.go
cd examples/grpc && go run main.go
```

## Migration from v1.x to v2.0

v2.0 is **fully backward compatible**. Existing single-algorithm configs work unchanged:

```go
// v1.x code (still works in v2.0)
cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

// v2.0 new feature (optional)
cfg, _ := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),
    jwtauth.WithRS256(publicKey), // Add RS256 support
)
```

## Security

- ✅ Algorithm confusion attack prevention (SEC-001)
- ✅ "none" algorithm explicit rejection (SEC-002)
- ✅ Constant-time comparisons for sensitive operations
- ✅ Comprehensive security test suite
- ✅ Structured audit logging with algorithm metadata

## Testing

```bash
# Run tests
go test ./jwtauth/...

# Run with race detector
go test -race ./jwtauth/...

# Run benchmarks
go test -bench=. -benchmem ./jwtauth/
```

## Documentation

- [Specification](specs/002-dual-algorithm-validation/spec.md)
- [Implementation Plan](specs/002-dual-algorithm-validation/plan.md)
- [API Documentation](https://pkg.go.dev/github.com/user/vibrant-auth-middleware-go/jwtauth)

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! Please ensure:
1. Tests pass (`go test ./...`)
2. Code formatted (`gofmt -w .`)
3. No race conditions (`go test -race ./...`)
4. Benchmarks meet targets

---

**Note**: This middleware is designed for high-performance production use with security-first principles.
