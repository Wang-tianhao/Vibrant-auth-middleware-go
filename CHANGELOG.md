# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-11-09

### Added

- **Dual-Algorithm Support**: Middleware can now accept both HS256 and RS256 tokens simultaneously
  - Configure with `NewConfig(WithHS256(secret), WithRS256(publicKey))`
  - Zero-allocation algorithm routing (<10ns overhead)
  - Use case: Accept tokens from multiple issuers using different signing methods

- **Enhanced Error Reporting**:
  - New error code: `UNSUPPORTED_ALGORITHM` - returned when token uses an algorithm not configured
  - New error code: `MALFORMED_ALGORITHM_HEADER` - returned when `alg` field is malformed
  - Error responses now include available algorithms: `"algorithm ES256 not supported (available: HS256, RS256)"`
  - Clear distinction between error types (unsupported algorithm vs invalid signature vs expired)

- **Algorithm Logging**:
  - `SecurityEvent` struct now includes `Algorithm` field for audit trails
  - All authentication events (success and failure) log the algorithm used
  - Enables security monitoring for algorithm anomaly detection

- **New Public API**:
  - `Config.AvailableAlgorithms()` - returns sorted list of configured algorithms
  - Enhanced validation error messages with actionable details

- **Performance**:
  - Comprehensive benchmark suite added
  - Algorithm routing: 8.6ns (zero allocations)
  - HS256 validation: 1.7μs average
  - RS256 validation: 22μs average
  - No performance regression for single-algorithm configs

- **Security**:
  - Algorithm confusion attack prevention (explicit validation)
  - Case-sensitive algorithm matching per RFC 7519
  - "none" algorithm explicitly rejected (all variants: none, None, NONE)
  - Enhanced security test suite

### Changed

- **Internal**: `Config` struct now uses validators map instead of single algorithm field (backward compatible)
- **Logging**: Security events now include algorithm metadata for all events

### Deprecated

- `Config.Algorithm()` - use `Config.AvailableAlgorithms()` instead (still functional, returns first algorithm)
- `Config.SigningKey()` - use algorithm-specific accessors (still functional, returns first key)

### Fixed

- Improved error unwrapping for JWT library errors (handles wrapped ValidationErrors correctly)

### Migration Guide (v1.x → v2.0)

**No breaking changes!** Existing code works unchanged:

```go
// v1.x code (still works in v2.0)
cfg, _ := jwtauth.NewConfig(jwtauth.WithHS256(secret))

// v2.0 new feature (optional upgrade)
cfg, _ := jwtauth.NewConfig(
    jwtauth.WithHS256(secret),      // Accept internal HS256 tokens
    jwtauth.WithRS256(publicKey),   // Accept external RS256 tokens
)
```

**What's backward compatible:**
- ✅ Single-algorithm configs work identically
- ✅ Error codes unchanged for existing scenarios
- ✅ Middleware signatures unchanged
- ✅ Claims extraction unchanged
- ✅ Performance maintained (no regression)

**New features (optional):**
- ✨ Add RS256 support alongside existing HS256 (or vice versa)
- ✨ Use enhanced error messages with available algorithms
- ✨ Access algorithm metadata in logs for security monitoring

## [1.0.0] - 2025-11-08

### Added

- Initial release with JWT authentication middleware
- HS256 (HMAC-SHA256) support
- RS256 (RSA-SHA256) support
- Gin HTTP middleware
- gRPC interceptors
- Structured security logging
- Custom claims support
- Cookie-based token extraction
- Clock skew tolerance

---

**Legend:**
- 🎯 Major feature
- ✨ Enhancement
- 🐛 Bug fix
- 🔒 Security
- ⚡ Performance
- 📝 Documentation
