# Databricks TUI

A terminal UI for browsing Databricks workspaces. Built with Go, Bubble Tea, and hexagonal architecture.

## Quick Start

```bash
# Build
make build

# Run (uses DEFAULT profile from ~/.databrickscfg)
make run

# Run with a specific profile
make run-profile PROFILE=MYPROFILE
# or
./bin/databricks-tui MYPROFILE
```

## Architecture

Hexagonal (ports & adapters) organized by use case:

```
cmd/tui/          → Entry point, manual DI
internal/
  domain/         → Pure models & repository interfaces (zero external deps)
    cluster/      → Cluster browsing use case
    job/          → Job browsing use case
    notebook/     → Notebook browsing use case
  adapters/
    databricks/   → SDK client implementing domain ports
  ports/
    tui/          → Bubble Tea views & navigation
```

**Rules:**
- `domain/` has zero external dependencies — pure Go, testable in isolation
- Dependencies flow inward: `cmd` → `adapters/ports` → `domain`
- Each use case is independently testable and maintainable

## Testing

```bash
make test-unit   # Domain logic tests
make test-arch   # Hexagonal architecture validation
make test        # All tests
```

## Requirements

- Go 1.22+
- A `.databrickscfg` file in `~/.databrickscfg` with at least a `DEFAULT` profile
- Valid Databricks credentials (PAT, OAuth, Azure SP, etc.)
