# Implementation Plan: JWT Authentication Middleware

**Branch**: `001-jwt-auth-middleware` | **Date**: 2025-11-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-jwt-auth-middleware/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Production-ready JWT authentication middleware for Golang with default-deny security posture, supporting both Gin HTTP framework and gRPC-style middleware patterns. Provides token validation via HS256/RS256 algorithms, multi-source token extraction (header/cookie), comprehensive security telemetry with structured JSON logging, and <1ms p99 latency performance targets. Features immediate initialization with no async delays and immutable configuration for production safety.

## Technical Context

**Language/Version**: Go 1.23+ (leverages generics for test helpers)
**Primary Dependencies**:
  - `github.com/golang-jwt/jwt/v5` (JWT validation - zero external dependencies, algorithm confusion protection)
  - `log/slog` (structured logging - stdlib, 40 B/op performance)
  - `github.com/gin-gonic/gin` (HTTP framework - user story dependency, not library dependency)
  - `google.golang.org/grpc` (RPC framework - user story dependency, not library dependency)
**Storage**: N/A (stateless middleware, no persistence)
**Testing**: Go standard library `testing` package with custom generic helpers (zero external test dependencies)
**Target Platform**: Linux/macOS/Windows server environments (cross-platform)
**Project Type**: single (Go library/middleware package)
**Performance Goals**: <1ms p99 token validation latency, <100μs p99 middleware overhead, <3 allocations per request in hot path
**Constraints**: No async operations, no network calls during request processing, immutable config post-init, Go 1.23+ compatibility
**Scale/Scope**: Production-grade library for 10k+ concurrent requests, ~2-3k LOC, 4 user stories covering Gin, gRPC, RS256, and telemetry

**Research Completed**: See [research.md](./research.md) for detailed rationale on library selection, testing approach, logging strategy, context key patterns, and middleware design patterns.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Go Idioms (NON-NEGOTIABLE)
- ✅ **Explicit error handling**: All functions return errors explicitly
- ✅ **Context propagation**: Context.Context as first parameter in validation functions
- ✅ **Functional options**: Configuration uses functional options pattern (FR-023)
- ✅ **Zero values**: Middleware usable with sensible defaults
- ✅ **Interface discipline**: Accept interfaces (e.g., http.Handler), return concrete middleware
- ✅ **Package design**: Flat structure by functionality (auth, validation, logging)
- ✅ **NO panic**: Library code returns errors, never panics

### II. Security-First (NON-NEGOTIABLE)
- ✅ **Default-deny**: All requests denied without valid JWT (FR-010)
- ✅ **Algorithm validation**: Prevents algorithm confusion attacks (FR-019, SEC-001)
- ✅ **"none" algorithm**: Explicitly rejected (FR-020, SEC-004)
- ✅ **Timing attacks**: Use crypto/subtle for constant-time comparisons (SEC-002)
- ✅ **Token replay**: Validates exp/nbf claims (FR-007, FR-008, SEC-003)
- ✅ **Secret redaction**: Logs never expose tokens/secrets (FR-015, SEC-006)
- ✅ **Input validation**: All JWT parsing sanitized and validated
- ✅ **Secure defaults**: Clock skew 60s, secure algorithms only (HS256/RS256)

### III. Middleware Design Patterns
- ✅ **Gin compatibility**: Standard gin.HandlerFunc signature (FR-001)
- ✅ **gRPC compatibility**: func(HandlerFunc) HandlerFunc pattern (FR-002)
- ✅ **Context injection**: Claims stored in request context (FR-011)
- ✅ **Early termination**: Invalid auth stops before next handler (FR-010)
- ✅ **Composability**: Can chain with other middleware independently
- ✅ **No global state**: Instance-based configuration (C-005)
- ✅ **Immutable config**: Configuration frozen post-init (FR-013, C-004)

### IV. Testing (NON-NEGOTIABLE)
- ✅ **TDD required**: Tests written before implementation
- ✅ **Coverage targets**: 80% general, 100% security-critical (SC-008)
- ✅ **Test types**: Unit, integration, security, benchmark tests required
- ✅ **Table-driven**: Preferred for multiple scenarios
- ✅ **Security tests**: Algorithm attacks, replay, tampering, timing scenarios
- ✅ **Benchmark tests**: Token validation, crypto operations performance

### V. Performance & Efficiency
- ✅ **Latency targets**: <1ms p99 validation, <100μs p99 overhead (SC-001, SC-002)
- ✅ **Allocation targets**: <3 allocations in hot path (SC-009)
- ✅ **Concurrent safety**: All state safe for concurrent access (FR-021)
- ✅ **No goroutine leaks**: Synchronous operation only (C-006)
- ✅ **Benchmark tests**: Performance-critical paths instrumented

### VI. Documentation & Developer Experience
- ✅ **Package godoc**: Comprehensive package-level documentation
- ✅ **Function godoc**: All exported functions/types documented
- ✅ **Example tests**: Common use cases (Gin, gRPC) as Example_* tests
- ✅ **Quickstart**: Installation and integration in <15 minutes (SC-010, SC-007)
- ✅ **Security docs**: Security considerations section in README
- ✅ **Error messages**: Clear, informative, non-leaking (FR-025, SEC-007)

### VII. Versioning & Backward Compatibility
- ✅ **Semantic versioning**: Strict semver 2.0 compliance
- ✅ **Go modules**: Proper go.mod, v2+ in import paths when needed
- ✅ **API stability**: v1+ APIs stable (C-007)
- ✅ **CHANGELOG**: Security-relevant changes highlighted

### Security Requirements Check
- ✅ **No custom crypto**: Use crypto/* stdlib or golang.org/x/crypto only
- ✅ **Algorithm minimums**: HMAC-SHA256 (HS256), RSA-SHA256 (RS256) minimum
- ✅ **crypto/rand only**: NO math/rand for security operations
- ✅ **Structured logging**: JSON format for security events (FR-014)
- ✅ **Audit trail**: Request ID, timestamp, user ID, action (FR-016)
- ✅ **NO PII in logs**: Redaction required (FR-015)

### Code Quality Gates Check
- ✅ **golangci-lint**: Enforced in CI
- ✅ **gofmt + goimports**: Automatic formatting
- ✅ **go vet**: Must pass
- ✅ **gosec**: Security scanning in CI
- ✅ **govulncheck**: Dependency vulnerability scanning

**GATE STATUS**: ✅ **PASS** - All constitutional requirements satisfied. No violations requiring justification.

---

## Post-Phase 1 Constitution Re-Evaluation

*After completing research, data model, and API contract design*

### Design Decisions Validation

✅ **Go Idioms Compliance**:
- data-model.md: Explicit error returns in all validation rules
- public-api.md: Context.Context as first parameter in validation functions
- public-api.md: Functional options pattern implemented (NewConfig with WithHS256, etc.)
- public-api.md: Zero values specified (Config unusable without constructor - safe)
- data-model.md: Immutable Config with no setters post-construction
- Research decision: stdlib testing (no external test dependencies)

✅ **Security-First Verification**:
- data-model.md: Algorithm validation rejects "none" explicitly
- public-api.md: Typed ValidationError prevents security detail leakage
- data-model.md: SecurityEvent with automatic token redaction
- Research decision: golang-jwt/jwt v5 with algorithm confusion protection
- Research decision: log/slog with RedactedToken type for secret protection
- data-model.md: Constant-time signature comparison (golang-jwt uses crypto/subtle)

✅ **Middleware Design Patterns Alignment**:
- public-api.md: JWTAuth returns gin.HandlerFunc (standard Gin signature)
- public-api.md: UnaryServerInterceptor returns grpc.UnaryServerInterceptor (standard gRPC)
- data-model.md: contextKey unexported with typed access functions
- data-model.md: Claims stored via context.WithValue (standard Go pattern)
- public-api.md: Early termination via c.AbortWithStatusJSON (Gin) and status.Error (gRPC)

✅ **Testing Requirements Met**:
- public-api.md: MockClaims and WithMockClaims test helpers provided
- quickstart.md: Benchmark example with target <1000 ns/op, <3 allocs/op
- data-model.md: All entities have validation rules (testable)
- Research decision: Table-driven tests supported by stdlib testing

✅ **Performance Targets Achievable**:
- Research: golang-jwt ~6725 ns/op for ES256 (HS256 faster, target <1ms = 1,000,000 ns ✅)
- Research: log/slog 40 B/op (minimal overhead for logging ✅)
- data-model.md: 3 allocations in hot path identified (extract, parse, context) ✅
- public-api.md: Config reused across requests (zero allocations for shared state) ✅

✅ **Documentation & Developer Experience**:
- quickstart.md: Complete with 5-minute Gin and 10-minute gRPC integration guides
- public-api.md: Full API documentation with examples
- data-model.md: Comprehensive entity documentation with validation rules
- quickstart.md: Troubleshooting section for common issues
- public-api.md: API stability guarantees and semver policy

✅ **Dependency Hygiene**:
- Research decision: golang-jwt/jwt v5 (zero external dependencies beyond stdlib crypto)
- Research decision: log/slog (stdlib, not third-party)
- Research decision: No testify (zero external test dependencies)
- Technical Context: Gin and gRPC as user story dependencies, not library dependencies

✅ **Security Requirements Validation**:
- Research: crypto/* stdlib usage only (HMAC-SHA256, RSA-SHA256)
- Research: No custom crypto (golang-jwt/jwt vetted library)
- data-model.md: Algorithm minimums enforced (HS256 ≥32 bytes, RS256 ≥2048-bit)
- public-api.md: ParseRSAPublicKeyFromPEM helper for PEM parsing
- data-model.md: Structured JSON logging with PII redaction

**POST-DESIGN GATE STATUS**: ✅ **PASS** - All design decisions align with constitutional requirements. Ready for Phase 2 (task generation).

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Go middleware library (single project)
jwtauth/                      # Main package (or choose name during implementation)
├── middleware.go             # Gin middleware implementation
├── grpc.go                   # gRPC middleware implementation
├── validator.go              # JWT validation logic (HS256/RS256)
├── extractor.go              # Token extraction (header/cookie)
├── config.go                 # Configuration with functional options
├── context.go                # Context keys and helpers
├── logger.go                 # Structured logging with redaction
└── errors.go                 # Error types and messages

examples/
├── gin/
│   └── main.go               # Gin integration example
└── grpc/
    └── main.go               # gRPC integration example

jwtauth_test.go               # Test files co-located with source
├── middleware_test.go        # Unit tests for middleware
├── validator_test.go         # Unit tests for validation
├── integration_test.go       # Integration tests (full flow)
├── security_test.go          # Security attack scenario tests
└── benchmark_test.go         # Performance benchmarks

go.mod                        # Go module definition
go.sum                        # Dependency checksums
README.md                     # Documentation
CHANGELOG.md                  # Version history
LICENSE                       # License file
```

**Structure Decision**: Single Go library package structure chosen because:
- This is a middleware library, not an application (no separate src/)
- Go convention: package files in root, tests co-located with _test.go suffix
- Examples in separate examples/ directory to demonstrate usage
- Flat package structure per Go idioms (no deep nesting)
- All tests in same package for white-box testing access
- No contract/ directory needed - this IS the contract (public API)

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. All constitutional requirements satisfied.
