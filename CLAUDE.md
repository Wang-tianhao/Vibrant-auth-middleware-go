# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Vibrant Auth Middleware is a high-performance JWT authentication middleware for Go web frameworks with dual-algorithm support (HS256 and RS256). The project is designed for production use with a focus on security, performance, and backward compatibility.

## Architecture

### Core Package Structure

- **`jwtauth/`** - Main middleware package with zero external dependencies beyond jwt/gin/grpc
  - `config.go` - Immutable configuration with functional options pattern
  - `validator.go` - Token parsing, algorithm validation, and claims extraction
  - `middleware.go` - Gin HTTP middleware implementation
  - `grpc.go` - gRPC unary interceptor implementation
  - `claims.go` - JWT claims structure with standard and custom fields
  - `context.go` - Context injection for claims and request ID
  - `errors.go` - Typed error codes for authentication failures
  - `logger.go` - Structured security event logging
  - `extractor.go` - Token extraction from headers/cookies/metadata

### Key Design Patterns

1. **Dual-Algorithm Routing** - The `Config` struct uses a `map[string]algorithmValidator` to support multiple signing algorithms simultaneously. Algorithm selection happens in `validateAlgorithm()` at <10ns overhead.

2. **Immutable Configuration** - `NewConfig()` returns an immutable `*Config` using functional options. All validators are frozen at initialization.

3. **Security-First Validation** - Algorithm confusion attacks are prevented by explicit algorithm matching in `validator.go:112-119`. The "none" algorithm is rejected at both config-time and validation-time.

4. **Zero-Allocation Claims** - Claims are injected into `context.Context` once and retrieved with `GetClaims()`. No repeated parsing.

## Common Development Tasks

### Running Tests

```bash
# All tests
go test ./jwtauth/...

# Specific test
go test -run TestDualAlgorithm ./jwtauth/...

# With coverage
go test -cover ./jwtauth/...

# With race detector (important for concurrent validation)
go test -race ./jwtauth/...

# Verbose output
go test -v ./jwtauth/...
```

### Running Benchmarks

```bash
# All benchmarks
go test -bench=. -benchmem ./jwtauth/

# Specific benchmark
go test -bench=BenchmarkAlgorithmRouting -benchmem ./jwtauth/

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./jwtauth/
go tool pprof cpu.prof
```

### Running Examples

```bash
# Gin HTTP server example
cd examples/gin && go run main.go

# gRPC server example
cd examples/grpc && go run main.go

# Token generator CLI
go run cmd/tokengen/main.go
# Or use the compiled binary:
./tokengen
```

### Code Formatting

```bash
# Format all Go files
gofmt -w .

# Check formatting without writing
gofmt -l .
```

## Critical Implementation Details

### Algorithm Validation Flow

When a token arrives, the validation sequence is:

1. `middleware.go:extractToken()` - Extract JWT from Authorization header/cookie/metadata
2. `validator.go:parseAndValidateJWT()` - Parse token with algorithm callback
3. `validator.go:validateAlgorithm()` - **Critical security check**:
   - Reject "none" algorithm (lines 94-97)
   - Look up validator from config map (line 100)
   - Verify token's signing method matches expected method (lines 110-119)
4. `jwt.Parse()` completes with correct signing key
5. `validator.go:validateClaims()` - Validate exp/nbf with clock skew
6. `context.go:WithClaims()` - Inject claims into context

### Error Response Format

All authentication failures return HTTP 401 with structured JSON:

```json
{
  "error": "unauthorized",
  "reason": "UNSUPPORTED_ALGORITHM",
  "message": "algorithm ES256 not supported (available: HS256, RS256)"
}
```

Error codes are defined in `errors.go` and used throughout validation. The `message` field is only included for `UNSUPPORTED_ALGORITHM` and `MALFORMED_ALGORITHM_HEADER` errors.

### Security Logging

When a `*slog.Logger` is configured via `WithLogger()`, the middleware logs:

- **Success events**: requestID, userID, algorithm, latency (no token/secret in logs)
- **Failure events**: requestID, algorithm, error code, latency

See `logger.go:SecurityEvent` for the complete event structure.

## Performance Targets

All benchmarks must meet these targets (measured on Apple M4 Pro):

- **Algorithm routing**: <10 μs (actual: ~8.6 ns)
- **HS256 validation**: <1 ms (actual: ~1.7 μs)
- **RS256 validation**: <1 ms (actual: ~22 μs)

No performance regression between single-algorithm and dual-algorithm configurations.

## Testing Strategy

The test suite covers:

1. **Unit tests** (`*_test.go`) - 98+ test functions with table-driven tests
2. **Integration tests** (`integration_test.go`) - Full middleware flows with real tokens
3. **Security tests** (`security_test.go`) - Algorithm confusion attacks, "none" algorithm, malformed tokens
4. **Error message tests** (`error_messages_test.go`) - Verify all error codes and messages
5. **Benchmarks** (`benchmark_test.go`) - Performance regression prevention

Security-critical code has >85% coverage. Always run tests with `-race` flag before committing.

## Dependencies

- **Go 1.23+** (1.24+ recommended) - Leverages generics for test helpers
- **github.com/golang-jwt/jwt/v5 v5.3.0** - JWT parsing and validation
- **github.com/gin-gonic/gin v1.11.0** - HTTP middleware support
- **google.golang.org/grpc v1.76.0** - gRPC interceptor support

All dependencies are production-stable with no known CVEs.

## Backward Compatibility

v2.0 maintains 100% backward compatibility with v1.x:

- All v1.x single-algorithm configs work unchanged
- No breaking changes in public API
- Error codes unchanged for existing scenarios
- Performance maintained (no regression)

New v2.0 features are purely additive (dual-algorithm support, enhanced error messages).

## Security Considerations

1. **Algorithm Confusion Prevention** - Always validate that `token.Method.Alg()` matches the expected algorithm in `validator.go:112-119`. Never trust the header alone.

2. **Secret Key Requirements** - HS256 secrets must be ≥32 bytes (enforced in `config.go:74-76`). Shorter secrets will be rejected at config initialization.

3. **"none" Algorithm Rejection** - The "none" algorithm is rejected at both config-time (`config.go:52-56`) and validation-time (`validator.go:94-97`) to prevent bypassing signature verification.

4. **No Secret Leakage** - Tokens and secrets are never logged. Only algorithm names and error codes appear in logs.

## Project Status

- **Version**: 2.0.0
- **Status**: Production-ready
- **License**: MIT
- **Test Coverage**: 62.2% overall, >85% security-critical
- **All Tests**: Passing (98+ tests)

## Contact

- **Author**: Tianhao Wang (tianhao.wang@vibrant-america.com)
- **Repository**: https://github.com/Wang-tianhao/Vibrant-auth-middleware-go
