# Implementation Plan: Dual Algorithm JWT Validation

**Branch**: `002-dual-algorithm-validation` | **Date**: 2025-11-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-dual-algorithm-validation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enable JWT middleware to support both HS256 (HMAC-SHA256) and RS256 (RSA-SHA256) algorithm validation simultaneously. The middleware will inspect the `alg` field in incoming JWT headers and dynamically route to the appropriate validator (HMAC or RSA) based on the algorithm specified in the token. This enables microservices to accept tokens from multiple issuers using different signing methods without requiring separate middleware instances.

## Technical Context

**Language/Version**: Go 1.24.0 (toolchain go1.24.10)
**Primary Dependencies**: github.com/golang-jwt/jwt/v5 v5.3.0, github.com/gin-gonic/gin v1.11.0, google.golang.org/grpc v1.76.0
**Storage**: N/A (stateless middleware - no persistence)
**Testing**: Go standard library testing package (testing), table-driven tests
**Target Platform**: Linux/Darwin servers (cross-platform Go middleware)
**Project Type**: Single library project (middleware package)
**Performance Goals**: <1ms p99 token validation latency, <10μs algorithm routing overhead
**Constraints**: No external dependencies beyond existing (golang-jwt, Gin, gRPC), maintain backward compatibility, synchronous validation only
**Scale/Scope**: Middleware library for microservices handling 1000+ req/s authentication

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Go Idioms ✅ PASS
- **Effective Go compliance**: Implementation will follow Go standard patterns (options pattern for config, explicit error handling)
- **Error handling**: All validation errors return explicit error types (no panic in library code)
- **Interface discipline**: Existing codebase accepts `*Config`, returns concrete types (middleware functions)
- **Naming conventions**: Current naming follows MixedCaps (e.g., `JWTAuth`, `WithHS256`)
- **Context propagation**: Context.Context properly used in existing middleware (context injection pattern)
- **Zero values**: Config struct properly initialized with defaults in `NewConfig`

**Assessment**: No violations. Feature extends existing idiomatic patterns.

### Principle II: Security-First ✅ PASS
- **OWASP compliance**: Feature explicitly addresses algorithm confusion attacks (JWT security best practice)
- **Input validation**: `alg` header will be sanitized and validated before routing (FR-006, SEC-006)
- **Secure defaults**: Existing code rejects `none` algorithm by default; feature maintains this (FR-006)
- **Timing attacks**: Algorithm routing will use constant-time checks per SEC-003
- **Token security**: Maintains existing HMAC-SHA256/RSA-SHA256 standards (no algorithm downgrade)
- **Rate limiting**: Out of scope (handled by existing infrastructure)

**Assessment**: No violations. Feature enhances security posture (prevents algorithm confusion).

### Principle III: Middleware Design Patterns ✅ PASS
- **Standard signature**: Existing middleware returns `func(http.Handler) http.Handler` for gRPC, `gin.HandlerFunc` for Gin
- **Chain compatibility**: Currently compatible with Gin/gRPC; no changes to middleware signature
- **Context injection**: Existing `WithClaims` and `WithRequestID` pattern maintained
- **Early termination**: Existing middleware aborts on validation failure (line middleware.go:25-30)
- **Configuration**: Existing options pattern (`ConfigOption` functional options) will be extended

**Assessment**: No violations. Feature extends existing middleware patterns without breaking compatibility.

### Principle IV: Testing (NON-NEGOTIABLE) ✅ PASS WITH REQUIREMENTS
- **Test-Driven Development**: MUST write tests BEFORE implementation per constitution
- **Coverage minimum**: 80% required, 100% for security-critical algorithm routing logic
- **Test types required**:
  - Unit tests: Algorithm routing logic, config validation
  - Integration tests: End-to-end dual-algorithm validation flows
  - Security tests: Algorithm confusion attacks, `none` algorithm rejection, case-sensitivity
  - Benchmark tests: Algorithm routing overhead (<10μs target per SC-004)
- **Table-driven tests**: Required for testing multiple algorithm scenarios (HS256, RS256, unsupported)

**Requirements for Implementation**:
1. Write failing tests for FR-001 through FR-013 BEFORE coding
2. Security tests MUST cover all edge cases in spec (algorithm confusion, malformed headers, etc.)
3. Benchmark tests MUST verify <10μs routing overhead
4. Integration tests MUST verify backward compatibility (existing single-algorithm config)

**Assessment**: PASS with mandatory test requirements documented above.

### Principle V: Performance & Efficiency ✅ PASS
- **Latency targets**: <1ms p99 maintained (existing target), <10μs routing overhead (SC-004)
- **Zero allocations**: Algorithm routing will use map lookup (negligible allocation)
- **Concurrent safety**: Existing `Config` is immutable after initialization (C-005 constraint)
- **Synchronous operation**: No goroutines in validation path (C-004 constraint)

**Assessment**: No violations. Algorithm routing adds minimal overhead.

### Principle VI: Documentation & Developer Experience ✅ PASS
- **Package documentation**: Will update godoc for dual-algorithm configuration
- **Function documentation**: New options (`WithDualAlgorithm` or similar) will have godoc
- **Example tests**: Will add `Example_DualAlgorithm` test function
- **README completeness**: Will document dual-algorithm quickstart
- **Migration guides**: Not breaking - backward compatible with existing single-algorithm config

**Assessment**: No violations. Documentation updates planned in Phase 1 (quickstart.md).

### Principle VII: Versioning & Backward Compatibility ✅ PASS
- **Backward compatibility**: MUST maintain existing `WithHS256`/`WithRS256` single-algorithm config (C-001)
- **No breaking changes**: New feature is additive (allows both algorithms, not required)
- **API stability**: Middleware signature unchanged (Gin `gin.HandlerFunc`, gRPC interceptor)
- **Semver compliance**: Minor version bump (new feature, no breaking changes)

**Assessment**: No violations. Feature is backward compatible.

### Security Requirements ✅ PASS
- **Cryptography Standards**:
  - Uses existing golang-jwt/jwt v5 (vetted third-party)
  - Maintains HMAC-SHA256 and RSA-SHA256 only (no algorithm downgrade)
  - No custom crypto implementation
- **Audit & Logging**:
  - Will extend existing `SecurityEvent` struct to include `algorithm` field (FR-010)
  - Existing PII protection maintained (no token values in logs)
  - Structured logging already in place (slog)

**Assessment**: No violations. Extends existing secure logging.

### Development Workflow ✅ PASS
- **Linting**: Will run golangci-lint (existing project practice)
- **Formatting**: gofmt + goimports enforced
- **Security scanning**: gosec will scan algorithm routing logic
- **Dependency checks**: No new dependencies added

**Assessment**: No violations. Standard workflow applies.

### GATE RESULT: ✅ ALL GATES PASS

**Proceed to Phase 0 Research**

No violations or complexity justifications required. Feature aligns with all constitution principles.

---

## POST-DESIGN CONSTITUTION RE-CHECK

*Re-evaluated after Phase 1 design completion (research.md, data-model.md, contracts/, quickstart.md)*

### Design Artifacts Review

**Generated Artifacts**:
1. ✅ research.md - All technical unknowns resolved with decision rationale
2. ✅ data-model.md - Entities defined with validation rules and state transitions
3. ✅ contracts/error-responses.yaml - OpenAPI error response contracts
4. ✅ contracts/config-api.md - Configuration API documentation
5. ✅ quickstart.md - Developer onboarding guide with examples

### Constitution Compliance (Post-Design)

#### Principle I: Go Idioms ✅ CONFIRMED
- **Research Decision**: Validator map pattern (`map[string]algorithmValidator`) - idiomatic Go dispatch table
- **Data Model**: Immutable Config struct with functional options pattern (existing pattern extended)
- **API Design**: Chain multiple `WithHS256` + `WithRS256` options (composable, backward compatible)
- **Error Handling**: Explicit `*ValidationError` returns with structured error codes

**Verdict**: Design follows Go idioms. No deviations.

#### Principle II: Security-First ✅ CONFIRMED
- **Algorithm Confusion Prevention**: Explicit test strategy defined (research.md section 6)
- **None Algorithm Rejection**: Enforced in validation flow (data-model.md state transitions)
- **Input Validation**: `alg` header sanitized and type-checked before routing
- **Audit Logging**: SecurityEvent extended with `Algorithm` field (non-breaking)
- **Constant-Time Analysis**: Risk assessed as low (algorithm names are public), standard map lookup used

**Verdict**: Security requirements met. Defense-in-depth approach confirmed.

#### Principle III: Middleware Design Patterns ✅ CONFIRMED
- **Signature Unchanged**: `JWTAuth(cfg *Config) gin.HandlerFunc` remains unchanged
- **Context Injection**: Existing `WithClaims` and `WithRequestID` pattern maintained
- **Early Termination**: Validation flow aborts on first error (401 response)
- **Composability**: Dual-algorithm config composable with other options (WithLogger, WithClockSkew, etc.)

**Verdict**: Middleware patterns preserved. Backward compatible.

#### Principle IV: Testing (NON-NEGOTIABLE) ✅ PLAN CONFIRMED
- **Test Strategy Defined**: research.md section 6-8 specifies:
  - Security test suite for algorithm confusion attacks
  - Benchmark tests for <10μs routing overhead
  - Integration tests for backward compatibility
  - Table-driven tests for multiple algorithm scenarios
- **TDD Requirement**: Constitution mandates tests BEFORE implementation (deferred to /speckit.tasks execution)

**Verdict**: Test strategy comprehensive and documented. TDD enforcement during implementation phase.

#### Principle V: Performance & Efficiency ✅ CONFIRMED
- **Algorithm Routing**: O(1) map lookup, <10ns latency (research.md section 1)
- **Zero Allocations**: Map lookup non-allocating (data-model.md performance section)
- **Latency Target**: <1ms p99 maintained (same as single-algorithm case)
- **Benchmarking**: Methodology defined in research.md section 7

**Verdict**: Performance targets achievable. No regressions expected.

#### Principle VI: Documentation & Developer Experience ✅ CONFIRMED
- **Quickstart Created**: quickstart.md provides 5-minute onboarding with examples
- **API Contract**: contracts/config-api.md documents all public APIs with examples
- **Error Messages**: contracts/error-responses.yaml specifies informative error responses
- **Migration Path**: Backward compatibility documented with examples (quickstart.md scenario 1)

**Verdict**: Documentation comprehensive. Developer experience prioritized.

#### Principle VII: Versioning & Backward Compatibility ✅ CONFIRMED
- **Version**: 2.0.0 (minor bump - new feature, no breaking changes)
- **Backward Compatibility**: Existing single-algorithm configs work unchanged (verified in contracts/config-api.md)
- **API Additions Only**: No removals or breaking changes to existing APIs
- **Deprecation**: Old `Algorithm()` and `SigningKey()` methods retained for compatibility

**Verdict**: Semantic versioning followed. Fully backward compatible.

### Security Requirements ✅ CONFIRMED
- **Cryptography**: Uses existing golang-jwt/jwt v5 (vetted library) - no custom crypto
- **Logging**: SecurityEvent extended with `Algorithm` field (structured JSON logging)
- **PII Protection**: No tokens or secrets logged (only preview for correlation)

**Verdict**: Security requirements met.

### Development Workflow ✅ CONFIRMED
- **Linting**: golangci-lint will run on modified files
- **Formatting**: gofmt + goimports enforced
- **Security Scanning**: gosec will scan algorithm routing logic
- **No New Dependencies**: Uses existing golang-jwt, Gin, gRPC only

**Verdict**: Standard workflow applies. No special tooling required.

---

### POST-DESIGN GATE RESULT: ✅ ALL GATES PASS

**Design is constitution-compliant. Proceed to Phase 2 (Task Generation via /speckit.tasks).**

**Key Validations**:
1. ✅ Go idioms followed (validator map, options pattern, explicit errors)
2. ✅ Security-first design (algorithm confusion prevention, audit logging)
3. ✅ Middleware patterns preserved (backward compatible)
4. ✅ Test strategy comprehensive (TDD requirements documented)
5. ✅ Performance targets achievable (<10μs routing, <1ms validation)
6. ✅ Documentation complete (quickstart, API contracts, error specs)
7. ✅ Backward compatible (v2.0.0 minor version, no breaking changes)

**No issues identified. Design ready for implementation.**

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
jwtauth/                  # Core middleware package
├── config.go            # MODIFIED: Config struct to support dual algorithms
├── validator.go         # MODIFIED: Algorithm routing logic
├── middleware.go        # MODIFIED: Logging enhancements (algorithm field)
├── grpc.go             # MODIFIED: gRPC interceptor (if needed)
├── errors.go           # MODIFIED: New error code UNSUPPORTED_ALGORITHM
├── logger.go           # MODIFIED: SecurityEvent struct (add algorithm field)
├── claims.go           # No changes
├── context.go          # No changes
├── extractor.go        # No changes
└── keys.go             # No changes

examples/
├── gin/
│   └── main.go         # MODIFIED: Add dual-algorithm example
└── grpc/
    └── main.go         # MODIFIED: Add dual-algorithm example (if applicable)

cmd/
└── tokengen/
    └── main.go         # MODIFIED: Support generating tokens with different algorithms

tests/                   # NEW: Comprehensive test suite
├── dual_algorithm_test.go       # NEW: Table-driven tests for dual-algorithm routing
├── security_test.go             # NEW: Algorithm confusion, none rejection tests
├── benchmark_test.go            # NEW: Algorithm routing performance tests
└── integration_test.go          # NEW: End-to-end Gin/gRPC integration tests
```

**Structure Decision**: Single library project. Core changes concentrated in `jwtauth/` package with modifications to existing files (config.go, validator.go, errors.go, logger.go, middleware.go). New comprehensive test files in `tests/` directory to ensure backward compatibility and security requirements.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No complexity violations. All gates passed without justifications required.
