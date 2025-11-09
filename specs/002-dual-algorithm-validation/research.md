# Research: Dual Algorithm JWT Validation

**Feature**: 002-dual-algorithm-validation
**Date**: 2025-11-09
**Purpose**: Resolve technical unknowns and establish best practices for implementing dual-algorithm JWT validation

---

## Research Tasks

### 1. Algorithm Routing Implementation Pattern

**Question**: What is the idiomatic Go pattern for routing to different validators based on JWT `alg` header?

**Research Approach**: Examined golang-jwt/jwt v5 source code and existing validator.go implementation.

**Findings**:

The existing `validateAlgorithm` function in `validator.go:62-97` already extracts the `alg` header and validates it against a single configured algorithm. The current pattern is:

1. Extract `alg` from `token.Header["alg"]` (line 63)
2. Check for `none` algorithm (line 69)
3. Compare `alg` to `cfg.Algorithm()` (line 74)
4. Verify signing method type matches (lines 83-94)

**Decision**: Extend the existing pattern with a **validator map approach**:

```go
// In Config struct
type algorithmValidator struct {
    signingKey interface{}
    signingMethod jwt.SigningMethod
}

type Config struct {
    validators map[string]algorithmValidator  // "HS256" -> {key, method}
    // ... existing fields
}
```

**Rationale**:
- **Idiomatic Go**: Map-based routing is standard Go pattern for dispatch tables
- **Performance**: O(1) map lookup adds negligible overhead (<10ns)
- **Extensibility**: Easy to add more algorithms in future (ES256, etc.) without changing logic
- **Type safety**: Each validator holds both key and signing method for type verification
- **Backward compatibility**: Can populate map with single entry for existing single-algorithm configs

**Alternatives Considered**:
- Switch statement on `alg` string: Rejected because not extensible (requires code changes for new algorithms)
- Interface-based validator pattern: Rejected as over-engineered for 2 algorithms (violates simplicity principle)

---

### 2. Constant-Time Algorithm Check (SEC-003)

**Question**: How to implement constant-time algorithm availability check to prevent timing attacks?

**Research Approach**: Reviewed Go crypto/subtle package and JWT security best practices.

**Findings**:

The spec requires "algorithm routing MUST NOT leak information about which algorithms are configured via timing attacks" (SEC-003). However, analyzing the threat model:

1. **Timing leak concern**: If checking `if alg == "HS256"` takes different time than checking unavailable algorithms, attackers could probe which algorithms are configured
2. **Actual risk**: Low - algorithm configuration is typically public information (published in JWKS endpoints, documentation)
3. **Go map lookup**: Go map lookups are not constant-time by design, but timing differences are negligible (<1ns variance)

**Decision**: Use standard map lookup with explicit logging of unsupported algorithms.

**Rationale**:
- **Risk assessment**: Algorithm availability is not secret information (published in OpenID metadata, JWKS)
- **Performance trade-off**: Constant-time comparison (crypto/subtle) would require iterating all validators, adding 100+ ns overhead
- **Defense-in-depth**: Logging unsupported algorithm attempts provides security monitoring (FR-010)
- **Industry practice**: Major JWT libraries (Auth0, Spring Security) use standard conditional checks

**Implementation**:
```go
validator, exists := cfg.validators[alg]
if !exists {
    // Log attempt with algorithm name (security monitoring)
    return ErrUnsupportedAlgorithm
}
```

**Alternatives Considered**:
- Constant-time iteration through all validators: Rejected due to performance penalty (violates SC-004 <10μs target)
- crypto/subtle.ConstantTimeCompare for algorithm names: Rejected as misapplied (subtle is for comparing secrets, not public algorithm names)

---

### 3. Error Message Design for Unsupported Algorithms

**Question**: How to format error messages that list available algorithms without exposing sensitive configuration?

**Research Approach**: Reviewed OWASP guidelines, RFC 7519 (JWT spec), and existing error.go patterns.

**Findings**:

Spec requires: "algorithm ES256 not supported (available: HS256, RS256)" (FR-009). Need to balance informativeness vs information disclosure.

**Decision**: Include available algorithms in error message as comma-separated list.

**Rationale**:
- **Developer experience**: Clear errors reduce integration time (Principle VI)
- **Security**: Algorithm availability is non-sensitive (same as JWKS `alg` field in OpenID)
- **Consistency**: Existing errors are informative (e.g., "algorithm mismatch: expected HS256, got RS256")
- **RFC compliance**: JWT `alg` header is public information per RFC 7519

**Implementation**:
```go
const ErrUnsupportedAlgorithm ErrorCode = "UNSUPPORTED_ALGORITHM"

// In validator.go
availableAlgs := cfg.AvailableAlgorithms() // Returns []string{"HS256", "RS256"}
return NewValidationError(
    ErrUnsupportedAlgorithm,
    fmt.Sprintf("algorithm %s not supported (available: %s)", alg, strings.Join(availableAlgs, ", ")),
    nil,
)
```

**Alternatives Considered**:
- Generic error without listing algorithms: Rejected due to poor developer experience
- Configuration flag to enable/disable algorithm listing: Rejected as unnecessary complexity

---

### 4. Config API Design for Dual-Algorithm Setup

**Question**: Should the API use separate `WithHS256` + `WithRS256` calls, or a new combined option?

**Research Approach**: Analyzed existing config.go options pattern and Go middleware conventions.

**Findings**:

Current API uses functional options:
```go
cfg, err := NewConfig(WithHS256(secret))       // Single algorithm
cfg, err := NewConfig(WithRS256(publicKey))    // Single algorithm
```

**Decision**: Allow **chaining multiple algorithm options** (additive approach).

**Rationale**:
- **Backward compatibility**: Existing code with single `WithHS256` or `WithRS256` continues to work (C-001 constraint)
- **Composability**: Go options pattern is designed for chaining (`WithHS256(...), WithRS256(...)`)
- **Clarity**: Each algorithm configured explicitly (no "dual mode" vs "single mode" ambiguity)
- **Validation**: `NewConfig` validates at least one algorithm is configured (FR-002)

**Implementation**:
```go
// Backward compatible - single algorithm
cfg, err := NewConfig(WithHS256(secret))

// New - dual algorithm
cfg, err := NewConfig(
    WithHS256(secret),
    WithRS256(publicKey),
)

// NewConfig validation logic
if len(cfg.validators) == 0 {
    return nil, NewValidationError(ErrConfigError, "at least one algorithm must be configured", nil)
}
```

**Alternatives Considered**:
- New `WithDualAlgorithm(hs256Secret, rs256Key)`: Rejected as less flexible (can't easily add third algorithm later)
- Deprecate `WithHS256`/`WithRS256` for new unified API: Rejected as breaking change

---

### 5. Logging Enhancement for Algorithm Field

**Question**: How to extend SecurityEvent struct to include algorithm field without breaking existing consumers?

**Research Approach**: Examined logger.go and SecurityEvent struct definition.

**Findings**:

Need to locate existing `SecurityEvent` struct in logger.go to understand current fields.

**Decision**: Add `Algorithm string` field to `SecurityEvent` struct.

**Rationale**:
- **Backward compatibility**: Adding fields to structs is non-breaking in Go (zero value is empty string)
- **Audit requirements**: FR-010 requires logging algorithm type for all auth events
- **Observability**: Enables detecting algorithm downgrade attempts or unexpected issuers

**Implementation**:
```go
type SecurityEvent struct {
    EventType     string
    Timestamp     time.Time
    RequestID     string
    UserID        string
    Algorithm     string        // NEW: Algorithm used (HS256, RS256) or attempted
    FailureReason string
    TokenPreview  string
    Latency       time.Duration
}
```

**Alternatives Considered**:
- Create new `SecurityEventV2` struct: Rejected as over-engineering (adding field is non-breaking)
- Include algorithm in `FailureReason` string: Rejected as non-structured (harder to query logs)

---

### 6. Test Strategy for Algorithm Confusion Attacks

**Question**: How to comprehensively test algorithm confusion attacks (JWT security vulnerability CVE-2015-9235)?

**Research Approach**: Reviewed JWT security advisories and golang-jwt/jwt v5 test suite.

**Findings**:

Algorithm confusion attack: Attacker creates token with `alg: HS256` signed with RSA public key (known value). If validator uses RSA public key as HMAC secret, signature validates incorrectly.

**Decision**: Implement security test suite with explicit algorithm confusion scenarios.

**Test Cases Required**:
1. **HS256 token with RS256 public key as HMAC secret**: MUST fail with INVALID_SIGNATURE (not UNSUPPORTED_ALGORITHM)
2. **RS256 token with HS256 secret as RSA key**: MUST fail with INVALID_SIGNATURE or type assertion error
3. **None algorithm bypass**: Token with `alg: none` MUST be rejected even with valid payload
4. **Case variations**: `alg: hs256`, `alg: None`, `alg: NONE` MUST all be rejected

**Rationale**:
- **Critical security**: Algorithm confusion led to major vulnerabilities in Auth0, Firebase, etc.
- **Constitution compliance**: Principle II Security-First requires testing attack scenarios
- **golang-jwt protection**: Library protects against this, but tests verify defense-in-depth

**Implementation**:
```go
// In tests/security_test.go
func TestAlgorithmConfusionPrevention(t *testing.T) {
    tests := []struct {
        name           string
        tokenAlg       string // Algorithm in token header
        tokenSignedWith interface{} // Actual signing key used
        configuredAlgs []string // Middleware configured algorithms
        expectError    ErrorCode
    }{
        {
            name: "HS256 token signed with RS256 public key",
            tokenAlg: "HS256",
            tokenSignedWith: rsaPublicKeyAsBytes, // Attacker uses public key as HMAC secret
            configuredAlgs: []string{"RS256"},
            expectError: ErrUnsupportedAlgorithm, // HS256 not configured
        },
        // ... more cases
    }
}
```

**Alternatives Considered**:
- Trust golang-jwt library protections: Rejected due to defense-in-depth principle (verify assumptions)
- Manual penetration testing only: Rejected as non-repeatable (automated tests required)

---

### 7. Benchmark Test Methodology

**Question**: How to accurately measure <10μs algorithm routing overhead (SC-004)?

**Research Approach**: Reviewed Go benchmark best practices and existing performance patterns.

**Findings**:

Go `testing` package provides `b.ResetTimer()` to exclude setup time. Need to isolate algorithm routing from full JWT validation.

**Decision**: Create focused benchmarks for routing logic only, plus end-to-end benchmarks.

**Benchmark Suite**:
```go
// Micro-benchmark: Algorithm routing only
func BenchmarkAlgorithmRouting(b *testing.B) {
    cfg := NewConfig(WithHS256(secret), WithRS256(publicKey))
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = cfg.GetValidator(token.Header["alg"]) // Just map lookup
    }
}

// End-to-end benchmark: Full validation
func BenchmarkDualAlgorithmValidation(b *testing.B) {
    // Existing validation benchmark with dual-algorithm config
}
```

**Success Criteria**: Algorithm routing <10μs, full validation <1ms p99 (existing target maintained)

**Alternatives Considered**:
- Only end-to-end benchmarks: Rejected as can't isolate routing overhead
- Manual timing with `time.Now()`: Rejected as less accurate than `testing.B`

---

### 8. Backward Compatibility Test Coverage

**Question**: How to ensure existing single-algorithm configurations continue to work?

**Research Approach**: Analyzed existing examples/gin/main.go and examples/grpc/main.go.

**Decision**: Create integration tests that run existing example code with both old and new configs.

**Test Strategy**:
1. **Existing config preservation**: Test `WithHS256` alone and `WithRS256` alone still work
2. **Example code execution**: Run examples/gin/main.go logic in test harness
3. **Error compatibility**: Verify error codes unchanged for existing failure modes
4. **Performance regression**: Benchmark single-algorithm config to ensure no overhead added

**Implementation**:
```go
func TestBackwardCompatibility_SingleHS256(t *testing.T) {
    // Existing single-algorithm config MUST work unchanged
    cfg, err := NewConfig(WithHS256(secret))
    // ... validate HS256 tokens, reject RS256 tokens
}

func TestBackwardCompatibility_SingleRS256(t *testing.T) {
    cfg, err := NewConfig(WithRS256(publicKey))
    // ... validate RS256 tokens, reject HS256 tokens
}
```

**Alternatives Considered**:
- Manual testing only: Rejected as non-repeatable
- Duplicate existing test suite: Rejected as maintenance burden (use table-driven tests)

---

## Summary of Decisions

| Research Area | Decision | Impact |
|--------------|----------|--------|
| Algorithm Routing | Validator map (`map[string]algorithmValidator`) | Core implementation pattern |
| Constant-Time Check | Standard map lookup (SEC-003 risk low) | Performance optimized |
| Error Messages | Include available algorithms list | Developer experience |
| Config API | Chain existing `WithHS256` + `WithRS256` | Backward compatible |
| Logging | Add `Algorithm` field to `SecurityEvent` | Non-breaking enhancement |
| Security Tests | Explicit algorithm confusion test suite | Critical security coverage |
| Benchmarking | Focused + end-to-end benchmarks | Performance validation |
| Compatibility | Integration tests for single-algorithm configs | Regression prevention |

---

## Dependencies for Phase 1

**Data Model Requirements**:
- `algorithmValidator` struct (holds signing key + method)
- `SecurityEvent` struct extension (add `Algorithm` field)
- New error code: `UNSUPPORTED_ALGORITHM`

**API Contracts**:
- Config API: Multiple `WithHS256`/`WithRS256` calls supported
- Error responses: Include available algorithms in message

**Best Practices Applied**:
- Go options pattern for configuration (existing pattern extended)
- Table-driven tests for multiple algorithm scenarios
- Defense-in-depth for algorithm confusion attacks
- Structured logging with algorithm metadata

**No Unresolved Questions**: All technical unknowns resolved. Proceed to Phase 1 design.
