# Tasks: JWT Authentication Middleware

**Input**: Design documents from `/specs/001-jwt-auth-middleware/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The constitution mandates TDD (Test-Driven Development). Tests are REQUIRED and must be written BEFORE implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go library**: Package files in `jwtauth/`, tests co-located with `_test.go` suffix
- Examples in `examples/gin/` and `examples/grpc/`
- Paths shown below follow Go idiomatic flat package structure

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Initialize Go module with `go mod init github.com/user/vibrant-auth-middleware-go/jwtauth`
- [ ] T002 Create package directory structure (jwtauth/, examples/gin/, examples/grpc/)
- [ ] T003 [P] Install golang-jwt/jwt v5 dependency with `go get github.com/golang-jwt/jwt/v5`
- [ ] T004 [P] Install Gin dependency for examples with `go get github.com/gin-gonic/gin`
- [ ] T005 [P] Install gRPC dependency for examples with `go get google.golang.org/grpc`
- [ ] T006 [P] Create go.sum and verify dependency checksums
- [ ] T007 [P] Create LICENSE file (choose license - MIT/Apache 2.0 recommended)
- [ ] T008 [P] Create CHANGELOG.md skeleton with v0.1.0-dev section
- [ ] T009 [P] Setup golangci-lint configuration file .golangci.yml with strict rules
- [ ] T010 [P] Create README.md with project description and quickstart placeholder

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests for Foundational Components (TDD - Write FIRST)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T011 [P] Create test helper package in jwtauth/internal/testutil/testutil.go with generic assertion helpers
- [ ] T012 [P] Write config validation tests in jwtauth/config_test.go (algorithm validation, key length, clock skew)
- [ ] T013 [P] Write functional options tests in jwtauth/config_test.go (WithHS256, WithRS256, WithClockSkew, WithCookie, WithLogger)
- [ ] T014 [P] Write error type tests in jwtauth/errors_test.go (ValidationError codes, Error() method, Unwrap())
- [ ] T015 [P] Write context key tests in jwtauth/context_test.go (WithClaims, GetClaims, MustGetClaims, key isolation)

### Implementation for Foundational Components

- [ ] T016 Define error types in jwtauth/errors.go (ValidationError with codes: EXPIRED, INVALID_SIGNATURE, MISSING_TOKEN, MALFORMED, ALGORITHM_MISMATCH, NONE_ALGORITHM, CONFIG_ERROR)
- [ ] T017 Implement Config struct in jwtauth/config.go with unexported fields (algorithm, signingKey, clockSkewLeeway, cookieName, requiredClaims, logger)
- [ ] T018 Implement NewConfig constructor in jwtauth/config.go with validation (algorithm check, key validation, clock skew ≥0)
- [ ] T019 [P] Implement WithHS256 functional option in jwtauth/config.go (validates key length ≥32 bytes)
- [ ] T020 [P] Implement WithRS256 functional option in jwtauth/config.go (validates non-nil *rsa.PublicKey)
- [ ] T021 [P] Implement WithClockSkew functional option in jwtauth/config.go (validates ≥0)
- [ ] T022 [P] Implement WithCookie functional option in jwtauth/config.go (sets cookie name)
- [ ] T023 [P] Implement WithLogger functional option in jwtauth/config.go (accepts *slog.Logger)
- [ ] T024 [P] Implement WithRequiredClaims functional option in jwtauth/config.go (stores required claim names)
- [ ] T025 Define Claims struct in jwtauth/claims.go (Subject, Issuer, Audience, ExpiresAt, NotBefore, IssuedAt, JWTID, Custom map)
- [ ] T026 Define contextKey type in jwtauth/context.go (unexported string with package prefix)
- [ ] T027 [P] Implement WithClaims in jwtauth/context.go (context.WithValue with typed key)
- [ ] T028 [P] Implement GetClaims in jwtauth/context.go (type-safe retrieval, returns (*Claims, bool))
- [ ] T029 [P] Implement MustGetClaims in jwtauth/context.go (panics if not present)
- [ ] T030 Implement ParseRSAPublicKeyFromPEM helper in jwtauth/keys.go (supports PKCS#1 and PKIX PEM formats)
- [ ] T031 Define SecurityEvent struct in jwtauth/logger.go (EventType, Timestamp, RequestID, UserID, FailureReason, TokenPreview, Latency)
- [ ] T032 Implement SecurityEvent.LogValue in jwtauth/logger.go (slog.LogValuer for redaction)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Gin HTTP Authentication (Priority: P1) 🎯 MVP

**Goal**: Working JWT middleware for Gin framework with HS256 validation, multi-source token extraction, and immediate initialization

**Independent Test**: Start Gin server with middleware, send requests with valid/invalid tokens, verify 401 responses and context injection work correctly

### Tests for User Story 1 (TDD - Write FIRST) ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T033 [P] [US1] Write token extraction tests in jwtauth/extractor_test.go (Authorization header "Bearer <token>", cookie extraction, precedence, malformed headers)
- [ ] T034 [P] [US1] Write JWT validator tests in jwtauth/validator_test.go (HS256 signature validation, exp/nbf validation, clock skew tolerance, algorithm mismatch rejection, "none" algorithm rejection)
- [ ] T035 [P] [US1] Write Gin middleware integration tests in jwtauth/middleware_test.go (valid token → 200 + context, no token → 401, expired token → 401, invalid signature → 401, clock skew within tolerance → 200)
- [ ] T036 [P] [US1] Write security attack tests in jwtauth/security_test.go (algorithm confusion attack, "none" algorithm attack, tampered payload, malformed JWT)

### Implementation for User Story 1

- [ ] T037 [US1] Implement extractTokenFromHeader in jwtauth/extractor.go (parse "Authorization: Bearer <token>", handle malformed headers)
- [ ] T038 [US1] Implement extractTokenFromCookie in jwtauth/extractor.go (retrieve from http.Request.Cookie, use configured cookie name)
- [ ] T039 [US1] Implement extractToken in jwtauth/extractor.go (try header first, fallback to cookie, return MISSING_TOKEN error if neither)
- [ ] T040 [US1] Implement parseAndValidateJWT in jwtauth/validator.go (parse JWT, validate algorithm, verify signature with golang-jwt/jwt, validate exp/nbf with clock skew)
- [ ] T041 [US1] Implement validateAlgorithm in jwtauth/validator.go (reject "none", verify matches config, prevent algorithm confusion)
- [ ] T042 [US1] Implement validateClaims in jwtauth/validator.go (check exp > now-clockSkew, check nbf ≤ now+clockSkew, check required claims present)
- [ ] T043 [US1] Implement mapJWTClaimsToClaims in jwtauth/validator.go (convert golang-jwt MapClaims to our Claims struct)
- [ ] T044 [US1] Implement JWTAuth function in jwtauth/middleware.go (returns gin.HandlerFunc, extracts token, validates, injects context, calls c.Next() or c.AbortWithStatusJSON(401))
- [ ] T045 [US1] Implement logSecurityEvent in jwtauth/logger.go (emit SecurityEvent via slog with redaction)
- [ ] T046 [US1] Add error response formatting in jwtauth/middleware.go (JSON: {"error": "unauthorized", "reason": code})

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently. Gin middleware works with HS256, multi-source extraction, and security logging.

---

## Phase 4: User Story 2 - gRPC Service Authentication (Priority: P2)

**Goal**: gRPC unary interceptor with JWT validation from metadata, compatible with middleware chaining

**Independent Test**: Create gRPC server with interceptor, send RPC requests with metadata containing tokens, verify Unauthenticated status codes and context injection

### Tests for User Story 2 (TDD - Write FIRST) ⚠️

- [ ] T047 [P] [US2] Write gRPC metadata extraction tests in jwtauth/grpc_test.go (extract from "authorization" key, handle missing metadata, malformed values)
- [ ] T048 [P] [US2] Write gRPC interceptor integration tests in jwtauth/grpc_test.go (valid token → RPC succeeds + context, no token → Unauthenticated status, invalid token → Unauthenticated + log, interceptor chaining with multiple middleware)
- [ ] T049 [P] [US2] Write gRPC context propagation tests in jwtauth/grpc_test.go (claims accessible in handler, context passed through correctly)

### Implementation for User Story 2

- [ ] T050 [US2] Implement extractTokenFromMetadata in jwtauth/grpc.go (retrieve from gRPC incoming metadata key "authorization", parse "Bearer <token>")
- [ ] T051 [US2] Implement UnaryServerInterceptor function in jwtauth/grpc.go (returns grpc.UnaryServerInterceptor, extracts from metadata, validates token, injects context, calls handler or returns status.Error(codes.Unauthenticated))
- [ ] T052 [US2] Add gRPC-specific error formatting in jwtauth/grpc.go (convert ValidationError codes to gRPC status messages)
- [ ] T053 [US2] Reuse validation core from US1 in jwtauth/grpc.go (call parseAndValidateJWT, share Claims injection logic)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently. Both Gin and gRPC integration patterns complete.

---

## Phase 5: User Story 3 - RS256 Public Key Authentication (Priority: P3)

**Goal**: RS256 (RSA-SHA256) signature validation with public key, enabling zero-trust architecture

**Independent Test**: Configure middleware with RSA public key, generate tokens with private key, verify valid tokens accepted and tampered tokens rejected

### Tests for User Story 3 (TDD - Write FIRST) ⚠️

- [ ] T054 [P] [US3] Write RS256 key parsing tests in jwtauth/keys_test.go (parse PKCS#1 PEM, parse PKIX PEM, reject invalid PEM, reject non-RSA keys)
- [ ] T055 [P] [US3] Write RS256 validation tests in jwtauth/validator_test.go (RS256 signature verification, public key immutability, tampered payload detection, key mismatch rejection)
- [ ] T056 [P] [US3] Write RS256 integration tests in jwtauth/middleware_test.go and jwtauth/grpc_test.go (RS256 tokens with Gin, RS256 tokens with gRPC, algorithm validation RS256 vs HS256)

### Implementation for User Story 3

- [ ] T057 [US3] Extend parseAndValidateJWT in jwtauth/validator.go to support RS256 (detect algorithm, use appropriate verification method from golang-jwt/jwt)
- [ ] T058 [US3] Extend validateAlgorithm in jwtauth/validator.go to accept RS256 (validate RS256 matches config, prevent HS256→RS256 confusion)
- [ ] T059 [US3] Test RS256 with both Gin and gRPC middleware (verify shared validation core works for both)

**Checkpoint**: All user stories (US1, US2, US3) should now be independently functional. Both HS256 and RS256 algorithms supported across Gin and gRPC.

---

## Phase 6: User Story 4 - Comprehensive Security Telemetry (Priority: P4)

**Goal**: Structured JSON logging with secret redaction, correlation IDs, and performance metrics under target (<100μs overhead)

**Independent Test**: Trigger auth success/failure scenarios, verify structured JSON logs emitted with redacted tokens, request IDs, and failure reasons

### Tests for User Story 4 (TDD - Write FIRST) ⚠️

- [ ] T060 [P] [US4] Write logging tests in jwtauth/logger_test.go (success event structure, failure event structure, token redaction verification, correlation ID presence, performance overhead measurement)
- [ ] T061 [P] [US4] Write secret redaction tests in jwtauth/logger_test.go (tokens reduced to first 8 chars + "...", signing keys never logged, custom claims not logged by default)
- [ ] T062 [P] [US4] Write performance tests in jwtauth/benchmark_test.go (BenchmarkJWTValidation <1ms p99, BenchmarkMiddlewareOverhead <100μs p99, BenchmarkLogging <10μs overhead)

### Implementation for User Story 4

- [ ] T063 [US4] Enhance logSecurityEvent in jwtauth/logger.go (add timestamp, request ID extraction/generation, latency measurement)
- [ ] T064 [US4] Implement token redaction in jwtauth/logger.go (first 8 chars + "..." for TokenPreview field)
- [ ] T065 [US4] Add request ID support in jwtauth/context.go (WithRequestID, GetRequestID, generate UUID if missing)
- [ ] T066 [US4] Integrate logging into Gin middleware in jwtauth/middleware.go (log on success and failure paths, measure latency)
- [ ] T067 [US4] Integrate logging into gRPC interceptor in jwtauth/grpc.go (log on success and failure paths, measure latency)
- [ ] T068 [US4] Add performance benchmarks in jwtauth/benchmark_test.go (measure allocations <3 in hot path, validate latency targets)

**Checkpoint**: All user stories complete and independently functional. Security telemetry provides full visibility into auth events.

---

## Phase 7: Examples & Documentation

**Purpose**: Demonstrate integration patterns and provide developer-friendly documentation

- [ ] T069 [P] Create Gin example in examples/gin/main.go (simple server with protected endpoint, demonstrates WithHS256, WithCookie, claims access)
- [ ] T070 [P] Create gRPC example in examples/grpc/main.go (simple gRPC service with interceptor, demonstrates WithRS256, key loading from PEM)
- [ ] T071 [P] Create Example_ginIntegration test in jwtauth/example_gin_test.go (godoc example)
- [ ] T072 [P] Create Example_grpcIntegration test in jwtauth/example_grpc_test.go (godoc example)
- [ ] T073 [P] Create Example_rs256KeyParsing test in jwtauth/example_keys_test.go (godoc example)
- [ ] T074 [P] Write package-level godoc in jwtauth/doc.go (overview, security considerations, quick example)
- [ ] T075 [P] Add function godoc comments to all exported functions (JWTAuth, UnaryServerInterceptor, NewConfig, all options, GetClaims, etc.)
- [ ] T076 [P] Write comprehensive README.md (installation, quick start, configuration, security checklist, contributing, license)
- [ ] T077 [P] Add security considerations section to README.md (key management, HTTPS requirement, clock sync, rate limiting recommendations)
- [ ] T078 [P] Update CHANGELOG.md with v0.1.0 release notes (initial features, breaking changes: none, security: first release)

---

## Phase 8: Test Coverage & Quality

**Purpose**: Ensure 80% coverage general, 100% security-critical paths, all quality gates pass

- [ ] T079 Run all tests with coverage and verify ≥80% overall: `go test -coverprofile=coverage.out ./...`
- [ ] T080 Verify 100% coverage on security-critical paths (validator.go, algorithm validation, signature verification)
- [ ] T081 Run golangci-lint and fix all warnings: `golangci-lint run ./...`
- [ ] T082 Run go vet and fix all issues: `go vet ./...`
- [ ] T083 Run gosec security scanner and address findings: `gosec ./...`
- [ ] T084 Run govulncheck for dependency vulnerabilities: `govulncheck ./...`
- [ ] T085 Run gofmt and goimports to format all code: `gofmt -w . && goimports -w .`
- [ ] T086 Verify all benchmarks meet performance targets (run `go test -bench=. -benchmem`)
- [ ] T087 Verify all Example_* tests pass and generate valid godoc output

---

## Phase 9: Final Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T088 [P] Review error messages for security leakage (ensure no signing keys, full tokens, or internal details exposed)
- [ ] T089 [P] Review all exported types for API stability (ensure no accidental exports, stable field names)
- [ ] T090 [P] Add table-driven test cases for edge cases (empty strings, nil pointers, boundary conditions)
- [ ] T091 [P] Optimize hot path allocations (profile and reduce to <3 allocations per request)
- [ ] T092 [P] Add CI/CD configuration file (.github/workflows/ci.yml) with test, lint, security scan steps
- [ ] T093 [P] Create CONTRIBUTING.md with development guidelines (TDD process, testing requirements, PR checklist)
- [ ] T094 [P] Create SECURITY.md with vulnerability reporting instructions
- [ ] T095 [P] Verify all TODOs and FIXMEs resolved in codebase
- [ ] T096 Run integration test suite against real Gin and gRPC servers (not just unit tests)
- [ ] T097 Run security test suite with attack scenarios (algorithm confusion, replay, tampering)
- [ ] T098 Perform final code review checklist (Go idioms, security, performance, documentation)
- [ ] T099 Tag v0.1.0-alpha release and push to Git
- [ ] T100 Run full validation from quickstart.md (verify <15 minute integration time for new users)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - **User Story 1 (P1 - Gin)**: Can start after Foundational - No dependencies on other stories
  - **User Story 2 (P2 - gRPC)**: Can start after Foundational - Independent of US1 (shares validation core but can implement in parallel)
  - **User Story 3 (P3 - RS256)**: Extends Foundational + US1/US2 - Should implement after US1 OR US2 complete (extends validator.go)
  - **User Story 4 (P4 - Telemetry)**: Extends US1 + US2 - Should implement after US1 AND US2 complete (adds logging to both)
- **Examples (Phase 7)**: Depends on US1 and US2 completion
- **Test Coverage (Phase 8)**: Depends on all user stories complete
- **Polish (Phase 9)**: Depends on Test Coverage passing

### User Story Dependencies

```
Foundational (Phase 2) [BLOCKS ALL]
    ├─→ US1 (Gin) [Independent]
    ├─→ US2 (gRPC) [Independent]
    ├─→ US3 (RS256) [Extends validator from US1/US2]
    └─→ US4 (Telemetry) [Integrates with US1 + US2]

Execution Order:
1. Setup → Foundational (sequential, blocking)
2. US1 + US2 (parallel, independent)
3. US3 (after any one of US1/US2, extends validator)
4. US4 (after US1 + US2, adds logging to both)
5. Examples (after US1 + US2)
6. Test Coverage → Polish (sequential)
```

### Within Each User Story

**TDD Cycle** (STRICT ORDER):
1. **Write tests FIRST** (all test tasks for the story)
2. **Run tests** (verify they FAIL - red phase)
3. **Implement code** (implementation tasks for the story)
4. **Run tests again** (verify they PASS - green phase)
5. **Refactor** (optimize while keeping tests green)

**Within Implementation**:
- Core structures (errors, claims, context) before validation
- Validation before middleware integration
- Middleware integration before logging/telemetry
- Security tests before moving to next story

### Parallel Opportunities

#### Setup Phase (can all run in parallel after T001-T002):
```bash
# T003-T010 all parallel (different files, independent)
go get packages + create docs
```

#### Foundational Phase - Tests (can run in parallel):
```bash
# T011-T015 all parallel (different test files)
Write: config_test.go, errors_test.go, context_test.go, testutil.go
```

#### Foundational Phase - Implementation:
```bash
# T019-T024 parallel (functional options in config.go, independent)
# T027-T029 parallel (context helpers, independent)
```

#### User Story 1 - Tests (can run in parallel):
```bash
# T033-T036 all parallel (different test files for US1)
Write: extractor_test.go, validator_test.go, middleware_test.go, security_test.go
```

#### User Story 2 - Tests (can run in parallel):
```bash
# T047-T049 all parallel (gRPC test files)
Write: grpc_test.go tests
```

#### User Story 3 - Tests (can run in parallel):
```bash
# T054-T056 all parallel (RS256 test files)
Write: keys_test.go, validator RS256 tests, integration RS256 tests
```

#### User Story 4 - Tests (can run in parallel):
```bash
# T060-T062 all parallel (telemetry test files)
Write: logger_test.go, benchmark_test.go
```

#### Examples & Documentation (can all run in parallel after US1+US2):
```bash
# T069-T078 all parallel (different files, independent)
Write: examples, godoc, README, CHANGELOG
```

---

## Parallel Example: Foundational Phase

```bash
# After T010 completes, launch T011-T015 together (TDD tests first):
Task T011: "Create test helper package in jwtauth/internal/testutil/testutil.go"
Task T012: "Write config validation tests in jwtauth/config_test.go"
Task T013: "Write functional options tests in jwtauth/config_test.go"
Task T014: "Write error type tests in jwtauth/errors_test.go"
Task T015: "Write context key tests in jwtauth/context_test.go"

# After T018 completes, launch T019-T024 together (functional options):
Task T019: "Implement WithHS256 in jwtauth/config.go"
Task T020: "Implement WithRS256 in jwtauth/config.go"
Task T021: "Implement WithClockSkew in jwtauth/config.go"
Task T022: "Implement WithCookie in jwtauth/config.go"
Task T023: "Implement WithLogger in jwtauth/config.go"
Task T024: "Implement WithRequiredClaims in jwtauth/config.go"

# After T026 completes, launch T027-T029 together (context helpers):
Task T027: "Implement WithClaims in jwtauth/context.go"
Task T028: "Implement GetClaims in jwtauth/context.go"
Task T029: "Implement MustGetClaims in jwtauth/context.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

**Minimum Viable Product** - Deliver working value ASAP:

1. Complete Phase 1: Setup (T001-T010)
2. Complete Phase 2: Foundational (T011-T032) - CRITICAL blocking phase
3. Complete Phase 3: User Story 1 (T033-T046) - Gin middleware with HS256
4. **STOP and VALIDATE**: Test User Story 1 independently
   - Start Gin server with middleware
   - Send valid token → verify 200 + claims in context
   - Send invalid token → verify 401 + error message
   - Send expired token → verify 401 + appropriate error
   - Verify security logging works
5. Deploy/demo if ready - **MVP delivered!**

**Estimated Tasks**: 46 tasks (Setup: 10, Foundational: 22, US1: 14)
**Estimated Time**: 2-3 days for experienced Go developer

---

### Incremental Delivery

**Each user story adds independent value**:

1. **Foundation → US1** (T001-T046): Gin middleware MVP → **Deploy/Demo** ✅
2. **+ US2** (T047-T053): Add gRPC support → **Deploy/Demo** ✅
3. **+ US3** (T054-T059): Add RS256 → **Deploy/Demo** ✅
4. **+ US4** (T060-T068): Add comprehensive telemetry → **Deploy/Demo** ✅
5. **+ Examples** (T069-T078): Documentation complete → **Public release ready** ✅
6. **+ Polish** (T079-T100): Production-ready v0.1.0 → **Official release** ✅

**Each increment is independently testable and deliverable** - ship value continuously!

---

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (T001-T032)
2. **Once Foundational is done**:
   - **Developer A**: User Story 1 (T033-T046) - Gin middleware
   - **Developer B**: User Story 2 (T047-T053) - gRPC interceptor
   - Developer C can start preparing examples or work on US3 after US1/US2 foundation
3. **After US1 + US2 complete**:
   - **Developer A**: User Story 3 (T054-T059) - RS256 extension
   - **Developer B**: User Story 4 (T060-T068) - Telemetry
   - **Developer C**: Examples (T069-T078) - Documentation
4. **Final phase** (T079-T100): Team review, quality gates, polish

**Key**: Foundational phase is the only blocking bottleneck. After that, stories are highly parallelizable.

---

## Notes

- **[P] tasks** = different files, no dependencies - safe to parallel
- **[Story] label** maps task to specific user story for traceability
- **Each user story should be independently completable and testable**
- **TDD MANDATORY**: Write tests first (red), implement (green), refactor
- **Verify tests fail** before implementing (validates test correctness)
- **Commit after each task** or logical group (small, focused commits)
- **Stop at any checkpoint** to validate story independently
- **Constitution compliance**: Every task aligns with Go idioms, security-first, TDD, performance targets

---

## Task Count Summary

- **Phase 1 (Setup)**: 10 tasks
- **Phase 2 (Foundational)**: 22 tasks (11 tests + 11 implementation)
- **Phase 3 (US1 - Gin)**: 14 tasks (4 tests + 10 implementation)
- **Phase 4 (US2 - gRPC)**: 7 tasks (3 tests + 4 implementation)
- **Phase 5 (US3 - RS256)**: 6 tasks (3 tests + 3 implementation)
- **Phase 6 (US4 - Telemetry)**: 9 tasks (3 tests + 6 implementation)
- **Phase 7 (Examples & Docs)**: 10 tasks
- **Phase 8 (Test Coverage)**: 9 tasks
- **Phase 9 (Polish)**: 13 tasks

**Total**: 100 tasks
**Parallel Opportunities**: ~40% of tasks can run in parallel within phases
**MVP Scope**: 46 tasks (Setup + Foundational + US1)
**Full Feature**: 100 tasks

---

## Success Criteria Validation

Each user story independently verifiable:

- **US1**: ✅ Gin server + middleware + HS256 validation + multi-source extraction works
- **US2**: ✅ gRPC server + interceptor + metadata extraction + context propagation works
- **US3**: ✅ RS256 validation + public key loading + algorithm detection works
- **US4**: ✅ Structured logs + secret redaction + correlation IDs + performance targets met

**Final validation**: Run quickstart.md integration in <15 minutes (SC-010) ✅
