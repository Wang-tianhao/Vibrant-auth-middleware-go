# Data Model: JWT Authentication Middleware

**Feature**: JWT Authentication Middleware
**Created**: 2025-11-09
**Purpose**: Define core data structures for JWT validation, configuration, context storage, and telemetry

## Overview

This document defines the key data structures and their relationships for the JWT authentication middleware. All structures follow Go idiomatic patterns with explicit error handling, zero values where practical, and immutable configuration after initialization.

## Core Entities

### 1. Config (Configuration)

**Purpose**: Immutable configuration for middleware initialization

**Fields**:
- `Algorithm`: Signing algorithm ("HS256" or "RS256")
- `SigningKey`: Secret key ([]byte) for HS256 or RSA public key for RS256
- `ClockSkewLeeway`: Duration tolerance for exp/nbf validation (default: 60 seconds)
- `CookieName`: Name of cookie containing JWT (default: "jwt", empty = header only)
- `RequiredClaims`: Optional list of claim names that must be present
- `Logger`: slog.Logger instance for structured logging (nil = disabled)
- `ContextKeyPrefix`: String prefix for context keys (default: package name)

**Validation Rules**:
- Algorithm MUST be "HS256" or "RS256" (reject "none" and others)
- SigningKey MUST be non-nil and appropriate type for algorithm:
  - HS256: []byte with length ≥ 32 bytes (256 bits)
  - RS256: *rsa.PublicKey parsed from PEM
- ClockSkewLeeway MUST be ≥ 0 (negative = validation error)
- CookieName MAY be empty (header-only mode)
- Logger MAY be nil (silent mode, not recommended for production)

**Initialization**:
- Use functional options pattern: `NewConfig(WithHS256(secret), WithClockSkew(30*time.Second))`
- Config frozen after construction (no setters)
- Validation occurs in constructor, returns error for invalid config

**Zero Value**: Not usable (Config MUST be constructed via NewConfig)

**Relationships**:
- Used by: Middleware, Validator
- Contains: SigningKey (polymorphic: []byte or *rsa.PublicKey)

---

### 2. Claims (JWT Claims)

**Purpose**: Parsed and validated JWT claims injected into request context

**Fields**:
- `Subject`: User identifier (sub claim)
- `Issuer`: Token issuer (iss claim, optional)
- `Audience`: Intended audience (aud claim, optional)
- `ExpiresAt`: Expiration time (exp claim, required)
- `NotBefore`: Not-before time (nbf claim, optional)
- `IssuedAt`: Issue time (iat claim, optional)
- `JWTID`: JWT ID (jti claim, optional)
- `Custom`: map[string]interface{} for application-specific claims

**Validation Rules**:
- ExpiresAt MUST be present and > current time (adjusted for clock skew)
- NotBefore MUST be ≤ current time (adjusted for clock skew) if present
- Subject SHOULD be present (warning if missing, not error)
- Custom claims undergo no validation (application responsibility)

**Serialization**:
- Parsed from JWT via golang-jwt/jwt MapClaims
- Stored in request context via typed context key
- Never serialized in logs (use redaction)

**Zero Value**: Empty Claims (all fields zero/nil) - not valid

**Relationships**:
- Extracted from: JWT Token (string)
- Stored in: context.Context via contextKey
- Logged via: SecurityEvent (with redaction)

---

### 3. SecurityEvent (Telemetry Event)

**Purpose**: Structured log entry for authentication events

**Fields**:
- `EventType`: Authentication result ("success" or "failure")
- `Timestamp`: Event time (time.Time)
- `RequestID`: Correlation ID from request context or generated
- `UserID`: Subject from claims (empty if auth failed)
- `FailureReason`: Error category ("expired", "invalid_signature", "missing_token", "malformed", "algorithm_mismatch")
- `TokenPreview`: First 8 chars of token (for debugging, redacted in production)
- `Latency`: Time spent in validation (time.Duration)

**Validation Rules**:
- EventType MUST be "success" or "failure"
- Timestamp MUST be set
- RequestID SHOULD be present (generate UUID if missing)
- FailureReason MUST be set if EventType = "failure"
- TokenPreview MUST be redacted to "***" if full token would be logged

**Serialization**:
- Emitted as JSON via slog structured logging
- Uses slog.LogValuer for sensitive field redaction
- Example: `{"event":"auth_failure","timestamp":"2025-11-09T10:30:00Z","request_id":"abc-123","failure_reason":"expired","token":"***REDA"}`

**Zero Value**: Invalid (all fields required)

**Relationships**:
- Created by: Validator, Middleware
- Logged via: slog.Logger from Config
- Contains: Redacted Claims data

---

### 4. ValidationError (Error Type)

**Purpose**: Typed errors for JWT validation failures

**Fields**:
- `Code`: Error code ("EXPIRED", "INVALID_SIGNATURE", "MISSING_TOKEN", "MALFORMED", "ALGORITHM_MISMATCH", "NONE_ALGORITHM", "CONFIG_ERROR")
- `Message`: Human-readable description (safe for debugging)
- `Internal`: Underlying error (not exposed to clients)

**Validation Rules**:
- Code MUST be one of predefined constants
- Message MUST NOT leak security details (e.g., signing keys)
- Internal MAY contain sensitive details for server-side logging

**Error Interface**:
```go
func (e *ValidationError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ValidationError) Unwrap() error {
    return e.Internal
}
```

**Zero Value**: Invalid (Code and Message required)

**Relationships**:
- Returned by: Validator functions
- Logged in: SecurityEvent
- Converted to: HTTP 401 (Gin) or gRPC Unauthenticated (gRPC)

---

## Context Storage

### contextKey (Internal Type)

**Purpose**: Type-safe context keys to prevent collisions

**Definition**:
```go
type contextKey string

const (
    claimsKey contextKey = "github.com/user/jwtauth:claims"
    requestIDKey contextKey = "github.com/user/jwtauth:request_id"
)
```

**Access Functions**:
- `WithClaims(ctx context.Context, claims *Claims) context.Context`
- `GetClaims(ctx context.Context) (*Claims, bool)`
- `WithRequestID(ctx context.Context, id string) context.Context`
- `GetRequestID(ctx context.Context) (string, bool)`

**Validation Rules**:
- Keys MUST be unexported to prevent external package collisions
- Key values MUST include package import path as prefix
- Access functions MUST perform type assertions safely

**Relationships**:
- Stores: Claims, RequestID
- Used by: Middleware (Gin, gRPC), Downstream handlers

---

## State Transitions

### JWT Validation Flow

```
[Incoming Request]
    ↓
[Extract Token] → (No token) → ValidationError{MISSING_TOKEN}
    ↓
[Parse JWT] → (Malformed) → ValidationError{MALFORMED}
    ↓
[Validate Algorithm] → (Mismatch/"none") → ValidationError{ALGORITHM_MISMATCH/NONE_ALGORITHM}
    ↓
[Verify Signature] → (Invalid) → ValidationError{INVALID_SIGNATURE}
    ↓
[Validate Claims] → (Expired/NotBefore) → ValidationError{EXPIRED}
    ↓
[Success] → Claims{...}
```

### Middleware Execution Flow

```
[Request Entry]
    ↓
[Extract Token from Header/Cookie]
    ↓ (token found)
[Validate Token] → (validation error) → [Log SecurityEvent{failure}] → [Return 401/Unauthenticated]
    ↓ (validation success)
[Inject Claims into Context]
    ↓
[Log SecurityEvent{success}]
    ↓
[Call Next Handler]
    ↓
[Return Response]
```

---

## Relationships Diagram

```
┌─────────────┐
│   Config    │ (Immutable)
│ - Algorithm │
│ - SigningKey│──────┐
│ - Logger    │──┐   │
└──────┬──────┘  │   │
       │         │   │
       │ used by │   │ logs to
       ↓         ↓   ↓
┌──────────────┐ ┌───────────────┐
│  Middleware  │ │ SecurityEvent │
│  (Gin/gRPC)  │ │ - EventType   │
└──────┬───────┘ │ - Timestamp   │
       │         │ - RequestID   │
       │ calls   └───────────────┘
       ↓
┌──────────────┐
│  Validator   │
│ - ParseToken │
│ - VerifySig  │
│ - CheckClaims│
└──────┬───────┘
       │ returns
       ↓
┌──────────────┐
│    Claims    │ (Stored in context.Context)
│ - Subject    │
│ - ExpiresAt  │
│ - Custom     │
└──────────────┘
       │ injected via
       ↓
┌──────────────┐
│ contextKey   │ (Type-safe storage)
│ - claimsKey  │
└──────────────┘
```

---

## Immutability Guarantees

### Config
- ✅ **Frozen post-construction**: No exported setters
- ✅ **Deep copy on access**: SigningKey not exposed directly (getter returns copy)
- ✅ **Thread-safe**: Read-only access safe for concurrent requests

### Claims
- ⚠️ **Shallow immutability**: Standard fields immutable, Custom map mutable (caller responsibility)
- ✅ **Context isolation**: Each request gets own Claims instance

### SecurityEvent
- ✅ **Write-once**: Created and logged immediately, not mutated

---

## Performance Considerations

### Memory Allocations (Target: <3 per request)
1. **Token extraction**: 1 allocation (string slice/copy)
2. **Claims parsing**: 1 allocation (Claims struct)
3. **Context storage**: 1 allocation (context.WithValue)

**Optimization**: Reuse Config and Logger across requests (zero allocations for shared state)

### Hot Path Optimizations
- **SigningKey**: Cached in Config, not reparsed per request
- **Logger**: Shared slog.Logger instance, not recreated
- **contextKey**: String constants (no allocation)
- **ValidationError**: Error constants with slog.LogValuer (minimal allocation)

---

## Security Properties

### Confidentiality
- ✅ Signing keys never logged
- ✅ JWT tokens redacted in logs (first 8 chars only in debug mode)
- ✅ Claims.Custom not logged by default (application handles sensitive data)

### Integrity
- ✅ Config immutable after init (no TOCTOU attacks)
- ✅ Algorithm validated before signature check (no algorithm confusion)
- ✅ Signature verified with constant-time comparison (golang-jwt uses crypto/subtle)

### Availability
- ✅ No blocking operations (all validation synchronous)
- ✅ No network calls (keys provided at init)
- ✅ Configurable clock skew prevents false negatives from minor time drift

---

## Extension Points

### Custom Claims
- Applications add custom claims via `Claims.Custom` map
- Middleware does not validate custom claims (pass-through)
- Applications retrieve via `GetClaims(ctx)` and type-assert

### Custom Logging
- Inject custom slog.Handler via `WithLogger(slog.New(customHandler))`
- SecurityEvent implements slog.LogValuer for structured output
- Applications can wrap handler to add extra context fields

### Algorithm Support (Future)
- Add `WithES256(publicKey *ecdsa.PublicKey)` option
- Extend Config.Algorithm validation to accept "ES256"
- golang-jwt/jwt already supports ECDSA (no library change needed)

---

## Validation Checklist

✅ All entities have clear purpose
✅ Fields have explicit types and validation rules
✅ Relationships documented
✅ State transitions defined
✅ Zero values behavior specified
✅ Security properties explicit
✅ Performance characteristics noted
✅ Extension points identified
✅ No implementation details (e.g., struct tags, unexported fields)
✅ Aligned with functional requirements FR-001 through FR-025
