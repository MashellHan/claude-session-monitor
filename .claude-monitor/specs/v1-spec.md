# Claude Session Monitor TUI — v1 Spec

> Lead: Architecture & Design
> Version: v1.0
> Date: 2026-04-14

---

## 1. Overview

A terminal TUI tool that provides real-time monitoring of all Claude Code sessions and their subagents running on the local machine. It reads Claude Code's on-disk session data (no API, no WebSocket — purely filesystem-based) and presents a live dashboard.

### Project Name: `claude-session-monitor` (binary: `csm`)

---

## 2. Data Source Architecture

### 2.1 Data Discovery

Claude Code stores all session data under `~/.claude/`. The monitor reads these data sources:

| Data Source | Path | Format | Purpose |
|-------------|------|--------|---------|
| Session registry | `~/.claude/sessions/{pid}.json` | JSON | Session metadata (pid, sessionId, cwd, startedAt, kind) |
| Task files | `~/.claude/tasks/{sessionId}/*.json` | JSON | Task status, subject, description, dependencies |
| Subagent meta | `~/.claude/projects/{pathHash}/{sessionId}/subagents/agent-{id}.meta.json` | JSON | Agent type, description |
| Subagent log | `~/.claude/projects/{pathHash}/{sessionId}/subagents/agent-{id}.jsonl` | JSONL | Full conversation log with tool calls, token usage |
| Session log | `~/.claude/projects/{pathHash}/{sessionId}.jsonl` | JSONL | Main session conversation log |
| Project registry | `~/.claude/homunculus/projects.json` | JSON | Project names, paths, git remotes |
| Bash commands | `~/.claude/bash-commands.log` | Text log | All executed bash commands with timestamps |
| Cost tracker | `~/.claude/cost-tracker.log` | Text log | Tool invocations |

### 2.2 Session File Schema

```json
// ~/.claude/sessions/{pid}.json
{
  "pid": 4394,
  "sessionId": "cc7b2bf4-d66e-4fda-a240-6c57c1e0fd4a",
  "cwd": "/Users/mengxionghan/.superset/projects/Tmp",
  "startedAt": 1776174878779,  // Unix ms
  "kind": "interactive",        // "interactive" | "background" | "batch"
  "entrypoint": "cli"
}
```

### 2.3 Task File Schema

```json
// ~/.claude/tasks/{sessionId}/{taskId}.json
{
  "id": "1",
  "subject": "Research Claude Code session data sources",
  "description": "Investigate how Claude Code stores...",
  "activeForm": "Researching Claude Code data sources",
  "status": "in_progress",  // "pending" | "in_progress" | "completed"
  "blocks": ["2"],
  "blockedBy": []
}
```

### 2.4 Subagent Meta Schema

```json
// ~/.claude/projects/{pathHash}/{sessionId}/subagents/agent-{id}.meta.json
{
  "agentType": "Explore",
  "description": "Explore exports.ts import chain"
}
```

### 2.5 Subagent JSONL Entry Schema

Each line in the agent JSONL is one of:
```json
// User prompt to agent
{
  "parentUuid": null,
  "isSidechain": true,
  "promptId": "...",
  "agentId": "a1370bcbcce67f63d",
  "type": "user",
  "message": { "role": "user", "content": "..." }
}

// Agent response with tool calls
{
  "parentUuid": "...",
  "isSidechain": true,
  "agentId": "a1370bcbcce67f63d",
  "message": {
    "type": "message",
    "role": "assistant",
    "content": [{ "type": "tool_use", "name": "Grep", "input": {...} }],
    "model": "claude-haiku-4-5-20251001",
    "usage": { "input_tokens": 1234, "output_tokens": 567 }
  }
}
```

### 2.6 Active Session Detection

A session is **active** if:
1. The session file exists at `~/.claude/sessions/{pid}.json`
2. The process with that PID is still running: `kill -0 {pid}` returns 0
3. (Optional) The JSONL file was modified within the last N seconds

Subagent liveness: A subagent is **running** if its `.jsonl` file is still being written to (mtime within last 30s).

---

## 3. Tech Stack

### Language: Go

**Rationale:**
- Native binary — no runtime dependencies
- Excellent `fsnotify` ecosystem for file watching
- Bubble Tea is the gold standard for terminal TUI in Go
- Low memory footprint ideal for a background monitor
- User preference from previous TUI projects (Bubble Tea + lipgloss)

### Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/charmbracelet/bubbletea` | latest | TUI framework |
| `github.com/charmbracelet/lipgloss` | latest | Styling |
| `github.com/charmbracelet/bubbles` | latest | Table, viewport, spinner components |
| `github.com/fsnotify/fsnotify` | latest | File system watching |

### Build

```bash
go build -o csm ./cmd/csm
```

---

## 4. Architecture

### 4.1 Component Diagram

```
┌─────────────────────────────────────────────────────┐
│                    TUI Layer                         │
│  ┌─────────────┐ ┌──────────────┐ ┌──────────────┐  │
│  │ SessionList  │ │ AgentTable   │ │ DetailView   │  │
│  │   View       │ │   View       │ │   (Viewport) │  │
│  └──────┬───────┘ └──────┬───────┘ └──────┬───────┘  │
│         └────────┬───────┴────────┬───────┘          │
│                  │                │                   │
│          ┌───────▼────────┐      │                   │
│          │   App Model    │◄─────┘                   │
│          │  (Bubble Tea)  │                          │
│          └───────┬────────┘                          │
└──────────────────┼───────────────────────────────────┘
                   │
         ┌─────────▼──────────┐
         │   Data Layer       │
         │  ┌──────────────┐  │
         │  │  Watcher     │  │  fsnotify + polling
         │  │  (fs events) │  │
         │  └──────┬───────┘  │
         │         │          │
         │  ┌──────▼───────┐  │
         │  │  Parser      │  │  JSON / JSONL parsing
         │  │  (sessions,  │  │
         │  │   tasks,     │  │
         │  │   agents)    │  │
         │  └──────┬───────┘  │
         │         │          │
         │  ┌──────▼───────┐  │
         │  │  Store        │  │  In-memory state
         │  │  (sessions,  │  │
         │  │   agents,    │  │
         │  │   metrics)   │  │
         │  └──────────────┘  │
         └────────────────────┘
```

### 4.2 Package Layout

```
claude-session-monitor/
├── cmd/
│   └── csm/
│       └── main.go              # Entry point
├── internal/
│   ├── model/
│   │   ├── app.go               # Root Bubble Tea model
│   │   ├── session_list.go      # Session list view
│   │   ├── agent_table.go       # Agent table view
│   │   ├── detail_view.go       # Detail/log viewport
│   │   └── summary_bar.go      # Bottom summary bar
│   ├── data/
│   │   ├── types.go             # Data types (Session, Task, Agent, etc.)
│   │   ├── scanner.go           # Initial full scan of ~/.claude/
│   │   ├── watcher.go           # fsnotify-based file watcher
│   │   └── parser.go            # JSON/JSONL parsers
│   ├── store/
│   │   └── store.go             # In-memory state store
│   └── ui/
│       ├── styles.go            # Lipgloss styles
│       ├── keys.go              # Key bindings
│       └── help.go              # Help overlay
├── .claude-monitor/
│   ├── specs/
│   │   └── v1-spec.md           # This file
│   └── reviews/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 5. TUI Layout

### 5.1 Main Dashboard View

```
┌─ Claude Session Monitor ─────────────────────────────────────────────┐
│                                                                       │
│  Sessions (20 active)                                                 │
│  ┌─────┬──────────────────────┬──────────┬─────────┬────────────────┐ │
│  │ PID │ Project              │ Kind     │ Uptime  │ Agents         │ │
│  ├─────┼──────────────────────┼──────────┼─────────┼────────────────┤ │
│  │▸4394│ Tmp                  │ interact │ 2h 15m  │ 3 running      │ │
│  │ 7183│ claude-statusline-hud│ interact │ 1h 30m  │ 1 running      │ │
│  │64783│ Tmp                  │ interact │ 5h 02m  │ 0 idle         │ │
│  │ ...                                                               │ │
│  └─────┴──────────────────────┴──────────┴─────────┴────────────────┘ │
│                                                                       │
│  Subagents for Session 4394 (3 agents)                               │
│  ┌──────────────────┬──────────┬─────────┬──────────────────────────┐ │
│  │ Agent Type       │ Status   │ Elapsed │ Description              │ │
│  ├──────────────────┼──────────┼─────────┼──────────────────────────┤ │
│  │ Explore          │ ● running│ 45s     │ Research Claude Code data│ │
│  │ claude-code-guide│ ● running│ 1m 30s  │ Research Claude Code docs│ │
│  │ general-purpose  │ ✓ done   │ 3m 15s  │ Build Python Textual TUI │ │
│  └──────────────────┴──────────┴─────────┴──────────────────────────┘ │
│                                                                       │
│  Tasks (4 total: 1 completed, 1 in_progress, 2 pending)             │
│  ┌────┬─────────┬────────────────────────────────────────────┬──────┐ │
│  │ ID │ Status  │ Subject                                    │ Deps │ │
│  ├────┼─────────┼────────────────────────────────────────────┼──────┤ │
│  │  1 │ ✓ done  │ Research Claude Code session data sources  │      │ │
│  │  2 │ ● work  │ Architecture design and v1 spec            │ ←1   │ │
│  │  3 │ ○ pend  │ Dev: Implement v1 based on spec            │ ←2   │ │
│  │  4 │ ○ pend  │ Lead: Review v1 implementation             │ ←3   │ │
│  └────┴─────────┴────────────────────────────────────────────┴──────┘ │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│ 20 sessions │ 4 agents (2 running) │ ↑↓ navigate │ enter detail │ ? │
└───────────────────────────────────────────────────────────────────────┘
```

### 5.2 Detail View (Enter on agent row)

Inline expansion below the selected row (no modal — per user preference):

```
│  │ Explore          │ ● running│ 45s     │ Research Claude Code data│ │
│  │ ┌─ Detail ─────────────────────────────────────────────────────┐ │ │
│  │ │ Agent ID: a1370bcbcce67f63d                                  │ │ │
│  │ │ Model: claude-haiku-4-5-20251001                             │ │ │
│  │ │ Tokens: 12,345 in / 5,678 out                                │ │ │
│  │ │                                                               │ │ │
│  │ │ Latest activity:                                              │ │ │
│  │ │ [22:10:15] → Grep: pattern="from.*exports" type=ts           │ │ │
│  │ │ [22:10:17] → Read: /src/index.ts                             │ │ │
│  │ │ [22:10:20] → Glob: pattern="**/*.test.ts"                    │ │ │
│  │ └──────────────────────────────────────────────────────────────┘ │ │
│  │ claude-code-guide│ ● running│ 1m 30s  │ Research Claude Code docs│ │
```

### 5.3 Summary Bar

Fixed bottom bar showing aggregated metrics:

```
│ 20 sessions │ 12 agents (4 running, 2 idle, 6 done) │ 45.2K tokens │ ↑↓ nav │ enter expand │ q quit │ ? help │
```

---

## 6. Key Bindings

| Key | Action |
|-----|--------|
| `↑`/`k` | Move cursor up |
| `↓`/`j` | Move cursor down |
| `Enter` | Toggle inline detail expansion |
| `Tab` | Switch focus: Sessions → Agents → Tasks |
| `1`/`2`/`3` | Jump to panel: 1=Sessions, 2=Agents, 3=Tasks |
| `r` | Force refresh |
| `f` | Filter: show only active sessions |
| `/` | Search/filter by text |
| `q`/`Ctrl+C` | Quit |
| `?` | Toggle help |

---

## 7. Data Flow

### 7.1 Startup Sequence

```
1. Scan all ~/.claude/sessions/*.json
2. For each session, check if PID is alive (kill -0)
3. Active sessions: load their tasks from ~/.claude/tasks/{sessionId}/
4. Active sessions: discover subagents from ~/.claude/projects/{pathHash}/{sessionId}/subagents/
5. Parse subagent meta.json files for type/description
6. For running agents: tail last N lines of .jsonl for latest activity
7. Resolve project names from ~/.claude/homunculus/projects.json
8. Build in-memory store
9. Start fsnotify watchers on:
   - ~/.claude/sessions/        (new/removed sessions)
   - ~/.claude/tasks/           (task updates)
   - ~/.claude/projects/        (subagent activity)
10. Render initial TUI
```

### 7.2 Live Update Loop

```
fsnotify event or polling tick (every 2s)
  │
  ├── Session file changed → re-check PID liveness
  ├── Task file changed → re-parse task JSON, update store
  ├── Subagent meta.json created → new agent discovered
  ├── Subagent .jsonl modified → agent is active, parse latest entries
  └── Agent .jsonl stale (>30s) → mark agent as idle/done
  │
  └── Send Bubble Tea message → Update() → re-render
```

### 7.3 Agent Status Detection

```go
func detectAgentStatus(agentDir string, agentId string) AgentStatus {
    jsonlPath := filepath.Join(agentDir, "agent-"+agentId+".jsonl")
    info, err := os.Stat(jsonlPath)
    if err != nil {
        return StatusUnknown
    }

    age := time.Since(info.ModTime())
    if age < 30*time.Second {
        return StatusRunning
    }
    if age < 5*time.Minute {
        return StatusIdle
    }
    return StatusDone
}
```

---

## 8. Token Usage Extraction

From subagent JSONL entries with `message.usage`:

```json
{
  "message": {
    "usage": {
      "input_tokens": 1234,
      "output_tokens": 567,
      "cache_creation_input_tokens": 0,
      "cache_read_input_tokens": 890
    }
  }
}
```

Aggregate per-agent and per-session by summing all `usage` fields across JSONL entries.

---

## 9. Implementation Phases

### Phase 1: Core (v1.0)
- [ ] Project scaffolding (go mod, cmd/, internal/)
- [ ] Data types and parser for all JSON/JSONL schemas
- [ ] Full scanner: discover sessions, tasks, agents
- [ ] PID liveness check
- [ ] In-memory store
- [ ] Bubble Tea app model with 3-panel layout
- [ ] Session list view with cursor navigation
- [ ] Agent table view per selected session
- [ ] Task list view per selected session
- [ ] Summary bar with aggregate metrics
- [ ] Inline detail expansion on Enter
- [ ] fsnotify watcher for live updates
- [ ] Polling fallback (2s interval)
- [ ] Auto-refresh indicator in UI

### Phase 2: Polish (v1.1)
- [ ] Token usage aggregation
- [ ] Search/filter functionality
- [ ] Sort by various columns
- [ ] Color-coded status indicators
- [ ] Elapsed time auto-update
- [ ] Help overlay (?)
- [ ] CLI flags: `--session <id>`, `--project <name>`, `--watch-interval <seconds>`

### Phase 3: Advanced (v2.0)
- [ ] Agent conversation viewer (full JSONL log viewer)
- [ ] Bash command log tailing
- [ ] Export session report (JSON/Markdown)
- [ ] Notification on agent completion
- [ ] Cross-session agent comparison

---

## 10. CLI Interface

```bash
# Default: monitor all active sessions
csm

# Monitor specific session
csm --session cc7b2bf4-d66e-4fda-a240-6c57c1e0fd4a

# Monitor sessions for a specific project
csm --project Tmp

# Monitor only the current terminal's session
csm --current

# Set custom refresh interval
csm --interval 5s

# Show version
csm --version
```

---

## 11. Error Handling

- **Missing directories**: Gracefully handle if `~/.claude/` doesn't exist
- **Permission errors**: Skip unreadable files, log warning
- **Malformed JSON**: Skip bad entries, show parse error count in status bar
- **Dead PIDs**: Clean up from active list on next poll
- **Race conditions**: File might be written to while reading — use read-retry with backoff
- **Large JSONL files**: Only read last N bytes (tail), don't load entire file

---

## 12. Performance Constraints

- **Memory**: < 50MB RSS for monitoring 20+ sessions
- **CPU**: < 1% idle, < 5% during refresh
- **Startup**: < 500ms to first render
- **Refresh**: < 100ms per update cycle
- **JSONL tail**: Read only last 8KB of large files for latest activity

---

## 13. Testing Strategy

- **Unit tests**: Parser tests with fixture JSON/JSONL files
- **Integration tests**: Scanner with mock `~/.claude/` directory
- **Snapshot tests**: TUI rendering with golden files
- **Target**: 80%+ coverage

---

## 14. Acceptance Criteria

1. `csm` launches and displays all active Claude Code sessions within 500ms
2. Each session shows: PID, project name, kind, uptime, agent count
3. Selecting a session shows its subagents with type, status, elapsed time, description
4. Selecting a session shows its tasks with status, subject, dependencies
5. Inline detail expansion shows agent ID, model, token usage, latest tool calls
6. UI auto-refreshes when agent/task files change on disk
7. Summary bar shows aggregate metrics
8. `q` or `Ctrl+C` cleanly exits
9. No panics on malformed data or missing files
10. Works on macOS (primary target)
