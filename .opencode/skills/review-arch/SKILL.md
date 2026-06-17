---
name: review-arch
description: Review code for hexagonal architecture and DDD compliance, combining goarchtest automated tests with AI-powered analysis. Use when user wants to review, audit, or validate hexagonal architecture, DDD patterns, or clean architecture in a Go project.
---

Review the codebase for compliance with hexagonal architecture (ports and adapters), DDD principles, and clean code practices.

This review combines **automated architecture testing** (using goarchtest) with **AI-powered code analysis** for comprehensive validation.

## Phase 1: Automated Architecture Tests

**FIRST: Execute goarchtest tests**

1. Run the architecture tests:
   ```bash
   go test ./test/architecture/... -v
   ```

2. If tests fail, capture the violations and include them in the review report

3. If architecture tests don't exist yet:
   - Warn the user that automated tests are missing
   - Suggest creating them (or offer to create them)
   - Proceed with AI-only review

## Phase 2: AI-Powered Code Analysis

**What to Review:**

## 1. Dependency Rules (CRITICAL)

Check that dependencies flow in the correct direction:
- ✅ Infrastructure → Application → Domain
- ❌ Domain should NOT import application or infrastructure
- ❌ Application should NOT import infrastructure
- Verify import statements in each layer

## 2. Domain Layer Purity

**Check `internal/[domain]/domain/`:**
- ✅ Only standard library and domain-specific imports
- ✅ Entities have business logic methods
- ✅ Repository interfaces (ports) are defined here
- ✅ Domain errors are defined here
- ❌ NO database, HTTP, or external library dependencies
- ❌ NO DTOs (those belong in application layer)
- ❌ NO infrastructure concerns

**Domain Organization:**
- ✅ Each bounded context has its own folder under `internal/`
- ✅ Domain logic is isolated within `internal/[domain]/domain/`
- ✅ Shared concepts are in `internal/shared/`

**Domain Entity Review:**
- Has identity (ID field)?
- Has business logic methods (not just getters/setters)?
- Protects invariants?
- Uses value objects for complex concepts?
- Constructor validates business rules?

## 3. Application Layer (Use Cases)

**Check `internal/[domain]/application/`:**
- ✅ Uses domain repository interfaces (NOT implementations)
- ✅ DTOs are defined here
- ✅ Orchestrates domain objects
- ❌ NO business logic (should be in domain)
- ❌ NO infrastructure dependencies
- ❌ NO database queries or HTTP handling

**Use Case Review:**
- Single responsibility?
- Uses dependency injection?
- Stateless?
- Returns application DTOs (not domain entities)?
- Handles transactions?

## 4. Infrastructure Layer (Adapters)

**Check `internal/[domain]/infrastructure/`:**
- ✅ Implements domain repository interfaces
- ✅ Contains database, HTTP, external service code
- ✅ Maps between domain and infrastructure models
- ❌ Does NOT contain business logic

**Shared Infrastructure:**
- ✅ Common infrastructure (middleware, config, database connection) in `internal/shared/`
- ✅ Domain-specific adapters in domain's infrastructure folder

**Repository Implementation Review:**
- Implements domain repository interface?
- Handles database-specific concerns?
- Maps correctly between domain entities and DB models?
- Returns domain errors?

**HTTP Handler Review:**
- Thin layer (no business logic)?
- Validates HTTP input?
- Calls use cases?
- Maps to proper HTTP status codes?
- Returns JSON responses?

## 5. Testing Strategy

**Check test coverage:**
- Unit tests for domain entities?
- Unit tests for use cases (with mocks)?
- Integration tests for repositories?
- BDD tests (Godog feature files)?
- Step definitions implemented?

**Test Quality:**
- Tests are independent?
- Use table-driven tests?
- Test edge cases and errors?
- Follow AAA pattern (Arrange-Act-Assert)?

## 6. DDD Tactical Patterns

Look for proper use of:
- Entities (with identity)
- Value Objects (immutable, self-validating)
- Aggregates (consistency boundaries)
- Domain Services (when behavior doesn't belong to entity)
- Repository pattern (persistence abstraction)
- Domain Events (if applicable)

## 7. Common Anti-Patterns to Flag

❌ Anemic domain model (entities with only getters/setters)
❌ Leaking domain entities to HTTP responses
❌ Business logic in handlers or controllers
❌ Domain depending on infrastructure
❌ God objects (classes doing too much)
❌ Missing validations
❌ Improper error handling
❌ Missing tests

## 8. Code Quality

- Proper error handling?
- Context propagation?
- Appropriate logging?
- No TODO comments in production code?
- Follows Go conventions and idioms?
- Proper package naming?

## 9. Architecture Tests (goarchtest)

**Check if architecture tests exist** in `test/architecture/`:
- ✅ Tests exist and are comprehensive
- ❌ Tests missing - offer to create them
- ⚠️ Tests exist but incomplete - suggest improvements

**Common architectural constraints to test:**

```go
// Domain layer purity
goarchtest.InPath(projectPath).
    That().
    ResideInNamespace("internal/*/domain").
    ShouldNot().
    HaveDependencyOn("internal/*/infrastructure").
    And().
    ShouldNot().
    HaveDependencyOn("internal/*/application")

// Application layer independence
goarchtest.InPath(projectPath).
    That().
    ResideInNamespace("internal/*/application").
    ShouldNot().
    HaveDependencyOn("internal/*/infrastructure")

// Domain isolation (no cross-domain dependencies)
goarchtest.InPath(projectPath).
    That().
    ResideInNamespace("internal/user/").
    ShouldNot().
    HaveDependencyOn("internal/order/")

// Naming conventions
goarchtest.InPath(projectPath).
    That().
    ResideInNamespace("internal/*/domain").
    And().
    HaveNameEndingWith("Repository").
    Should().
    BeInterfaces()
```

**Output Format:**

Provide a comprehensive review combining automated test results with AI analysis:

### 1. Architecture Test Results (goarchtest)
- **Status**: Pass/Fail
- **Violations Found**: List specific violations from goarchtest
- **Failed Rules**: Which architectural constraints were broken
- **Affected Files**: Files that violated the rules

### 2. AI Analysis Summary
- **Overall Health**: Good/Needs Improvement/Critical Issues
- **Architecture Score**: Based on both automated tests and AI review

### 3. Dependency Violations
- From goarchtest (automated)
- From AI analysis (patterns not caught by tests)

### 4. Layer-by-Layer Issues
- **Domain Layer**: Purity violations, missing business logic
- **Application Layer**: Improper dependencies, missing use cases
- **Infrastructure Layer**: Leaking concerns, improper implementations

### 5. DDD Compliance
- Tactical patterns usage
- Bounded context boundaries
- Ubiquitous language

### 6. Test Coverage
- Unit tests
- Integration tests
- Architecture tests
- BDD tests

### 7. Recommendations (Prioritized)
- **Critical**: Must fix (architectural violations)
- **High**: Should fix soon (design issues)
- **Medium**: Improvements (code quality)
- **Low**: Nice to have (optimizations)

### 8. Quick Wins
Easy fixes that provide immediate value

### 9. Next Steps
1. Fix goarchtest violations first
2. Address critical AI-detected issues
3. Improve test coverage
4. Refactor for better DDD compliance

**Execution Instructions:**

1. ALWAYS run `go test ./test/architecture/... -v` first
2. Parse the output and include results in the review
3. Then perform AI analysis on the code
4. Combine both results into a single comprehensive report
5. If architecture tests are missing, create them using the architecture test template
6. Generate dependency graph if violations are found (optional but helpful)

Be specific with file paths and line numbers. Focus on architectural issues, not style.
