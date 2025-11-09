<!--
  Sync Impact Report:
  Version: 0.0.0 → 1.0.0

  Changes:
  - Initial constitution created for Golang auth middleware project
  - All 7 core principles defined: Go Idioms, Security-First, Middleware Design, Testing, Performance, Documentation, Versioning
  - Security Requirements section added
  - Development Workflow section added
  - Governance rules established

  Modified Principles: N/A (initial creation)
  Added Sections: All sections (initial creation)
  Removed Sections: None

  Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section aligned with Go principles
  ✅ spec-template.md - User stories and requirements compatible with middleware patterns
  ✅ tasks-template.md - Task organization supports Go project structure and testing practices

  Follow-up TODOs: None
-->

# Vibrant Auth Middleware Go Constitution

## Core Principles

### I. Go Idioms (NON-NEGOTIABLE)

All code MUST follow Go idiomatic best practices:

- **Effective Go compliance**: Code follows guidelines from Effective Go and official Go documentation
- **Go proverbs adherence**: Simple, readable, explicit code over clever abstractions
- **Error handling**: Explicit error returns; NO panic in library code except for truly unrecoverable conditions
- **Interface discipline**: Accept interfaces, return concrete types; keep interfaces small and focused
- **Naming conventions**: Use MixedCaps for exported names, camelCase for unexported; names should be concise and descriptive
- **Package design**: Flat package structure preferred; packages organized by functionality, not by layer
- **Zero values**: Types must be usable with their zero values when practical
- **Context propagation**: Context.Context MUST be first parameter in functions that accept it

**Rationale**: Go has established conventions that make code immediately readable to any Go developer. Deviation creates cognitive overhead and maintenance burden.

### II. Security-First (NON-NEGOTIABLE)

Security is the primary concern for authentication middleware:

- **OWASP compliance**: All implementations MUST address OWASP Top 10 vulnerabilities
- **Input validation**: ALL external input validated and sanitized before processing
- **Secure defaults**: Configuration defaults MUST be secure; insecure options require explicit opt-in
- **Dependency hygiene**: NO dependencies with known CVEs; automated vulnerability scanning required
- **Secret management**: NO hardcoded secrets; all sensitive data via environment or secure stores
- **Timing attacks**: Use constant-time comparisons for sensitive operations (crypto/subtle)
- **Token security**: JWT/session tokens follow industry standards (HMAC-SHA256 minimum, secure random generation)
- **Rate limiting**: Built-in rate limiting for authentication attempts

**Rationale**: Auth middleware is a security boundary. A single vulnerability can compromise entire systems.

### III. Middleware Design Patterns

Middleware MUST follow Go HTTP middleware conventions:

- **Standard signature**: Middleware returns `func(http.Handler) http.Handler`
- **Chain compatibility**: Compatible with standard library and common routers (chi, mux, echo, gin)
- **Context injection**: User/auth data stored in request context using typed keys
- **Early termination**: Invalid auth returns immediately without calling next handler
- **Non-invasive**: Minimal performance overhead; no global state modifications
- **Composability**: Middleware components can be chained independently
- **Configuration**: Options pattern for middleware configuration

**Rationale**: Standard middleware patterns ensure compatibility across frameworks and predictable behavior.

### IV. Testing (NON-NEGOTIABLE)

Comprehensive testing is mandatory:

- **Test-Driven Development**: Write tests BEFORE implementation; verify tests fail before coding
- **Coverage minimum**: 80% code coverage required; security-critical paths MUST have 100% coverage
- **Test types required**:
  - **Unit tests**: All public functions and methods
  - **Integration tests**: Middleware chain behavior, auth flows end-to-end
  - **Security tests**: Attack scenarios (replay, injection, tampering, timing attacks)
  - **Benchmark tests**: Performance-critical paths (token validation, crypto operations)
- **Table-driven tests**: Preferred for testing multiple scenarios
- **Test helpers**: Use testify/assert or similar; keep test code DRY
- **Mocking**: Use interfaces for external dependencies; prefer standard library testing.T over frameworks

**Rationale**: Auth middleware has zero tolerance for bugs. Comprehensive testing is the only way to ensure reliability and catch regressions.

### V. Performance & Efficiency

Authentication MUST NOT become a bottleneck:

- **Latency targets**: Token validation <1ms p99; middleware overhead <100μs p99
- **Zero allocations**: Hot paths (token validation) should minimize or eliminate allocations
- **Concurrent safety**: All shared state MUST be safe for concurrent access
- **Resource management**: Proper cleanup of resources; no goroutine leaks
- **Benchmarking**: Performance-critical code MUST include benchmark tests
- **Profiling awareness**: Code should be profile-friendly (cpu, memory, blocking)

**Rationale**: Every request passes through auth middleware. Poor performance multiplies across the entire system.

### VI. Documentation & Developer Experience

Clear documentation is essential for security software:

- **Package documentation**: Every exported package has comprehensive package-level godoc
- **Function documentation**: All exported functions/types documented per godoc conventions
- **Example tests**: Example_* test functions for common use cases
- **README completeness**: Installation, quickstart, configuration reference, security considerations
- **Migration guides**: Breaking changes require migration documentation
- **Security advisories**: Security issues disclosed per responsible disclosure practices
- **Code comments**: Complex security logic MUST include rationale comments

**Rationale**: Misconfigured auth middleware is a security vulnerability. Clear documentation prevents misuse.

### VII. Versioning & Backward Compatibility

Stable API ensures safe upgrades:

- **Semantic versioning**: Strict semver 2.0 compliance
- **Go modules**: Proper go.mod versioning; major versions in import paths (v2+)
- **Deprecation policy**: 2 minor versions notice before removal; clear migration path
- **Breaking changes**: Only in major versions; deprecated patterns retained when safe
- **API stability**: v1+ APIs are stable; experimental features clearly marked
- **Changelog**: Keep comprehensive CHANGELOG.md with security-relevant changes highlighted

**Rationale**: Auth middleware is foundational infrastructure. Breaking changes ripple through entire systems.

## Security Requirements

### Cryptography Standards

- **NO custom crypto**: Use only crypto/* standard library or vetted third-party (e.g., golang.org/x/crypto)
- **Algorithm minimums**:
  - Hashing: SHA-256 or better
  - Symmetric encryption: AES-256-GCM
  - Asymmetric: RSA-2048/ECDSA P-256 minimum
  - JWT signing: HMAC-SHA256/RS256/ES256 only
- **Random generation**: crypto/rand only; NEVER math/rand for security tokens
- **Key rotation**: Support key rotation for long-lived deployments

### Audit & Logging

- **Security events**: Log all auth failures, token validation failures, configuration errors
- **PII protection**: NO passwords, tokens, or sensitive data in logs
- **Structured logging**: Use structured logging format (JSON) for security events
- **Audit trail**: Include request ID, timestamp, user identifier (when available), action taken

## Development Workflow

### Code Quality Gates

- **Linting**: golangci-lint with strict config; zero warnings policy
- **Formatting**: gofmt + goimports; enforced in CI
- **Vetting**: go vet MUST pass; staticcheck recommended
- **Security scanning**: gosec in CI pipeline
- **Dependency checks**: govulncheck in CI; dependabot or renovate enabled

### Review Process

- **Peer review**: All changes require code review approval
- **Security review**: Changes to crypto, auth logic, or security-critical paths require security-focused review
- **Breaking changes**: Require explicit design review and documentation
- **Test validation**: CI MUST pass; reviewer verifies test coverage

### Contribution Standards

- **Feature branches**: Work in feature branches; clean commit history
- **Commit messages**: Conventional commits format preferred (feat:, fix:, security:, breaking:)
- **Small PRs**: Keep PRs focused and reviewable (<400 lines when possible)
- **Documentation updates**: Code changes MUST include relevant doc updates

## Governance

This constitution supersedes all other development practices and preferences.

### Amendment Process

- **Proposal**: Amendments require written proposal with rationale
- **Review**: Team review with minimum 3-day comment period
- **Approval**: Unanimous approval for NON-NEGOTIABLE principles; majority for others
- **Migration plan**: Breaking amendments require migration guide
- **Version bump**: Minor version for additions; major for removals or redefinitions

### Versioning Policy

- **MAJOR**: Backward incompatible governance changes; principle removals/redefinitions
- **MINOR**: New principles added; expanded guidance; new mandatory practices
- **PATCH**: Clarifications; wording improvements; typo fixes

### Compliance Review

- **PR reviews**: Every PR MUST verify compliance with this constitution
- **Periodic audits**: Quarterly codebase audits against principles
- **Violation handling**: Constitution violations block PR approval; require fix or amendment proposal
- **Complexity justification**: Deviations from simplicity principle MUST be documented in plan.md Complexity Tracking section

### Development Guidance

For runtime development guidance and detailed command workflows, see `.claude/commands/speckit.*.md` files.

**Version**: 1.0.0 | **Ratified**: 2025-11-09 | **Last Amended**: 2025-11-09
