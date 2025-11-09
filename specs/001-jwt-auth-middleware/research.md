# Technical Research: JWT Authentication Middleware

**Branch**: `001-jwt-auth-middleware` | **Date**: 2025-11-09
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

This document provides detailed technical research for critical design decisions required to implement a production-ready Go JWT authentication middleware library.

---

## 1. JWT Library Selection

### Decision: golang-jwt/jwt (v5)

**Selected**: `github.com/golang-jwt/jwt/v5`

### Rationale

#### Security Track Record
- **Actively maintained** community fork that replaced the unmaintained `dgrijalva/jwt-go` (archived May 2022 with high-severity vulnerabilities)
- **Algorithm confusion protection**: Library enforces key type matching with expected algorithm to prevent attacks
- **None algorithm protection**: Tokens using `alg=none` only accepted if constant `jwt.UnsafeAllowNoneSignatureType` explicitly provided as key, preventing accidental use of unsecured JWTs
- **Validation enforcement**: `WithValidMethods` option strongly encouraged to prevent algorithm confusion vulnerabilities
- **Production-ready**: Considered stable with established API and semantic versioning compliance

#### Performance Characteristics
- **Optimized allocations**: Recent improvements reduced ES256 signing from 65 allocs/op to 61 allocs/op (4 fewer allocations)
- **Memory efficiency**: ES256 signing at ~6725 ns/op with 4121 B/op after optimization using `big.Int.FillBytes` and `base64.RawURLEncoding`
- **HMAC performance**: HS256/HS384/HS512 recommended for best performance; HMAC is most common and fastest method
- **Minimal overhead**: Focused on JWT signing/validation only, avoiding unnecessary features that could add latency

#### Algorithm Support
- **HS256 (HMAC-SHA256)**: Expects `[]byte` values for signing and validation (symmetric encryption)
- **RS256 (RSA-SHA256)**: Expects `*rsa.PrivateKey` for signing and `*rsa.PublicKey` for validation with algorithm validation
- **Additional support**: Also supports HS384, HS512, RS384, RS512, RSA-PSS, and ECDSA
- **Critical security feature**: Library requires explicit algorithm validation in keyfunc to prevent substitution attacks

#### API Ergonomics and Go Idioms
- **Simple, focused API**: Provides bare minimum of required tooling for JWT issuing and validation
- **Type safety**: Enforces key type matching to prevent security issues at compile time
- **Idiomatic patterns**: Follows Go conventions for error handling and type assertions
- **Explicit > implicit**: Requires developers to specify algorithms and keys explicitly, reducing magic behavior
- **Standard library alignment**: Minimal dependencies, relies on Go's crypto/* packages

#### Community Adoption
- **Community-blessed fork**: Official successor to dgrijalva/jwt-go, endorsed by original author
- **Production usage**: Widely used with Gin, Fiber, and other Go web frameworks
- **Stable versioning**: v4.0.0+ adds Go module support while maintaining backward compatibility; v5.0.0 introduces major validation improvements
- **Semantic versioning**: Strict SemVer 2.0.0 compliance with minimal breaking changes outside major versions

#### Dependency Footprint
- **Minimal dependencies**: Only relies on Go standard library crypto packages
- **No transitive dependencies**: Clean dependency tree suitable for security-conscious environments
- **Small import surface**: Core JWT functionality without bloat from unused features

### Alternatives Considered

#### lestrrat-go/jwx (v3)
**Strengths**:
- Complete JOSE implementation (JWA, JWE, JWK, JWS, JWT)
- Advanced features like auto-refresh for JWK key rotation
- Support for JWS messages with multiple signatures
- Both compact and JSON serialization support
- Recent performance optimizations with low-level JWS API
- Uniform, opinionated API design with `WithXXXX()` style options
- Comprehensive handling of all JWx specifications

**Weaknesses**:
- **Significantly larger dependency footprint**: v1 dependencies include 9+ external packages (decred/dcrd, goccy/go-json, lestrrat-go/backoff, lestrrat-go/blackmagic, lestrrat-go/httpcc, lestrrat-go/iter, lestrrat-go/option, pkg/errors, golang.org/x/crypto)
- **Import complexity**: JWE package has 31 imports, JWS package has 37 imports (compared to golang-jwt's minimal imports)
- **Feature overload**: Full JOSE implementation includes many features not needed for basic JWT middleware (JWE, JWK auto-refresh)
- **Security vulnerabilities**: Has had multiple DoS vulnerabilities (nil pointer dereference in v2.0.19, p2c parameter DoS, AES-CBC Padding Oracle Attack, all patched but indicates surface area concerns)
- **Version churn**: v0/v1 deprecated, users must migrate to v3 (highest tagged major version)
- **Complexity**: 3x-4x more complex than needed for straightforward HS256/RS256 validation in middleware

**Why NOT chosen**:
- Violates constraint C-001: "Must not introduce external dependencies beyond Go standard library and well-vetted JWT libraries"
- Significantly heavier dependency footprint contradicts "minimal dependencies" principle
- Features like JWE encryption, JWK key rotation, multiple signatures are overkill for middleware authentication
- Larger attack surface with more dependencies and code paths
- For middleware validation-only use case, the comprehensive JOSE implementation is unnecessary complexity

### Trade-offs

**Chosen Approach (golang-jwt/jwt)**:
- **Pros**: Minimal dependencies, focused API, strong community adoption, proven security track record, optimized for JWT-only use case, Go idiomatic, lightweight
- **Cons**: Lacks advanced JOSE features (JWE, JWK auto-refresh, multiple signatures) if needed in future; less comprehensive than full JOSE implementation

**Impact**: The minimalist approach of golang-jwt/jwt aligns perfectly with middleware requirements (validation only, HS256/RS256 support, low latency). If future requirements demand JWE encryption or JWK auto-refresh, we can add lestrrat-go/jwx as an optional enhancement, but the core middleware should remain simple and focused.

**Mitigation**: For organizations needing JWK auto-refresh, document pattern of using golang-jwt/jwt with separate JWK fetching logic rather than bundling it into the middleware. This keeps concerns separated and dependencies minimal.

---

## 2. Testing Framework Selection

### Decision: Go Standard Library (testing package)

**Selected**: Go standard library `testing` package with optional `golang.org/x/exp/constraints` for generic test helpers

### Rationale

#### Alignment with Go Idiomatic Testing
- **Go philosophy**: "Writing idiomatic tests in Go isn't that different from writing application code" - there isn't a single assertion function in the testing package by design
- **Explicit over magic**: Using `if got != want { t.Errorf(...) }` makes test logic immediately clear without hidden abstractions
- **No external dependencies**: Stays true to constraint C-001 and minimizes dependency bloat
- **Community momentum (2024)**: Experienced Go developers increasingly favor standard library approach for simplicity and clarity

#### Readability and Maintainability
- **Direct comparisons**: Test failures show exactly what comparison was made without navigating through assertion library internals
- **go-cmp integration**: For complex comparisons, `google/go-cmp` provides clearer, more to-the-point error messages than testify
- **Post-generics era**: With Go 1.23+ generics, writing type-safe helper functions is trivial: `func assertEqual[T comparable](t *testing.T, got, want T)` eliminates need for third-party libraries
- **Reduced indirection**: No need to understand testify's 80+ exported functions (14,000 lines of code) - test logic is self-contained

#### Performance Impact
- **Zero overhead**: No assertion library wrapping means no extra allocations or function calls
- **Benchmark-friendly**: Standard library testing integrates seamlessly with Go's benchmarking (`testing.B`)
- **Minimal build time**: No external package compilation reduces CI/CD build times

#### Community Standards in Go Middleware
- **Standard library preference**: Many production Go middleware libraries (grpc-ecosystem/go-grpc-middleware, go-kit/kit) use standard library testing
- **Table-driven tests**: Standard library's `t.Run()` subtests (Go 1.7+) provide excellent support for table-driven tests without needing testify
- **Go project conventions**: Official Go projects (standard library, golang.org/x/*) exclusively use standard library testing

#### Support for Table-Driven Tests
- **Built-in subtests**: `t.Run(testCase.name, func(t *testing.T) {...})` provides scoped test execution
- **Parallel execution**: `t.Parallel()` works seamlessly with standard library
- **Clear test structure**: Anonymous structs for test cases are idiomatic:
  ```go
  tests := []struct {
      name string
      input string
      want  string
  }{
      {name: "valid token", input: "...", want: "..."},
  }
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) {
          if got := validate(tt.input); got != tt.want {
              t.Errorf("validate(%q) = %v, want %v", tt.input, got, tt.want)
          }
      })
  }
  ```

### Alternatives Considered

#### testify/assert
**Strengths**:
- Fluent API: `assert.Equal(t, expected, actual)` saves 2-3 lines per assertion
- Rich assertion library: 80+ assertion functions for various comparison types
- Suite support: `testify/suite` provides setup/teardown and test organization
- Mock support: `testify/mock` for creating test doubles
- Better Stack traces: Helpful for complex nested assertions

**Weaknesses**:
- **Unnecessary abstraction**: While `assert.Equal` saves 3 lines of code, it "bloats tests with dependencies and indirections, making it harder to understand what's happening behind the scenes" (2024 community perspective)
- **Complexity**: Full assert package adds 14,000 lines and exports 80 functions - significant overhead for what amounts to comparison helpers
- **Post-generics redundancy**: With Go 1.18+ generics, creating type-safe helpers is trivial, reducing testify's value proposition
- **External dependency**: Violates principle of minimal dependencies for a library
- **Not idiomatic**: Go team and experienced gophers advocate for standard library approach
- **Learning curve**: New contributors must learn testify API vs. straightforward Go comparisons

**Why NOT chosen**:
- Constraint C-001: "Must not introduce external dependencies beyond Go standard library and well-vetted JWT libraries"
- Testing libraries don't meet the bar of "well-vetted" in same category as security-critical JWT libraries
- With Go 1.23 generics, we can write `testhelpers.go` with type-safe comparison helpers in <50 lines
- Community trend (2024) favors standard library for production middleware
- Examples from `math` and `time` packages show standard library testing is sufficient for complex logic

### Trade-offs

**Chosen Approach (Standard Library)**:
- **Pros**: Zero dependencies, idiomatic Go, explicit test logic, no external learning curve, perfect integration with benchmarks, faster builds, future-proof (no external API changes)
- **Cons**: Slightly more verbose for complex assertions (mitigated by go-cmp), no built-in mocking (use interfaces and manual mocks), 2-3 extra lines per comparison

**Impact**: Tests will be 10-20% more verbose but 100% transparent. For a security-critical middleware library, the trade-off of verbosity for clarity and zero dependencies is strongly favorable. Example:

**With testify**:
```go
assert.Equal(t, expectedClaims, actualClaims)
```

**Standard library**:
```go
if diff := cmp.Diff(expectedClaims, actualClaims); diff != "" {
    t.Errorf("claims mismatch (-want +got):\n%s", diff)
}
```

**Mitigation**: Create `internal/testutil/testutil.go` with generic helpers for common patterns:
```go
func RequireEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Fatalf("got %v, want %v", got, want)
    }
}
```

This provides testify-like ergonomics with zero external dependencies.

---

## 3. Structured JSON Logging

### Decision: log/slog (Go 1.21+ standard library)

**Selected**: `log/slog` from Go standard library (Go 1.21+)

### Rationale

#### Zero Dependencies
- **Standard library**: Part of Go 1.21+ (released August 8, 2023), no external dependencies required
- **Constraint compliance**: Perfectly aligns with C-001 "Must not introduce external dependencies beyond Go standard library"
- **Future-proof**: Will be maintained indefinitely as part of Go's commitment to backward compatibility
- **No dependency churn**: No risk of external library abandonment or breaking changes

#### Performance for High-Throughput Middleware
- **Efficient memory usage**: slog allocates only **40 B/op** (same as zerolog, 4x better than zap's 168 B/op)
- **Fewest allocations**: slog has the **fewest allocations per operation**, closely followed by zerolog, while zap requires 3 allocs/op
- **Speed trade-off**: slog is somewhat slower than zerolog/zap in raw operation time, but memory efficiency is more critical than sheer speed for middleware
- **Good balance**: "slog offers a good balance of features and zero external dependencies" for production applications
- **Middleware context**: For middleware logging (not ultra-high-frequency trading), slog's performance profile is more than adequate for <100μs overhead target (SC-002)

#### Secret Redaction Patterns
- **Custom LogValuer interface**: "A big reason to move to slog is the ease with which we can build field redaction with a custom LogValuer"
- **Nested struct support**: LogValuer supports nested structs, enabling comprehensive redaction across complex claim structures
- **Type-safe redaction**: Implement custom types with `LogValue()` method to automatically redact sensitive fields:
  ```go
  type RedactedToken string
  func (t RedactedToken) LogValue() slog.Value {
      if len(t) < 10 {
          return slog.StringValue("***REDACTED***")
      }
      return slog.StringValue(string(t[:8]) + "***")
  }
  ```
- **Handler-level filtering**: Can create custom slog.Handler wrappers to intercept and redact patterns globally
- **Production pattern**: Use slog.GroupValue for JWT claims with redacted token but logged metadata (exp, iss, sub)

#### Integration with Middleware
- **Context-aware logging**: slog integrates naturally with context.Context for request-scoped logging
- **Structured attributes**: `slog.String("request_id", id)` provides clean structured logging for telemetry
- **Handler flexibility**: JSON handler for production (`slog.NewJSONHandler`), text handler for development
- **Middleware example**: httplog library (go-chi/httplog) demonstrates production-ready HTTP middleware using slog with zero dependencies

#### Go Ecosystem Alignment
- **Standard library commitment**: Go team's addition of slog signals long-term commitment to structured logging in stdlib
- **Ethereum adoption**: Major projects like go-ethereum considering slog migration (Issue #28244)
- **2024-2025 recommendation**: "For most new projects, slog provides a great balance of features and simplicity while avoiding external dependencies"

### Alternatives Considered

#### uber-go/zap
**Strengths**:
- Fastest structured logging after zerolog
- Excellent customization capabilities
- Battle-tested in high-performance microservices
- Rich ecosystem and middleware integrations
- Good for ultra-low-latency requirements

**Weaknesses**:
- **External dependency**: Adds dependency tree (uber-go/atomic, uber-go/multierr)
- **Memory overhead**: 168 B/op (4x worse than slog/zerolog)
- **Higher allocations**: 3 allocs/op vs. slog's minimal allocations
- **Complexity**: More configuration options than needed for middleware logging
- **Overkill**: For <100μs middleware overhead target, slog is sufficient

#### rs/zerolog
**Strengths**:
- **Fastest logging library**: Best operation time among all Go loggers
- **Memory efficient**: 40 B/op (same as slog)
- **JSON-centric design**: Optimized for structured JSON logging
- **Production adoption**: Used by CrowdStrike and other high-performance systems
- **Excellent performance**: Ideal for high-throughput applications

**Weaknesses**:
- **External dependency**: Violates C-001 constraint preference for minimal dependencies
- **Not necessary**: For middleware (not core business logic), slog's performance is adequate
- **Ecosystem risk**: Third-party library maintenance depends on continued community support
- **Learning curve**: Different API patterns from slog (method chaining vs. attribute passing)

**Why NOT chosen (both zap and zerolog)**:
- Constraint C-001 strongly prefers standard library when possible
- slog's performance (40 B/op, minimal allocations) meets SC-002 target (<100μs overhead)
- For middleware logging (not ultra-high-frequency operations), speed difference is negligible
- Zero dependencies reduce supply chain risk and dependency management overhead
- slog provides built-in secret redaction via LogValuer, reducing custom code
- Future Go versions will continue optimizing slog, while third-party libraries may stagnate

### Trade-offs

**Chosen Approach (log/slog)**:
- **Pros**: Zero dependencies, memory efficient (40 B/op), fewest allocations, native secret redaction support, future-proof, Go ecosystem alignment, meets performance targets
- **Cons**: Somewhat slower raw operation time than zerolog/zap (acceptable for middleware), newer API (less Stack Overflow content than zap)

**Impact**: Logging will add minimal overhead (<50μs) and zero external dependencies. Secret redaction is built-in via LogValuer. For security telemetry (FR-014, FR-015, FR-016), slog's structured JSON output with custom handlers provides production-grade observability.

**Example Secret Redaction Pattern**:
```go
type Claims struct {
    UserID string `json:"user_id"`
    Token  RedactedToken `json:"token"`
    Exp    int64  `json:"exp"`
}

type RedactedToken string
func (t RedactedToken) LogValue() slog.Value {
    return slog.StringValue("***REDACTED***")
}

// Usage in middleware
slog.Info("auth success",
    slog.String("user_id", claims.UserID),
    slog.Int64("exp", claims.Exp),
    slog.Any("token", claims.Token), // Automatically redacted
)
```

**Mitigation**: If future profiling reveals slog is a bottleneck (unlikely for middleware), provide configuration option to swap logger via interface, but default to slog for dependency-free experience.

---

## 4. Context Key Safety

### Decision: Typed Context Keys with Unexported Custom Type

**Selected**: Unexported `type contextKey string` pattern with exported getter/setter functions

### Rationale

#### Type Safety and Collision Prevention
- **Custom unexported type**: "Packages should define keys as an unexported type to avoid collisions" - Go documentation
- **Prevents accidental collisions**: Using plain string keys across packages can cause value overwrites in context
- **Compile-time safety**: Custom type ensures keys from different packages are incompatible at type level
- **Go best practice**: "The provided key must be comparable and should not be of type string or any other built-in type to avoid collisions between packages using context"

#### Best Practice Patterns from Production Libraries

**Pattern 1: Unexported string-based type (recommended for this project)**:
```go
// internal/context.go
type contextKey string

const (
    claimsKey contextKey = "jwt-claims"
    userIDKey contextKey = "user-id"
)

// Exported getters/setters
func WithClaims(ctx context.Context, claims *Claims) context.Context {
    return context.WithValue(ctx, claimsKey, claims)
}

func GetClaims(ctx context.Context) (*Claims, bool) {
    claims, ok := ctx.Value(claimsKey).(*Claims)
    return claims, ok
}
```

**Pattern 2: Pointer to struct (used by Auth0 go-jwt-middleware)**:
```go
type ContextKey struct{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
    return context.WithValue(ctx, ContextKey{}, claims)
}
```

**Pattern 1 selected** because:
- More readable with named constants (claimsKey vs. anonymous ContextKey{})
- Easier debugging (can print key value for inspection)
- Slight memory advantage (no pointer allocation)
- Industry standard (used by go-kit/kit, iris-contrib)

#### Encapsulation and API Safety
- **Type-specific getters**: "Don't set context values directly, but instead use getters and setters that are type specific"
- **Prevents misuse**: Users cannot directly call `context.WithValue` with wrong key types
- **Type assertions safety**: Always use two-value type assertion `claims, ok := ctx.Value(...)` to avoid panics
- **Clear API surface**: Exported `WithClaims()` and `GetClaims()` functions document intended usage

#### JWT Claims Storage Pattern
- **Common practice**: Store validated claims as `*Claims` pointer in context for downstream handlers
- **Nil-safe**: Check `ok` return value before dereferencing claims pointer
- **Immutable**: Claims stored in context should not be modified by handlers (defensive copy if needed)
- **Multiple keys**: Use separate keys for different auth information (user ID, roles, permissions)

#### Go Kit Example (Industry Standard)
```go
// From go-kit/kit/auth/jwt/middleware.go
type contextKey string

const (
    JWTContextKey       contextKey = "JWTToken"
    JWTClaimsContextKey contextKey = "JWTClaims"
)

// Separate keys for token and claims
```

### Implementation Pattern for This Project

```go
// pkg/jwtauth/context.go
package jwtauth

type contextKey string

const (
    claimsContextKey contextKey = "github.com/vibrant/jwtauth:claims"
    tokenContextKey  contextKey = "github.com/vibrant/jwtauth:token"
)

// WithClaims stores validated JWT claims in the request context.
// Claims are immutable and should not be modified by downstream handlers.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
    return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaims retrieves validated JWT claims from the request context.
// Returns nil, false if claims are not present or have wrong type.
// Always check the ok return value before using claims.
func GetClaims(ctx context.Context) (*Claims, bool) {
    claims, ok := ctx.Value(claimsContextKey).(*Claims)
    return claims, ok
}

// WithToken stores the raw JWT token string in context (for logging/debugging).
// Token is redacted in logs via RedactedToken type.
func WithToken(ctx context.Context, token string) context.Context {
    return context.WithValue(ctx, tokenContextKey, RedactedToken(token))
}

// GetToken retrieves the raw JWT token from context.
// Use sparingly; prefer GetClaims for accessing user information.
func GetToken(ctx context.Context) (string, bool) {
    token, ok := ctx.Value(tokenContextKey).(RedactedToken)
    return string(token), ok
}

// RedactedToken wraps string to provide automatic redaction in slog output
type RedactedToken string

func (t RedactedToken) LogValue() slog.Value {
    if len(t) == 0 {
        return slog.StringValue("")
    }
    return slog.StringValue("***REDACTED***")
}
```

### Alternatives Considered

#### Plain String Keys
**Why NOT chosen**:
- "should not use basic type string as key in context.WithValue" - golint warning
- High collision risk across packages using similar string keys ("user", "claims")
- No compile-time safety
- Against Go documentation recommendations

#### Exported Context Keys
**Why NOT chosen**:
- Encourages direct `context.WithValue(PublicKey, ...)` usage bypassing type safety
- Users might set wrong value types, causing runtime panics
- Better to export getter/setter functions instead of raw keys

#### Interface{} or Comparable Constraint
**Why NOT chosen**:
- Over-engineered for simple key usage
- Doesn't provide additional safety over unexported custom type
- Adds complexity without benefit

### Trade-offs

**Chosen Approach (Unexported contextKey string)**:
- **Pros**: Type-safe, prevents collisions, idiomatic Go, explicit API via getters/setters, debuggable, minimal overhead
- **Cons**: Requires getter/setter boilerplate (mitigated by codegen or simple implementation), users can't inspect keys directly (intentional for encapsulation)

**Impact**: Downstream handlers access claims via type-safe API:
```go
// In Gin handler
func ProtectedHandler(c *gin.Context) {
    claims, ok := jwtauth.GetClaims(c.Request.Context())
    if !ok {
        c.JSON(500, gin.H{"error": "claims not found"})
        return
    }
    c.JSON(200, gin.H{"user_id": claims.UserID})
}
```

**Mitigation**: Provide comprehensive examples in documentation showing context key usage patterns to prevent misuse.

---

## 5. Gin and gRPC Middleware Patterns

### Decision: Standard Gin Middleware + gRPC Unary Interceptor Patterns

**Selected**:
- **Gin**: Standard `gin.HandlerFunc` with `c.Next()` / `c.Abort()` pattern
- **gRPC**: Unary interceptor with `grpc.UnaryServerInterceptor` signature

### Rationale

#### Gin Middleware Pattern (FR-001)

**Standard Signature**:
```go
func JWTAuth(config Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extract token from request (header/cookie)
        token, err := extractToken(c.Request)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        // 2. Validate token
        claims, err := validateToken(token, config)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        // 3. Inject claims into context
        ctx := WithClaims(c.Request.Context(), claims)
        c.Request = c.Request.WithContext(ctx)

        // 4. Continue to next handler
        c.Next()
    }
}
```

**Key Patterns**:
- **HandlersChain**: Middleware executed in order added via `router.Use()`
- **c.Next()**: Explicitly executes pending handlers in the chain; use only inside middleware
- **c.Abort()** / **c.AbortWithStatusJSON()**: Terminates request chain, prevents downstream handlers from executing
- **Context propagation**: Use `c.Request = c.Request.WithContext(ctx)` to propagate modified context
- **Execution order**: Global middleware via `Use()` runs for all routes including 404/405
- **Sharing data**: Set values via `c.Set(key, value)` for Gin-specific context, or use `context.Context` for standard Go propagation

**Best Practices from Research**:
- Middleware functions execute in order they are added (order matters for auth → logging → business logic)
- Use HandlersChain for reusable, composable middleware components (DRY principle)
- Separate concerns: auth middleware only handles authentication, not authorization
- Always call `c.Next()` after setting context to allow chain to continue
- Use `c.AbortWithStatusJSON()` for early returns on auth failure (default-deny, FR-010)

**Gin Context Safety**:
- Gin Context is only valid for request duration
- If spawning goroutines, copy context: `cCopy := c.Copy()`
- For background operations, extract `c.Request.Context()` for standard context propagation

#### gRPC Interceptor Pattern (FR-002)

**Unary Interceptor Signature** (matches spec's "HandlerFunc wrapping style"):
```go
func UnaryServerInterceptor(config Config) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 1. Extract token from gRPC metadata
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "metadata not found")
        }

        token, err := extractTokenFromMetadata(md)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "token not found")
        }

        // 2. Validate token
        claims, err := validateToken(token, config)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }

        // 3. Inject claims into context
        ctx = WithClaims(ctx, claims)

        // 4. Call handler with enriched context
        return handler(ctx, req)
    }
}
```

**Key Patterns**:
- **Interceptor signature**: `func(ctx, req, info, handler) (interface{}, error)`
- **Context propagation**: Return modified context to handler (automatically propagated)
- **Metadata extraction**: Use `metadata.FromIncomingContext(ctx)` to access gRPC metadata
- **Error handling**: Return `status.Error(codes.Unauthenticated, msg)` for auth failures (FR-018)
- **Handler invocation**: Call `handler(ctx, req)` to continue processing

**Chaining Multiple Interceptors**:
```go
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metrics.UnaryInterceptor(),
        logging.UnaryInterceptor(),
        jwtauth.UnaryServerInterceptor(config), // Our auth interceptor
        recovery.UnaryInterceptor(),
    ),
)
```

**Interceptor Execution Order**:
- First interceptor in chain is **outermost** (wraps everything)
- Last interceptor is **innermost** (closest to actual handler)
- Example: metrics → logging → auth → recovery → handler
- Auth should be early in chain to deny unauthenticated requests before heavy processing

**Best Practices from grpc-ecosystem/go-grpc-middleware**:
- Use `grpc.ChainUnaryInterceptor()` for multiple interceptors (don't nest manually)
- Pre-handler operations: extract metadata, validate, start timing
- Post-handler operations: record metrics, transform errors, add response metadata
- Respect context cancellation: check `ctx.Done()` for long-running operations
- Propagate context values correctly via modified context return

#### Context Propagation Between Middleware Layers

**Shared Pattern**:
```go
// Common validation logic shared by Gin and gRPC
func validateAndEnrichContext(ctx context.Context, token string, config Config) (context.Context, error) {
    claims, err := validateToken(token, config)
    if err != nil {
        return ctx, err
    }
    return WithClaims(ctx, claims), nil
}

// Gin middleware
func GinMiddleware(config Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        token, _ := extractTokenFromRequest(c.Request)
        ctx, err := validateAndEnrichContext(c.Request.Context(), token, config)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
            return
        }
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}

// gRPC interceptor
func GRPCInterceptor(config Config) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        md, _ := metadata.FromIncomingContext(ctx)
        token, _ := extractTokenFromMetadata(md)
        ctx, err := validateAndEnrichContext(ctx, token, config)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, err.Error())
        }
        return handler(ctx, req)
    }
}
```

**Context Safety Principles**:
- `context.Context` is immutable; `WithValue` returns new context
- Always propagate enriched context to next handler (Gin: `c.Request.WithContext`, gRPC: return to handler)
- Use same context key accessors (`GetClaims`) in both Gin and gRPC handlers
- Context values set in middleware are available in downstream handlers

#### Production Examples

**go-kit/kit JWT middleware**:
- Defines `type contextKey string` for claim keys
- Provides `NewParser` with keyfunc for token validation
- Uses `jwt.ParseWithClaims` from golang-jwt/jwt
- Stores claims in `context.WithValue(ctx, JWTClaimsContextKey, claims)`

**grpc-ecosystem/go-grpc-middleware**:
- Shows interceptor chaining for auth, logging, metrics, recovery
- Demonstrates pre/post handler hooks
- Uses context propagation for request-scoped values

### Implementation Structure

```
jwtauth/
├── middleware.go       # Gin middleware (func(Config) gin.HandlerFunc)
├── grpc.go            # gRPC interceptor (func(Config) grpc.UnaryServerInterceptor)
├── validator.go       # Shared validation logic
├── extractor.go       # Token extraction (HTTP headers/cookies, gRPC metadata)
├── context.go         # Context key definitions and helpers
└── config.go          # Shared configuration
```

### Alternatives Considered

#### Custom Middleware Type
**Why NOT chosen**:
- Gin and gRPC have established middleware signatures
- Custom type would require adapters, adding complexity
- Standard signatures enable easy integration with existing middleware ecosystems
- Users expect `gin.HandlerFunc` and `grpc.UnaryServerInterceptor` types

#### Single Unified Middleware
**Why NOT chosen**:
- Gin and gRPC have fundamentally different request/response models
- Forced abstraction would leak implementation details
- Better to have two thin adapters over shared validation core
- Separate functions provide clearer API surface

### Trade-offs

**Chosen Approach (Standard Patterns)**:
- **Pros**: Idiomatic for each framework, easy integration, familiar to users, composable with other middleware, clear separation of concerns
- **Cons**: Slight code duplication in adapters (mitigated by shared validation core), must maintain two middleware implementations

**Impact**:
- Gin users: `router.Use(jwtauth.GinMiddleware(config))`
- gRPC users: `grpc.NewServer(grpc.UnaryServerInterceptor(jwtauth.GRPCInterceptor(config)))`
- Shared validation logic in `validator.go` (60-70% code reuse)
- Context propagation works identically in both frameworks via standard `context.Context`

**Mitigation**: Comprehensive examples in `examples/gin/` and `examples/grpc/` demonstrating integration patterns, context access, and middleware chaining.

---

## Summary of Decisions

| Decision Area | Selected | Key Rationale |
|--------------|----------|---------------|
| **JWT Library** | golang-jwt/jwt v5 | Minimal dependencies, proven security, optimized performance, focused API, community-blessed |
| **Testing Framework** | Go stdlib testing | Zero dependencies, idiomatic, explicit test logic, post-generics type safety, community momentum |
| **Logging** | log/slog | Standard library, 40 B/op efficiency, native secret redaction, zero dependencies, future-proof |
| **Context Keys** | Unexported contextKey string | Type safety, collision prevention, Go best practice, clear API via getters/setters |
| **Middleware Patterns** | Standard Gin HandlerFunc + gRPC Unary Interceptor | Idiomatic, composable, familiar, framework-specific optimizations, shared validation core |

All decisions align with:
- **Constitution requirements**: Go idioms, security-first, minimal dependencies, performance targets
- **Constraints**: C-001 (minimal dependencies), C-002 (idiomatic patterns), C-003 (Go 1.23+)
- **Success criteria**: SC-001/SC-002 (performance), SC-003 (security), SC-008 (test coverage)
- **Functional requirements**: FR-001/FR-002 (framework compatibility), FR-014/FR-015/FR-016 (telemetry)

---

## Next Steps

1. **Phase 1 - Design**: Create data-model.md defining JWT Claims structure, Config options, and Error types
2. **Phase 1 - Design**: Create contracts/ defining public API surface for Gin/gRPC middleware
3. **Phase 1 - Design**: Create quickstart.md with integration examples for both frameworks
4. **Phase 2 - Tasks**: Generate tasks.md with dependency-ordered implementation tasks
5. **Phase 3 - Implementation**: TDD implementation starting with validator tests, then middleware tests

**References**:
- golang-jwt/jwt: https://github.com/golang-jwt/jwt
- Go slog: https://pkg.go.dev/log/slog
- grpc-ecosystem/go-grpc-middleware: https://github.com/grpc-ecosystem/go-grpc-middleware
- go-kit/kit JWT: https://github.com/go-kit/kit/blob/master/auth/jwt/middleware.go
- Go context best practices: https://pkg.go.dev/context
