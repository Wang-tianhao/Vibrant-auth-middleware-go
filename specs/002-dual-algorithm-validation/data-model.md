# Data Model: Dual Algorithm JWT Validation

**Feature**: 002-dual-algorithm-validation
**Date**: 2025-11-09
**Purpose**: Define entities, relationships, and state transitions for dual-algorithm validation

---

## Core Entities

### 1. AlgorithmValidator

**Purpose**: Encapsulates signing key and signing method for a specific JWT algorithm.

**Fields**:
```go
type algorithmValidator struct {
    signingKey    interface{}           // []byte for HS256, *rsa.PublicKey for RS256
    signingMethod jwt.SigningMethod     // jwt.SigningMethodHS256 or jwt.SigningMethodRS256
}
```

**Validation Rules**:
- `signingKey` MUST NOT be nil
- `signingMethod` MUST NOT be nil
- Type of `signingKey` MUST match `signingMethod`:
  - `jwt.SigningMethodHS256` → `signingKey` is `[]byte` with length ≥ 32
  - `jwt.SigningMethodRS256` → `signingKey` is `*rsa.PublicKey`

**Relationships**:
- One `Config` has 0-2 `algorithmValidator` entries (map keyed by algorithm name)
- Each validator is immutable after creation

**Lifecycle**: Created during `NewConfig()`, immutable throughout middleware lifetime

---

### 2. Config (Modified)

**Purpose**: Immutable configuration for JWT validation with support for multiple algorithms.

**Modified Fields**:
```go
type Config struct {
    // REMOVED: algorithm string (replaced by validators map)
    // REMOVED: signingKey interface{} (moved into validators)

    // NEW: Map of algorithm name to validator
    validators       map[string]algorithmValidator  // "HS256" -> validator, "RS256" -> validator

    // EXISTING: Unchanged fields
    clockSkewLeeway  time.Duration
    cookieName       string
    requiredClaims   []string
    logger           *slog.Logger
    contextKeyPrefix string
}
```

**Validation Rules** (enforced in `NewConfig`):
- `validators` map MUST NOT be empty (at least one algorithm required - FR-002)
- `validators` map MUST NOT contain `"none"`, `"None"`, or `"NONE"` keys (FR-006)
- Each validator in map MUST have valid signingKey and signingMethod
- `clockSkewLeeway` MUST be ≥ 0

**State Transitions**: None (immutable after creation)

**New Methods**:
```go
// AvailableAlgorithms returns sorted list of configured algorithm names
func (c *Config) AvailableAlgorithms() []string

// GetValidator retrieves validator for given algorithm name
func (c *Config) GetValidator(alg string) (*algorithmValidator, bool)

// HasAlgorithm checks if algorithm is configured (constant-time safe)
func (c *Config) HasAlgorithm(alg string) bool
```

**Backward Compatibility**:
- Existing `WithHS256` and `WithRS256` options populate `validators` map with single entry
- Old `Algorithm()` method deprecated but retained (returns first algorithm in sorted order for backward compat)
- Old `SigningKey()` method deprecated but retained (returns first validator's key)

---

### 3. SecurityEvent (Modified)

**Purpose**: Structured log entry for authentication events with algorithm metadata.

**Modified Fields**:
```go
type SecurityEvent struct {
    EventType     string        // "success" or "failure"
    Timestamp     time.Time
    RequestID     string
    UserID        string        // Empty for failures
    Algorithm     string        // NEW: Algorithm used ("HS256", "RS256") or attempted
    FailureReason string        // Empty for success, ErrorCode for failures
    TokenPreview  string        // First 20 chars of token (for correlation)
    Latency       time.Duration
}
```

**Validation Rules**:
- `Algorithm` MUST be populated for all events (success and failure)
- For unsupported algorithms, `Algorithm` contains the invalid algorithm name
- For malformed headers, `Algorithm` set to `"MALFORMED"` or empty string

**State Transitions**: None (immutable after creation)

**Logging Format** (structured JSON):
```json
{
  "event_type": "failure",
  "timestamp": "2025-11-09T10:30:00Z",
  "request_id": "abc-123",
  "algorithm": "ES256",
  "failure_reason": "UNSUPPORTED_ALGORITHM",
  "token_preview": "eyJhbGciOiJFUzI1NiIs...",
  "latency_ms": 0.5
}
```

---

### 4. ErrorCode (Extended)

**Purpose**: Enum of validation error codes.

**New Values**:
```go
const (
    // EXISTING error codes
    ErrExpired           ErrorCode = "EXPIRED"
    ErrInvalidSignature  ErrorCode = "INVALID_SIGNATURE"
    ErrMissingToken      ErrorCode = "MISSING_TOKEN"
    ErrMalformed         ErrorCode = "MALFORMED"
    ErrAlgorithmMismatch ErrorCode = "ALGORITHM_MISMATCH"  // Deprecated: use UNSUPPORTED_ALGORITHM
    ErrNoneAlgorithm     ErrorCode = "NONE_ALGORITHM"
    ErrConfigError       ErrorCode = "CONFIG_ERROR"

    // NEW error code for dual-algorithm feature
    ErrUnsupportedAlgorithm   ErrorCode = "UNSUPPORTED_ALGORITHM"
    ErrMalformedAlgorithmHeader ErrorCode = "MALFORMED_ALGORITHM_HEADER" // For non-string alg
)
```

**Deprecation Note**: `ErrAlgorithmMismatch` deprecated in favor of `ErrUnsupportedAlgorithm` for consistency. Retained for backward compatibility.

**Usage**:
- `UNSUPPORTED_ALGORITHM`: Token has valid `alg` header but algorithm not configured (e.g., ES256 when only HS256/RS256 configured)
- `MALFORMED_ALGORITHM_HEADER`: Token `alg` field is non-string, missing, or malformed
- `NONE_ALGORITHM`: Token explicitly uses `none` algorithm (security violation)

---

### 5. ValidationError (Unchanged)

**Purpose**: Structured error with code and message.

**Existing Structure** (no changes):
```go
type ValidationError struct {
    Code     ErrorCode
    Message  string
    Internal error
}
```

**Enhanced Messages** (for new error codes):
```go
// UNSUPPORTED_ALGORITHM example
NewValidationError(
    ErrUnsupportedAlgorithm,
    "algorithm ES256 not supported (available: HS256, RS256)",
    nil,
)

// MALFORMED_ALGORITHM_HEADER example
NewValidationError(
    ErrMalformedAlgorithmHeader,
    "algorithm header must be a string, got: <nil>",
    nil,
)
```

---

## Entity Relationships

```
Config (1) ─────► (0-2) algorithmValidator
   │
   └─ validators map[string]algorithmValidator
        ├─ "HS256" → {signingKey: []byte, signingMethod: SigningMethodHS256}
        └─ "RS256" → {signingKey: *rsa.PublicKey, signingMethod: SigningMethodRS256}

Config (1) ─────► (0-1) Logger
   │
   └─ Logs SecurityEvent (includes Algorithm field)

Token Header (1) ────► (0-1) algorithmValidator
   │
   └─ `alg` field matches validator key in Config.validators map

ValidationError (1) ────► (1) ErrorCode
```

---

## State Transitions

### Config Initialization State Machine

```
[Start]
   │
   ├─ NewConfig called
   │     │
   │     ├─ Apply WithHS256 option → Add "HS256" validator to map
   │     ├─ Apply WithRS256 option → Add "RS256" validator to map
   │     ├─ Apply other options (WithLogger, WithCookie, etc.)
   │     │
   │     └─ Validation
   │           ├─ validators map empty? → ERROR: ErrConfigError
   │           ├─ validators contains "none"? → ERROR: ErrConfigError
   │           ├─ Any validator has nil key? → ERROR: ErrConfigError
   │           └─ All valid? → [Immutable Config Created]
   │
[Immutable Config Created] → Config used by middleware (no further state changes)
```

### Token Validation Flow

```
[Token Received]
   │
   ├─ Extract `alg` from token header
   │     │
   │     ├─ Missing `alg` field? → ErrMalformedAlgorithmHeader
   │     ├─ `alg` not a string? → ErrMalformedAlgorithmHeader
   │     └─ `alg` is valid string → Continue
   │
   ├─ Check if `alg` is "none", "None", or "NONE"
   │     ├─ YES → ErrNoneAlgorithm (reject immediately)
   │     └─ NO → Continue
   │
   ├─ Lookup validator in Config.validators map
   │     ├─ Not found? → ErrUnsupportedAlgorithm + log available algorithms
   │     └─ Found → Retrieve algorithmValidator
   │
   ├─ Verify token signing method matches validator.signingMethod
   │     ├─ Mismatch? → ErrInvalidSignature (algorithm confusion attack)
   │     └─ Match → Continue
   │
   ├─ Validate signature using validator.signingKey
   │     ├─ Invalid? → ErrInvalidSignature
   │     └─ Valid → Continue
   │
   ├─ Validate claims (expiration, not-before, required claims)
   │     ├─ Invalid? → ErrExpired or ErrMalformed
   │     └─ Valid → [Success]
   │
[Success] → Log SecurityEvent with algorithm name → Inject claims into context
```

---

## Data Validation Invariants

### Config Invariants (enforced at initialization)
1. `len(validators) ≥ 1` (at least one algorithm configured)
2. `∀ alg ∈ validators.keys: alg ∉ {"none", "None", "NONE"}` (none algorithm prohibited)
3. `∀ v ∈ validators.values: v.signingKey ≠ nil ∧ v.signingMethod ≠ nil` (validators fully populated)
4. `clockSkewLeeway ≥ 0` (non-negative skew tolerance)

### Token Validation Invariants (enforced during validation)
1. `token.Header["alg"]` MUST be a non-empty string
2. `token.Header["alg"] ∉ {"none", "None", "NONE"}` (none algorithm rejected)
3. `token.Header["alg"] ∈ Config.validators.keys` (algorithm must be configured)
4. `token.Method.Alg() == token.Header["alg"]` (prevent algorithm confusion)
5. `typeof(validator.signingKey)` matches `validator.signingMethod` expectations

### SecurityEvent Invariants (enforced during logging)
1. `Algorithm` field MUST be populated (non-empty) for all events
2. `EventType ∈ {"success", "failure"}` (only two valid event types)
3. `FailureReason ≠ "" ⟺ EventType == "failure"` (failure reason only for failures)
4. `UserID ≠ "" ⟺ EventType == "success"` (user ID only for successful auth)

---

## Migration from Single-Algorithm Model

### Existing Single-Algorithm Config
```go
// Current implementation (single algorithm)
cfg := &Config{
    algorithm:  "HS256",
    signingKey: []byte("secret"),
}
```

### New Dual-Algorithm Model
```go
// New implementation (multiple algorithms)
cfg := &Config{
    validators: map[string]algorithmValidator{
        "HS256": {
            signingKey:    []byte("secret"),
            signingMethod: jwt.SigningMethodHS256,
        },
    },
}
```

### Backward Compatibility Strategy
1. **Options pattern unchanged**: `WithHS256(secret)` still works, populates `validators` map with single entry
2. **Deprecated getters retained**: `Algorithm()` returns first key from sorted validators map
3. **Validation logic extended**: `validateAlgorithm` now checks `alg ∈ validators.keys` instead of `alg == cfg.algorithm`
4. **Error codes compatible**: Existing error codes retained, new code added (non-breaking)

---

## Performance Characteristics

### Config Creation
- **Complexity**: O(n) where n = number of algorithm options (max 2 for this feature)
- **Allocations**: 1 map allocation + n validator struct allocations
- **Latency**: <1μs (negligible, occurs once at startup)

### Algorithm Routing
- **Complexity**: O(1) map lookup
- **Allocations**: 0 (map lookup is non-allocating)
- **Latency**: <10ns (map lookup)

### Token Validation
- **Complexity**: O(1) routing + O(n) signature verification (n = token length, dominated by crypto)
- **Allocations**: 0 additional (same as single-algorithm case)
- **Latency**: <1ms p99 (existing target maintained)

---

## Security Properties

### Prevented Attacks
1. **Algorithm Confusion**: Prevented by verifying `token.Method.Alg() == token.Header["alg"]` before using validator
2. **None Algorithm**: Explicitly rejected in validation flow (checked before map lookup)
3. **Unsupported Algorithms**: Caught by map lookup failure, logged for monitoring
4. **Type Confusion**: Prevented by storing `signingMethod` alongside `signingKey` in validator

### Audit Trail
- All authentication attempts logged with `Algorithm` field
- Unsupported algorithm attempts logged as security events
- Error messages include available algorithms for debugging (non-sensitive information)

---

## Summary

**Entities Added**: 1 (AlgorithmValidator)
**Entities Modified**: 3 (Config, SecurityEvent, ErrorCode)
**Entities Unchanged**: 2 (ValidationError, Claims)

**Breaking Changes**: None (all changes are additive or internal)

**Key Design Decisions**:
- Validator map pattern for O(1) algorithm routing
- Immutable config with validators populated at initialization
- SecurityEvent extended with algorithm metadata for observability
- Backward compatibility via deprecated getters and single-entry map support
