# AGENTS.md

Entry point for AI agents working on this project. Keep this file current as the project evolves.

---

## Project

**databricks-tui** — A terminal UI to browse a Databricks workspace. Uses service principal auth via `.databrickscfg` profiles.

## Tech Stack

| Layer | Choice | Version |
|-------|--------|---------|
| Language | Go | ≥1.22 |
| TUI Framework | Bubble Tea (`charm.land/bubbletea`) | v2 |
| Components | Bubbles (`github.com/charmbracelet/bubbles`) | v2 |
| Styling | Lip Gloss (`charm.land/lipgloss`) | v2 |
| Databricks SDK | `github.com/databricks/databricks-sdk-go` | latest |
| Module | `github.com/youenchene/databricks-tui` | — |

## Architecture: Hexagonal (Ports & Adapters)

```
cmd/
  tui/           → main entry point, wires dependencies
internal/
  domain/        → pure models & use-case interfaces (zero external deps)
    cluster/
    job/
    notebook/
    ...
  adapters/
    databricks/  → SDK client, implements domain ports
  ports/
    tui/         → Bubble Tea views & navigation (implements display ports)
```

**Rules:**
- `domain/` must not import `adapters/` or `ports/` or any framework/SDK
- Each use case (browse clusters, inspect jobs, etc.) gets its own package in `domain/`
- Dependencies flow inward: `cmd` → `adapters/ports` → `domain`

## Auth

Launch with an optional `--profile` flag. Falls back to `DEFAULT` profile in `~/.databrickscfg`.

```bash
databricks-tui --profile MYPROFILE
databricks-tui                          # uses DEFAULT
```

The Databricks SDK handles credential resolution automatically (PAT, OAuth, Azure SP, Azure CLI, etc.).

## Git Workflow

- **Feature branches** — All work happens on branches (`feature/description`). Never push directly to `main`.
- **PRs mandatory** — Every change lands via pull request. No direct commits to `main`.
- **Branch naming** — `feature/<slug>`, `fix/<slug>`, `chore/<slug>`
- **Commit style** — Conventional commits with emoji: `✨ feat: ...`, `🐛 fix: ...`, `♻️ refactor: ...`

**Workflow:**
```bash
git checkout -b feature/my-feature
# ... make changes ...
make test
git add .
git commit -m "✨ feat: description"
git push -u origin feature/my-feature
# Create PR on GitHub → merge
```

## Conventions

- **Go idioms** — Explicit error handling, small interfaces, accept interfaces / return structs
- **Pure functions** — Domain logic must be testable without mocks
- **Dependency injection** — Wire dependencies manually in `cmd/tui/main.go` (no DI framework needed at this scale)
- **Testing** — Table-driven tests with `stretchr/testify`; `domain/` packages must have >90% coverage

## TDD Approach

All features follow Test-Driven Development. The cycle:

1. **Red** — Write a failing test that defines the expected behavior
2. **Green** — Write the minimum code to make the test pass
3. **Refactor** — Clean up both the code and the tests; improve names, extract helpers

**What to test:**
- Domain entities: pure logic, state transitions, summary methods
- Domain services: with stub/mock repositories (never real adapters)
- Adapter mappers: SDK type → domain type conversion functions
- TUI models: message handling, navigation state changes

**Test structure:**
```
test/
  unit/
    domain/
      cluster/
        cluster_test.go    # entity + service tests
      job/
        job_test.go        # entity + service tests
      notebook/
        notebook_test.go   # entity + service tests
  architecture/
    arch_test.go         # goarchtest hex validation
internal/
  adapters/
    databricks/
      mapper_test.go     # mapping function tests (internal package)
```

**Stub pattern for domain services:**
```go
type stubRepo struct {
    items []domain.Thing
    err   error
}
func (s *stubRepo) List(_ context.Context) ([]domain.Thing, error) { ... }
func (s *stubRepo) Get(_ context.Context, id string) (domain.Thing, error) { ... }
```

**Rules:**
- Never test through real SDK/network calls in unit tests
- Each test file covers one domain package
- Table-driven tests for pure functions; named subtests for service methods
- Target: >90% coverage on `internal/domain/` packages

## Skills to Install

Run these in the repo to install opencode skills locally:

```bash
# Project setup & architecture
opencode skill install start-project       # Scaffold hexagonal Go project
opencode skill install review-arch         # Validate architecture compliance

# Go language skills
opencode skill install golang-project-layout
opencode skill install golang-structs-interfaces
opencode skill install golang-design-patterns
opencode skill install golang-error-handling
opencode skill install golang-naming
opencode skill install golang-context
opencode skill install golang-concurrency

# Testing
opencode skill install golang-testing
opencode skill install golang-stretchr-testify

# Libraries
opencode skill install golang-spf13-cobra  # CLI flag parsing (--profile)
```

## Getting Started

```bash
# After scaffolding:
go mod init github.com/youenchene/databricks-tui
go get charm.land/bubbletea/v2
go get github.com/charmbracelet/bubbles/v2
go get charm.land/lipgloss/v2
go get github.com/databricks/databricks-sdk-go
```

## Agent Instructions

When working on this repo:
1. Read `AGENTS.md` first (this file)
2. Follow hexagonal architecture: domain models first, then adapters, then UI
3. Each new workspace feature (clusters, jobs, notebooks, etc.) should be its own use case
4. Write tests alongside code — domain logic must be testable in isolation
5. Use Bubble Tea v2 APIs (`tea.KeyPressMsg`, not `tea.KeyMsg`; `charm.land/bubbletea/v2` import path)
