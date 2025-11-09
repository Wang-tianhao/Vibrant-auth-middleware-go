# Feature Specification: Dual Algorithm JWT Validation

**Feature Branch**: `002-dual-algorithm-validation`
**Created**: 2025-11-09
**Status**: Draft
**Input**: User description: "Update. Middleware should be initialized with both HS256 and RS256 validators. And it should depends on the incoming token claim alg field to decide use which one to validate. Error on unsupported alg."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Dual Algorithm Configuration (Priority: P1)

As a developer integrating the JWT middleware, I need to configure both HS256 (symmetric) and RS256 (asymmetric) validators simultaneously during middleware initialization, so that my service can validate JWTs from multiple token issuers using different signing algorithms without requiring separate middleware instances.

**Why this priority**: This is the core capability - enabling algorithm-agnostic token validation. Without this, the middleware cannot dynamically handle tokens with different algorithms, which is essential for microservices architectures where different services may issue tokens with different signing methods.

**Independent Test**: Initialize middleware with both HS256 secret and RS256 public key, send tokens with `alg: HS256` header and verify validation succeeds with HMAC verification, send tokens with `alg: RS256` header and verify validation succeeds with RSA verification.

**Acceptance Scenarios**:

1. **Given** middleware is initialized with both HS256 secret (32+ bytes) and RS256 public key, **When** a valid JWT with `alg: HS256` header is presented, **Then** the middleware validates the token using HMAC-SHA256 signature verification and allows the request to proceed
2. **Given** middleware is initialized with both HS256 secret and RS256 public key, **When** a valid JWT with `alg: RS256` header is presented, **Then** the middleware validates the token using RSA-SHA256 signature verification and allows the request to proceed
3. **Given** middleware is initialized with only HS256 secret (no RS256 key provided), **When** a JWT with `alg: RS256` header is presented, **Then** the middleware rejects the token with "unsupported algorithm" error
4. **Given** middleware is initialized with only RS256 public key (no HS256 secret provided), **When** a JWT with `alg: HS256` header is presented, **Then** the middleware rejects the token with "unsupported algorithm" error
5. **Given** middleware is initialized with both validators, **When** a JWT with `alg: none` header is presented, **Then** the middleware rejects the token with "none algorithm not allowed" error

---

### User Story 2 - Algorithm-Based Routing (Priority: P2)

As a security engineer monitoring authentication events, I need the middleware to log which algorithm was used for each validation attempt, so that I can audit token sources and detect potential security issues like unexpected algorithm usage.

**Why this priority**: Extends the core capability with observability. Security teams need to track which algorithms are being used in production to detect anomalies, unauthorized token issuers, or algorithm downgrade attacks.

**Independent Test**: Enable structured logging, send tokens with different `alg` headers (HS256, RS256, unsupported), verify that all authentication events (success and failure) include the algorithm type in the log output.

**Acceptance Scenarios**:

1. **Given** logging is enabled and middleware successfully validates an HS256 token, **When** the authentication event is logged, **Then** the log entry includes `algorithm: HS256` field
2. **Given** logging is enabled and middleware successfully validates an RS256 token, **When** the authentication event is logged, **Then** the log entry includes `algorithm: RS256` field
3. **Given** logging is enabled and middleware rejects a token with unsupported algorithm, **When** the authentication failure is logged, **Then** the log entry includes `algorithm: [unsupported_alg]` and `failure_reason: UNSUPPORTED_ALGORITHM`
4. **Given** logging is enabled and middleware encounters a malformed `alg` header (non-string value), **When** the authentication failure is logged, **Then** the log entry includes `failure_reason: MALFORMED_ALGORITHM_HEADER`

---

### User Story 3 - Validation Error Clarity (Priority: P3)

As a frontend developer debugging authentication failures, I need clear error responses that distinguish between "unsupported algorithm" vs "invalid signature" vs "expired token", so that I can quickly identify whether the issue is token source misconfiguration, signing key mismatch, or timing problems.

**Why this priority**: Improves developer experience during integration and troubleshooting. Clear error messages reduce time spent debugging authentication issues.

**Independent Test**: Send tokens with various failure conditions (unsupported alg, wrong signature, expired, none algorithm), verify each returns a distinct error code in the 401 response.

**Acceptance Scenarios**:

1. **Given** middleware is initialized with HS256 only, **When** a token with `alg: ES256` is presented, **Then** the response includes error code `UNSUPPORTED_ALGORITHM` and message "algorithm ES256 not supported (available: HS256)"
2. **Given** middleware is initialized with both algorithms, **When** a token with `alg: HS384` is presented, **Then** the response includes error code `UNSUPPORTED_ALGORITHM` and message "algorithm HS384 not supported (available: HS256, RS256)"
3. **Given** an HS256 token with correct algorithm but invalid signature, **When** validation fails, **Then** the response includes error code `INVALID_SIGNATURE` (not `UNSUPPORTED_ALGORITHM`)
4. **Given** an RS256 token with correct algorithm but expired timestamp, **When** validation fails, **Then** the response includes error code `EXPIRED` (not `UNSUPPORTED_ALGORITHM`)

---

### Edge Cases

- What happens when a token has `alg: HS256` in header but was actually signed with RS256 private key? (Algorithm confusion attack - should fail signature verification with INVALID_SIGNATURE)
- How does the system handle a token with uppercase algorithm like `alg: "HS256"` vs lowercase `alg: "hs256"`? (Should normalize to uppercase and validate, or reject non-standard casing)
- What happens when middleware is initialized with neither HS256 nor RS256 validators? (Configuration validation should fail at initialization time before accepting requests)
- How does system handle missing `alg` header in JWT? (Should reject as MALFORMED token)
- What happens when `alg` header is an array like `["HS256", "RS256"]` instead of a string? (Should reject as MALFORMED_ALGORITHM_HEADER)
- How does system behave when both HS256 and RS256 are configured with the same keys/secrets? (Configuration error - RSA public key cannot be used for HMAC and vice versa)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Middleware initialization MUST accept both HS256 secret key ([]byte) and RS256 public key (*rsa.PublicKey) simultaneously via configuration options
- **FR-002**: Middleware initialization MUST allow partial configuration (only HS256, only RS256, or both) and validate that at least one algorithm is configured
- **FR-003**: Middleware MUST extract the `alg` field from the JWT header before validation
- **FR-004**: Middleware MUST inspect the `alg` field value and route to the appropriate validator (HS256 or RS256) based on the algorithm specified in the token
- **FR-005**: Middleware MUST reject tokens with `alg` values that do not match any configured validators with error code `UNSUPPORTED_ALGORITHM`
- **FR-006**: Middleware MUST reject tokens with `alg: none` or `alg: None` or `alg: NONE` explicitly, regardless of which algorithms are configured
- **FR-007**: Middleware MUST perform algorithm-specific signature verification using the correct validator (HMAC-SHA256 for HS256, RSA-SHA256 for RS256)
- **FR-008**: Middleware MUST return distinct error codes for different failure types: `UNSUPPORTED_ALGORITHM`, `INVALID_SIGNATURE`, `EXPIRED`, `MALFORMED`, `MALFORMED_ALGORITHM_HEADER`
- **FR-009**: Error responses for unsupported algorithms MUST include a list of available algorithms (e.g., "algorithm ES256 not supported (available: HS256, RS256)")
- **FR-010**: Security event logs MUST include the algorithm type used (or attempted) for each authentication event
- **FR-011**: Middleware MUST prevent algorithm confusion attacks by ensuring the token's `alg` header matches the signing method used (e.g., HS256 token cannot pass RS256 validation)
- **FR-012**: Configuration validation MUST fail at initialization if neither HS256 nor RS256 is configured (before processing any requests)
- **FR-013**: Middleware MUST treat algorithm matching as case-sensitive (e.g., `HS256` is valid, `hs256` is invalid) per JWT specification RFC 7519

### Key Entities

- **Algorithm Configuration**: Represents the set of configured validators with their respective keys/secrets (HS256 secret bytes, RS256 public key). Each algorithm is either enabled (key provided) or disabled (key not provided).
- **Token Header**: Represents the JWT header containing the `alg` field which determines validation routing. Must be extracted and validated before signature verification.
- **Validation Route**: Represents the decision path from `alg` header value to specific validator (HMAC vs RSA). Determines which cryptographic operation is performed.
- **Security Event**: Enhanced to include algorithm type field for audit trails and anomaly detection.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can configure middleware with both HS256 and RS256 validators in under 5 lines of configuration code
- **SC-002**: Middleware correctly validates 100% of tokens with supported algorithms (HS256 or RS256) without algorithm-related errors
- **SC-003**: Middleware rejects 100% of tokens with unsupported algorithms (e.g., ES256, HS384, none) with appropriate error codes
- **SC-004**: Algorithm routing adds less than 10 microseconds overhead to token validation latency (negligible performance impact)
- **SC-005**: Error messages for unsupported algorithms include the list of available algorithms in 100% of cases
- **SC-006**: Security logs capture algorithm type for 100% of authentication attempts (success and failure)
- **SC-007**: Algorithm confusion attacks (HS256 token presented to RS256 validator) are detected and rejected in 100% of attempts
- **SC-008**: Configuration errors (neither algorithm configured) are caught at initialization time before processing any requests (zero production incidents)

## Assumptions *(mandatory)*

- **A-001**: Developers using this middleware understand JWT structure (header, payload, signature) and algorithm differences (symmetric HMAC vs asymmetric RSA)
- **A-002**: Token issuers correctly set the `alg` header field according to the signing method used (standard JWT practice)
- **A-003**: Both HS256 secrets and RS256 public keys are securely managed and provided to middleware at initialization time (not dynamically rotated during runtime)
- **A-004**: Supported algorithms are limited to HS256 and RS256 only - other algorithms (ES256, HS384, etc.) are intentionally out of scope for this update
- **A-005**: Algorithm routing decision is based solely on the JWT `alg` header field, not on external configuration or request context
- **A-006**: The existing middleware architecture supports extending the validator to handle multiple algorithms without breaking existing functionality
- **A-007**: Performance requirements remain unchanged - algorithm routing must not significantly increase validation latency (target <1ms p99 maintained)

## Constraints *(mandatory)*

- **C-001**: Must maintain backward compatibility with existing single-algorithm configuration (WithHS256 or WithRS256 alone)
- **C-002**: Cannot add external dependencies beyond what's already included (golang-jwt/jwt v5, Go standard library)
- **C-003**: Must maintain the same middleware interface for Gin and gRPC (no breaking API changes)
- **C-004**: Algorithm routing logic must be synchronous (no async operations or goroutines)
- **C-005**: Configuration must remain immutable after initialization (consistent with existing design)
- **C-006**: Must support Go 1.23+ (no additional version requirements)
- **C-007**: All validation logic must remain in a single request context (no cross-request state sharing)

## Security Considerations *(mandatory)*

- **SEC-001**: Algorithm confusion attacks MUST be prevented by enforcing that the `alg` header matches the actual signing method used
- **SEC-002**: The `none` algorithm MUST be explicitly rejected regardless of configuration
- **SEC-003**: Algorithm routing MUST NOT leak information about which algorithms are configured via timing attacks (constant-time algorithm checks)
- **SEC-004**: Error messages MUST NOT expose sensitive information like secret keys or private key details
- **SEC-005**: Unsupported algorithm attempts MUST be logged as potential security events (anomaly detection)
- **SEC-006**: Token header parsing MUST sanitize and validate the `alg` field type and value before routing
- **SEC-007**: Middleware MUST continue to validate all other token aspects (expiration, signature, claims) after algorithm routing
