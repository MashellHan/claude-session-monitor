# Claude Session Monitor (CSM)

A terminal UI (TUI) for monitoring Claude Code sessions, subagents, and tasks in real time.

## Features

- **Session tracking** - View all active and recent Claude Code sessions with PID, project, kind, and uptime
- **Subagent monitoring** - See subagents per session with type, status (running/idle/done), elapsed time, model, and token usage
- **Task tracking** - View tasks per session with status, dependencies, and subjects
- **Live updates** - Auto-refreshes via fsnotify file watching with 2s polling fallback
- **Token metrics** - Aggregated input/output/cache token counts across all agents
- **Keyboard navigation** - Three-panel layout with cursor navigation, tab switching, and inline detail expansion

## Installation

```bash
go install github.com/MashellHan/claude-session-monitor/cmd/csm@latest
```

Or build from source:

```bash
make build
./csm
```

## Usage

```bash
# Launch the monitor (reads from ~/.claude/ by default)
csm

# Specify a custom Claude data directory
csm --claude-dir /path/to/.claude

# Show version
csm --version
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate within panel |
| `Tab` | Switch between panels |
| `1` / `2` / `3` | Jump to Sessions / Agents / Tasks panel |
| `Enter` | Expand/collapse agent detail |
| `f` | Toggle active-only filter |
| `r` | Manual refresh |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

## Architecture

```
cmd/csm/main.go          Entry point
internal/data/            Parsers, scanner, watcher, types
internal/store/           Thread-safe in-memory state store
internal/model/           Bubble Tea models (app, session list, agent table, task list)
internal/ui/              Styles, key bindings, help view
```

## Development

```bash
make build    # Build binary
make test     # Run tests with race detector
make vet      # Run go vet
make clean    # Clean build artifacts
```

## Requirements

- Go 1.21+
- macOS (primary target; Linux should work but is untested)

## License

MIT
