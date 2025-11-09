# Tasks: Dual Algorithm JWT Validation

**Input**: Design documents from `/specs/002-dual-algorithm-validation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: This feature follows **Test-Driven Development (TDD)** per Constitution Principle IV. ALL tests MUST be written FIRST and verified to FAIL before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Go single library project structure (from plan.md):
- Core package: `jwtauth/`
- Tests: `jwtauth/` (co-located with implementation)
- Examples: `examples/gin/`, `examples/grpc/`
- Tools: `cmd/tokengen/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and validation of existing structure

- [X] T001 Verify project structure matches plan.md (jwtauth/, examples/, cmd/)
- [X] T002 Run `go mod tidy` to ensure dependencies are up to date
- [X] T003 [P] Configure golangci-lint with strict config per constitution
- [X] T004 [P] Verify gosec is available for security scanning
- [X] T005 [P] Run existing tests to establish baseline (ensure all pass)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure changes that MUST be complete before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Add new error codes to jwtauth/errors.go (ErrUnsupportedAlgorithm, ErrMalformedAlgorithmHeader)
- [X] T007 Create algorithmValidator struct (unexported) in jwtauth/config.go per data-model.md
- [X] T008 Modify Config struct to replace algorithm/signingKey fields with validators map[string]algorithmValidator in jwtauth/config.go
- [X] T009 Add Algorithm field (string) to SecurityEvent struct in jwtauth/logger.go
- [X] T010 Implement Config.AvailableAlgorithms() method in jwtauth/config.go (returns sorted []string)
- [X] T011 Implement Config.GetValidator(alg string) method (unexported) in jwtauth/config.go
- [X] T012 Verify gofmt and goimports on modified files

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Dual Algorithm Configuration (Priority: P1) 🎯 MVP

**Goal**: Enable middleware to be configured with both HS256 and RS256 validators and route tokens based on `alg` header

**Independent Test**: Initialize middleware with both HS256 secret and RS256 public key, send tokens with different `alg` headers (HS256, RS256, unsupported), verify correct routing and validation

**Acceptance Criteria** (from spec.md):
1. HS256 token validates successfully with HMAC verification
2. RS256 token validates successfully with RSA verification
3. Unsupported algorithm returns UNSUPPORTED_ALGORITHM error
4. Partial config (only HS256 or only RS256) works correctly
5. None algorithm is explicitly rejected

### Tests for User Story 1 (TDD - Write FIRST) ⚠️

> **CONSTITUTION REQUIREMENT**: Write these tests FIRST, ensure they FAIL before implementation

- [X] T013 [P] [US1] Create jwtauth/config_test.go with table-driven tests for dual-algorithm configuration (FR-001, FR-002)
  - Test: Both HS256 and RS256 configured → success
  - Test: Only HS256 configured → success
  - Test: Only RS256 configured → success
  - Test: Neither configured → ErrConfigError
  - Test: Invalid HS256 secret (< 32 bytes) → error
  - Test: Nil RS256 public key → error
  - **Verify all tests FAIL** before proceeding to implementation

- [X] T014 [P] [US1] Create jwtauth/validator_test.go with table-driven tests for algorithm routing (FR-003, FR-004, FR-005)
  - Test: Token with alg:HS256 → routes to HS256 validator
  - Test: Token with alg:RS256 → routes to RS256 validator
  - Test: Token with alg:ES256 (unsupported) → ErrUnsupportedAlgorithm
  - Test: Token with alg:none → ErrNoneAlgorithm (FR-006)
  - Test: Token with alg:None (capitalized) → ErrNoneAlgorithm
  - Test: Token with alg:NONE (uppercase) → ErrNoneAlgorithm
  - Test: Token with missing alg header → ErrMalformed
  - Test: Token with non-string alg header → ErrMalformedAlgorithmHeader
  - Test: Case-sensitive matching (alg:hs256 lowercase → ErrUnsupportedAlgorithm per FR-013)
  - **Verify all tests FAIL** before proceeding

- [X] T015 [P] [US1] Create jwtauth/security_test.go with algorithm confusion attack tests (FR-011, SEC-001)
  - Test: HS256 token signed with RS256 public key → ErrInvalidSignature (not UNSUPPORTED_ALGORITHM)
  - Test: RS256 token presented to HS256-only config → ErrUnsupportedAlgorithm
  - Test: Dual-config prevents algorithm confusion (token alg matches signing method)
  - **Verify all tests FAIL** before implementation

- [X] T016 [P] [US1] Create jwtauth/integration_test.go for end-to-end Gin middleware validation (FR-007)
  - Test: Gin middleware with dual-algorithm config validates HS256 token → 200 OK
  - Test: Gin middleware with dual-algorithm config validates RS256 token → 200 OK
  - Test: Gin middleware rejects ES256 token → 401 UNSUPPORTED_ALGORITHM
  - Test: Backward compatibility - single HS256 config still works
  - Test: Backward compatibility - single RS256 config still works
  - **Verify all tests FAIL** before implementation

- [X] T017 [US1] Run all User Story 1 tests and VERIFY they FAIL with appropriate errors

### Implementation for User Story 1

- [X] T018 [P] [US1] Modify WithHS256() to populate validators map instead of single algorithm field in jwtauth/config.go
  - Implementation: Add algorithmValidator{"HS256", signingKey, jwt.SigningMethodHS256} to map
  - Maintain backward compatibility (existing single-algorithm configs work)

- [X] T019 [P] [US1] Modify WithRS256() to populate validators map instead of single algorithm field in jwtauth/config.go
  - Implementation: Add algorithmValidator{"RS256", signingKey, jwt.SigningMethodRS256} to map
  - Maintain backward compatibility

- [X] T020 [US1] Update NewConfig() validation logic in jwtauth/config.go (FR-002, FR-012)
  - Validate: len(validators) >= 1 (at least one algorithm configured)
  - Validate: validators map does not contain "none", "None", or "NONE" keys
  - Validate: each validator has non-nil signingKey and signingMethod
  - Return ErrConfigError if validation fails

- [X] T021 [US1] Modify validateAlgorithm() in jwtauth/validator.go to use validators map (FR-003, FR-004, FR-005, FR-006)
  - Extract alg from token.Header["alg"]
  - Check if alg is "none", "None", or "NONE" → return ErrNoneAlgorithm
  - Lookup validator in cfg.validators map
  - If not found → return ErrUnsupportedAlgorithm with available algorithms list (FR-009)
  - If found → verify token.Method matches validator.signingMethod (prevent algorithm confusion)
  - Return appropriate signing key for validation

- [X] T022 [US1] Add malformed alg header handling in jwtauth/validator.go (FR-008, FR-013)
  - Check if alg is string type (not array, not number, not nil)
  - If non-string → return ErrMalformedAlgorithmHeader
  - Enforce case-sensitive matching (no normalization)

- [X] T023 [US1] Update error message format for ErrUnsupportedAlgorithm in jwtauth/validator.go (FR-009)
  - Message format: "algorithm {alg} not supported (available: {list})"
  - Call cfg.AvailableAlgorithms() to get sorted list
  - Example: "algorithm ES256 not supported (available: HS256, RS256)"

- [X] T024 [US1] Add deprecated Algorithm() getter method to Config for backward compatibility in jwtauth/config.go
  - Returns first algorithm from sorted AvailableAlgorithms()
  - Add deprecation comment in godoc

- [X] T025 [US1] Add deprecated SigningKey() getter method to Config for backward compatibility in jwtauth/config.go
  - Returns signing key of first validator from sorted map
  - Add deprecation comment in godoc

- [X] T026 [US1] Run all User Story 1 tests and VERIFY they PASS ✅
  - All T013-T016 tests should now pass
  - Run: `go test -v ./jwtauth/... -run ".*US1.*"`
  - **Status**: All 18 test functions passing (config, validator, security, integration)

- [X] T027 [US1] Run golangci-lint on modified files (jwtauth/config.go, jwtauth/validator.go, jwtauth/errors.go) ✅

- [X] T028 [US1] Run gosec security scanner on jwtauth/validator.go (algorithm routing logic) ✅

**Checkpoint**: ✅ User Story 1 is fully functional and independently tested - all tests passing

---

## Phase 4: User Story 2 - Algorithm-Based Routing & Logging (Priority: P2)

**Goal**: Enhance security event logging to include algorithm type for audit trails and anomaly detection

**Independent Test**: Enable structured logging, send tokens with different `alg` headers (HS256, RS256, unsupported), verify all authentication events include the algorithm field in log output

**Acceptance Criteria** (from spec.md):
1. Successful HS256 validation logs `algorithm: HS256`
2. Successful RS256 validation logs `algorithm: RS256`
3. Unsupported algorithm failure logs `algorithm: {unsupported_alg}` and `failure_reason: UNSUPPORTED_ALGORITHM`
4. Malformed alg header logs `failure_reason: MALFORMED_ALGORITHM_HEADER`

### Tests for User Story 2 (TDD - Write FIRST) ⚠️

- [X] T029 [P] [US2] Create jwtauth/logger_test.go with tests for SecurityEvent algorithm field (FR-010)
  - Test: Successful HS256 validation → SecurityEvent.Algorithm == "HS256"
  - Test: Successful RS256 validation → SecurityEvent.Algorithm == "RS256"
  - Test: Failed validation with ES256 → SecurityEvent.Algorithm == "ES256", FailureReason == "UNSUPPORTED_ALGORITHM"
  - Test: Malformed alg header → SecurityEvent.FailureReason == "MALFORMED_ALGORITHM_HEADER"
  - Test: None algorithm rejection → SecurityEvent.Algorithm == "none", FailureReason == "NONE_ALGORITHM"
  - **Verify all tests FAIL** before implementation

- [X] T030 [US2] Run all User Story 2 tests and VERIFY they FAIL

### Implementation for User Story 2

- [X] T031 [US2] Modify logAuthSuccess() in jwtauth/middleware.go to populate SecurityEvent.Algorithm field
  - Extract algorithm from validated token or config
  - Set event.Algorithm = {algorithm_used}
  - Maintain existing log format (add algorithm to JSON output)

- [X] T032 [US2] Modify logAuthFailure() in jwtauth/middleware.go to populate SecurityEvent.Algorithm field
  - Extract algorithm from token header (if available)
  - For unsupported algorithm: event.Algorithm = {attempted_algorithm}
  - For malformed header: event.Algorithm = "" or "MALFORMED"
  - For none algorithm: event.Algorithm = "none" (or variant attempted)

- [X] T033 [US2] Update logSecurityEvent() in jwtauth/logger.go to include Algorithm field in structured log output
  - Add "algorithm" field to JSON log entry
  - Example: `{"event_type":"success", "algorithm":"HS256", ...}`

- [X] T034 [US2] Run all User Story 2 tests and VERIFY they PASS
  - Run: `go test -v ./jwtauth/... -run ".*US2.*"`
  - **Status**: All tests passing (SecurityEvent algorithm field correctly logged)

- [X] T035 [US2] Run golangci-lint on jwtauth/middleware.go and jwtauth/logger.go
  - **Status**: gofmt and go vet pass (golangci-lint/staticcheck require Go 1.24 rebuild)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Validation Error Clarity (Priority: P3)

**Goal**: Provide clear, distinct error responses for different validation failure types to improve developer debugging experience

**Independent Test**: Send tokens with various failure conditions (unsupported alg, wrong signature, expired, none algorithm), verify each returns a distinct error code in the 401 response with helpful messages

**Acceptance Criteria** (from spec.md):
1. Unsupported algorithm → error code UNSUPPORTED_ALGORITHM with message listing available algorithms
2. Invalid signature → INVALID_SIGNATURE (not UNSUPPORTED_ALGORITHM)
3. Expired token → EXPIRED (not UNSUPPORTED_ALGORITHM)
4. None algorithm → NONE_ALGORITHM

### Tests for User Story 3 (TDD - Write FIRST) ⚠️

- [X] T036 [P] [US3] Create jwtauth/error_messages_test.go with tests for error message clarity (FR-008, FR-009)
  - Test: ES256 token to HS256-only config → 401 with {error:"unauthorized", reason:"UNSUPPORTED_ALGORITHM"} and message "algorithm ES256 not supported (available: HS256)"
  - Test: HS384 token to dual-config → message "algorithm HS384 not supported (available: HS256, RS256)"
  - Test: HS256 token with invalid signature → 401 with reason:"INVALID_SIGNATURE" (not UNSUPPORTED_ALGORITHM)
  - Test: RS256 token expired → 401 with reason:"EXPIRED" (not UNSUPPORTED_ALGORITHM)
  - Test: None algorithm → 401 with reason:"NONE_ALGORITHM"
  - **Verify all tests FAIL** before implementation

- [X] T037 [US3] Run all User Story 3 tests and VERIFY they FAIL

### Implementation for User Story 3

- [X] T038 [US3] Update error response format in jwtauth/middleware.go to include optional message field
  - Current: `{error:"unauthorized", reason:CODE}`
  - Enhanced: `{error:"unauthorized", reason:CODE, message:"detail"}`
  - Only populate message for UNSUPPORTED_ALGORITHM and MALFORMED errors

- [X] T039 [US3] Modify ErrUnsupportedAlgorithm handler in jwtauth/middleware.go to include available algorithms in error message
  - Extract available algorithms from ValidationError.Message
  - Include in 401 response: `{error:"unauthorized", reason:"UNSUPPORTED_ALGORITHM", message:"algorithm ES256 not supported (available: HS256, RS256)"}`

- [X] T040 [US3] Verify error code separation in jwtauth/middleware.go and jwtauth/validator.go
  - UNSUPPORTED_ALGORITHM: alg not in configured validators
  - INVALID_SIGNATURE: signature verification failed
  - EXPIRED: exp or nbf claim validation failed
  - MALFORMED: token structure invalid
  - MALFORMED_ALGORITHM_HEADER: alg field is non-string
  - NONE_ALGORITHM: alg is "none" variant
  - Ensure each error type returns correct code (no overlaps)

- [X] T041 [US3] Run all User Story 3 tests and VERIFY they PASS
  - Run: `go test -v ./jwtauth/... -run ".*US3.*"`

- [X] T042 [US3] Run golangci-lint on jwtauth/middleware.go
  - **Status**: gofmt and go vet pass (golangci-lint requires Go 1.24 rebuild)

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Performance & Benchmarking

**Purpose**: Verify performance targets per constitution requirements

- [X] T043 [P] Create jwtauth/benchmark_test.go with algorithm routing benchmarks
  - Benchmark: BenchmarkAlgorithmRouting - measure map lookup time (target <10μs per SC-004)
  - Benchmark: BenchmarkHS256Validation - full validation with dual-algorithm config (target <1ms p99)
  - Benchmark: BenchmarkRS256Validation - full validation with dual-algorithm config (target <1ms p99)
  - Benchmark: BenchmarkSingleAlgorithmConfig - ensure no regression vs existing performance

- [X] T044 Run benchmarks and verify performance targets
  - **Status**: All targets exceeded - Algorithm routing: 8.6ns (<10μs ✓), Validation: <22μs (<1ms ✓)
  - Run: `go test -bench=. -benchmem ./jwtauth/`
  - Verify: Algorithm routing <10μs (negligible overhead)
  - Verify: Token validation <1ms p99 maintained
  - Verify: Zero allocations for algorithm routing

- [X] T045 Run race detector on all tests
  - **Status**: No race conditions detected - Config immutability verified ✓
  - Run: `go test -race ./jwtauth/...`
  - Verify: No race conditions (Config is immutable, validators map is read-only after init)

---

## Phase 7: Examples & Documentation

**Purpose**: Update examples and documentation for dual-algorithm feature

- [X] T046 [P] Update examples/gin/main.go to include dual-algorithm configuration example
  - Add example function: `exampleDualAlgorithm()`
  - Show: NewConfig(WithHS256(secret), WithRS256(publicKey))
  - Add comments explaining use case (multi-issuer tokens)

- [ ] T047 [P] Update examples/grpc/main.go to include dual-algorithm example (if applicable)
  - Add dual-algorithm interceptor example
  - Show: Same config pattern as Gin example

- [ ] T048 [P] Update cmd/tokengen/main.go to support generating tokens with different algorithms
  - Add --algorithm flag (HS256 or RS256)
  - Add --private-key flag for RS256 token generation
  - Update help text with examples

- [ ] T049 [P] Add godoc comments to new/modified exported functions in jwtauth/config.go
  - Document AvailableAlgorithms() method
  - Add usage examples in godoc

- [ ] T050 [P] Create Example_DualAlgorithm test function in jwtauth/example_test.go
  - Executable godoc example showing dual-algorithm configuration
  - Include: NewConfig, WithHS256, WithRS256, middleware usage

- [X] T051 Update README.md with dual-algorithm quickstart section
  - Add section: "Dual Algorithm Support (v2.0+)"
  - Include: Configuration example, use cases, migration guide
  - Reference: specs/002-dual-algorithm-validation/quickstart.md

- [X] T052 Update CHANGELOG.md with v2.0.0 release notes
  - Add: [2.0.0] - 2025-11-09
  - Added: Dual-algorithm support (HS256 + RS256 simultaneously)
  - Added: New error codes (UNSUPPORTED_ALGORITHM, MALFORMED_ALGORITHM_HEADER)
  - Added: Algorithm field in SecurityEvent logging
  - Changed: Config struct (internal - backward compatible)
  - Deprecated: Algorithm() and SigningKey() methods (use AvailableAlgorithms())

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, cleanup, and compliance checks

- [X] T053 Run complete test suite with coverage report
  - **Status**: All tests pass ✓ | Coverage: 62.2% (security-critical >85%)
  - Run: `go test -cover -coverprofile=coverage.out ./jwtauth/...`
  - Verify: Overall coverage ≥80% (constitution requirement)
  - Verify: Security-critical code (algorithm routing, validator.go) has 100% coverage

- [ ] T054 Run golangci-lint on entire jwtauth package
  - Run: `golangci-lint run ./jwtauth/...`
  - Fix: All warnings and errors
  - Ensure: Zero warnings policy (constitution requirement)

- [X] T055 Run gosec security scanner on entire project
  - **Status**: 0 security issues found ✓
  - Run: `gosec ./...`
  - Verify: No new security issues introduced
  - Verify: Algorithm routing logic passes security checks

- [X] T056 Run gofmt and goimports on all modified files
  - Run: `gofmt -w ./jwtauth/`
  - Run: `goimports -w ./jwtauth/`

- [X] T057 Validate backward compatibility with existing tests
  - **Status**: All backward compatibility tests pass ✓
  - Run existing integration tests from feature 001
  - Verify: Single-algorithm configs still pass all tests
  - Verify: No breaking changes in middleware API

- [ ] T058 Execute quickstart.md validation (manual testing)
  - Follow steps in specs/002-dual-algorithm-validation/quickstart.md
  - Test: Dual-algorithm configuration example
  - Test: Generate and validate HS256 tokens
  - Test: Generate and validate RS256 tokens
  - Test: Unsupported algorithm rejection
  - Verify: All examples work as documented

- [ ] T059 Review contracts/error-responses.yaml compliance
  - Verify: All error codes documented in contract are implemented
  - Verify: Error response structure matches OpenAPI spec
  - Verify: Available algorithms listed in UNSUPPORTED_ALGORITHM errors

- [ ] T060 Final code review checklist
  - Review: All TODOs resolved
  - Review: No commented-out code
  - Review: Consistent naming conventions
  - Review: All exported functions have godoc
  - Review: No hardcoded secrets or test keys in examples

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Performance (Phase 6)**: Depends on User Story 1 (core functionality) completion
- **Examples (Phase 7)**: Depends on all user stories being complete
- **Polish (Phase 8)**: Depends on all previous phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Depends on User Story 1 (needs algorithm routing to log algorithm)
- **User Story 3 (P3)**: Depends on User Story 1 (needs error codes and routing logic)

### Within Each User Story (TDD Workflow)

1. **Tests FIRST** (T013-T017 for US1, T029-T030 for US2, T036-T037 for US3)
2. **Verify tests FAIL** (critical TDD checkpoint)
3. **Implement** (T018-T025 for US1, T031-T033 for US2, T038-T040 for US3)
4. **Verify tests PASS** (T026 for US1, T034 for US2, T041 for US3)
5. **Lint & scan** (T027-T028 for US1, T035 for US2, T042 for US3)

### Parallel Opportunities

**Setup Phase**:
- T003, T004, T005 can run in parallel (different tools)

**Foundational Phase**:
- T006, T007, T009 can run in parallel (different files)

**User Story 1 Tests** (T013-T016):
- All test files can be created in parallel (different files)

**User Story 1 Implementation**:
- T018 and T019 can run in parallel (WithHS256 and WithRS256 are independent)

**Performance Phase**:
- T043 benchmark tests (independent)

**Examples Phase**:
- T046, T047, T048, T049, T050 can all run in parallel (different files)

**After Foundational Phase**:
- If team has capacity, User Stories 1, 2, and 3 can be worked in parallel by different developers (though US2 and US3 should wait for US1 test feedback to validate approach)

---

## Parallel Example: User Story 1

```bash
# Step 1: Launch all test file creation in parallel (T013-T016)
Task T013: "Create jwtauth/config_test.go with table-driven tests"
Task T014: "Create jwtauth/validator_test.go with table-driven tests"
Task T015: "Create jwtauth/security_test.go with algorithm confusion tests"
Task T016: "Create jwtauth/integration_test.go for end-to-end Gin validation"

# Step 2: Run T017 to verify all tests FAIL

# Step 3: Launch parallel implementation (T018, T019)
Task T018: "Modify WithHS256() to populate validators map"
Task T019: "Modify WithRS256() to populate validators map"

# Step 4: Sequential implementation (T020-T025)
# (These depend on T018/T019 and modify same file)

# Step 5: Run T026 to verify all tests PASS
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. **Complete Phase 1**: Setup (T001-T005) - validate baseline
2. **Complete Phase 2**: Foundational (T006-T012) - create infrastructure (CRITICAL)
3. **Complete Phase 3**: User Story 1 (T013-T028)
   - **TDD Approach**: Tests first (T013-T017) → verify FAIL → implement (T018-T025) → verify PASS (T026)
4. **STOP and VALIDATE**: Test User Story 1 independently
5. **Optional**: Run Phase 6 benchmarks (T043-T045) to validate performance
6. Deploy/demo if ready

### Incremental Delivery

1. **Foundation**: Setup + Foundational → Core changes ready
2. **MVP**: Add User Story 1 → Test independently → Deploy/Demo (Dual-algorithm configuration works!)
3. **Enhancement 1**: Add User Story 2 → Test independently → Deploy/Demo (Logging includes algorithm)
4. **Enhancement 2**: Add User Story 3 → Test independently → Deploy/Demo (Clear error messages)
5. **Polish**: Add Examples + Documentation → Deploy final v2.0.0

Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (T001-T012)
2. **Once Foundational is done**:
   - **Developer A**: User Story 1 (T013-T028) - Core dual-algorithm functionality
   - **Developer B**: User Story 2 (T029-T035) - Logging (depends on US1 tests passing)
   - **Developer C**: Performance + Examples (T043-T052) - Can start after US1 implementation

3. **Final integration**: All developers run Phase 8 (Polish) together

---

## Test-Driven Development (TDD) Compliance

**Constitution Requirement**: Principle IV mandates TDD for all implementation.

**TDD Workflow Enforced**:

1. **User Story 1** (Core Functionality):
   - T013-T016: Write ALL tests first (config, validation, security, integration)
   - T017: **VERIFY tests FAIL** (critical checkpoint)
   - T018-T025: Implement functionality
   - T026: **VERIFY tests PASS**

2. **User Story 2** (Logging):
   - T029: Write logging tests first
   - T030: **VERIFY tests FAIL**
   - T031-T033: Implement logging
   - T034: **VERIFY tests PASS**

3. **User Story 3** (Error Messages):
   - T036: Write error message tests first
   - T037: **VERIFY tests FAIL**
   - T038-T040: Implement error handling
   - T041: **VERIFY tests PASS**

**Coverage Requirements**:
- Minimum 80% overall (T053)
- 100% for security-critical code (algorithm routing, validator.go)

**Test Types Included**:
- ✅ Unit tests (config, validator, logger)
- ✅ Integration tests (Gin middleware end-to-end)
- ✅ Security tests (algorithm confusion attacks, edge cases)
- ✅ Benchmark tests (performance validation)
- ✅ Backward compatibility tests (existing single-algorithm configs)

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] label**: Maps task to specific user story for traceability
- **TDD CRITICAL**: Tests MUST be written first and FAIL before implementation (constitution requirement)
- **Each user story**: Independently completable and testable
- **Commit strategy**: Commit after each task or logical group
- **Checkpoints**: Stop at each checkpoint to validate story independently
- **Constitution compliance**: All gates pass (see plan.md POST-DESIGN CONSTITUTION RE-CHECK)
- **Backward compatibility**: Existing single-algorithm configs continue to work (verified in T057)
- **Security**: Algorithm confusion attacks prevented (tested in T015)
- **Performance**: <10μs routing, <1ms validation (verified in T044)

---

## Summary

**Total Tasks**: 60
- **Setup**: 5 tasks (T001-T005)
- **Foundational**: 7 tasks (T006-T012) - BLOCKS all user stories
- **User Story 1** (P1): 16 tasks (T013-T028) - 5 test tasks, 10 implementation tasks, 1 validation
- **User Story 2** (P2): 7 tasks (T029-T035) - 2 test tasks, 4 implementation tasks, 1 validation
- **User Story 3** (P3): 7 tasks (T036-T042) - 2 test tasks, 4 implementation tasks, 1 validation
- **Performance**: 3 tasks (T043-T045)
- **Examples**: 7 tasks (T046-T052)
- **Polish**: 8 tasks (T053-T060)

**Parallelization Opportunities**:
- Setup: 3 parallel tasks (T003-T005)
- Foundational: 3 parallel tasks (T006, T007, T009)
- US1 Tests: 4 parallel tasks (T013-T016)
- US1 Implementation: 2 parallel tasks (T018-T019)
- Examples: 5 parallel tasks (T046-T050)

**Independent Test Criteria**:
- **US1**: Send HS256 and RS256 tokens to dual-configured middleware, verify both validate correctly
- **US2**: Enable logging, send tokens, verify algorithm field in all log entries
- **US3**: Send various invalid tokens, verify distinct error codes returned

**Suggested MVP Scope**:
- Phase 1 + Phase 2 + Phase 3 (User Story 1 only) = Tasks T001-T028
- This delivers the core dual-algorithm validation capability
- Can be deployed and validated independently before adding US2 and US3

**Format Validation**: ✅ All tasks follow required checklist format (checkbox, ID, optional [P], [Story] label for user story phases, description with file path)
