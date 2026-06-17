# Databricks TUI

A terminal UI to browse Databricks workspaces — jobs, clusters, and notebooks. Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and hexagonal architecture.

<p align="center">
  <img src="https://img.shields.io/badge/go-%3E%3D1.22-blue" alt="Go version">
  <img src="https://img.shields.io/badge/tests-116%20passing-green" alt="Tests">
</p>

## Quick Start

```bash
# Build
make build

# Run with DEFAULT profile
./bin/databricks-tui

# Run with a named profile
./bin/databricks-tui MYPROFILE
```

Requires a `.databrickscfg` file at `~/.databrickscfg`. The Databricks SDK handles PAT, OAuth, Azure SP, and Azure CLI auth automatically.

## Features

### Jobs (tab `1`)

| Screen | What it shows |
|--------|-------------|
| **Job List** | All jobs with name, schedule, and last run state |
| **Job Detail** | Metadata (ID, schedule, creator, task count) + recent runs + task definitions |
| **Run Detail** | Run metadata, duration, task execution states, per-task logs & output |
| **Task Detail** | Task-level output, logs, error traces |

- **Favorites** — press `f` to star a job, `F` to show only favorites. Saved to `~/.databricks-tui/favorites.json`.
- **Search** — press `/` to filter by name, ID, or schedule. `esc` to exit search and resume commands.

### Clusters (tab `2`)

Browse all clusters with state, Spark version, and node type. Color-coded states (green=running, amber=pending, red=error).

### Notebooks (tab `3`)

Browse workspace notebooks with language and path. Directories and files shown with distinct icons.

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| `1` / `2` / `3` | Switch tabs (Jobs / Clusters / Notebooks) |
| `q` / `ctrl+c` | Quit |

### All List Screens

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Navigate items |
| `/` | Enter search (type to filter, `esc` to exit) |
| `enter` | Select / zoom into item |

### Jobs Screen

| Key | Action |
|-----|--------|
| `f` | Toggle ★ favorite on selected job |
| `F` | Toggle "favorites only" view |
| `tab` | Switch between Runs / Tasks focus in Job Detail |
| `esc` / `backspace` | Go back (detail → list, run → job, task → run) |

## Logging

Logs are written to `$TMPDIR/databricks-tui/tui.log` with structured slog format. API errors, timeouts, and auth failures are logged here without cluttering the UI.

## Development

### Build & Test

```bash
make build        # Build binary to bin/
make test         # Run all tests (unit + architecture + adapter mappers)
make test-unit    # Domain logic tests only
make test-arch    # Hexagonal architecture validation
make lint         # Run golangci-lint
```

### Architecture

Hexagonal (ports & adapters) organized by use case:

```
cmd/tui/            → Entry point, manual DI
internal/
  domain/           → Pure models & repository interfaces (zero external deps)
    cluster/        → Cluster browsing use case
    job/            → Job browsing use case
    notebook/       → Notebook browsing use case
  adapters/
    databricks/     → SDK client implementing domain ports
  ports/
    tui/            → Bubble Tea views & navigation
```

Dependencies flow inward: `cmd` → `adapters`/`ports` → `domain`. Domain packages have zero external dependencies and are testable in isolation.

### Tech Stack

| Layer | Choice |
|-------|--------|
| Language | Go ≥1.22 |
| TUI | Bubble Tea v2 + Bubbles + Lip Gloss |
| Databricks SDK | `databricks-sdk-go` |
| Testing | `stretchr/testify` + `goarchtest` |
| Logging | `log/slog` (structured, file-based) |

## Requirements

- Go 1.22+
- `~/.databrickscfg` with at least a `DEFAULT` profile
- Valid Databricks credentials (PAT, OAuth M2M, Azure SP, Azure CLI, etc.)
