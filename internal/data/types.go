// Package data provides types and utilities for reading Claude Code session
// data from the filesystem.
package data

import "time"

// AgentStatus represents the current state of a subagent.
type AgentStatus int

const (
	StatusUnknown AgentStatus = iota
	StatusRunning             // JSONL mtime < 30s
	StatusIdle                // JSONL mtime < 5min
	StatusDone                // JSONL mtime >= 5min
)

// String returns a human-readable label for AgentStatus.
func (s AgentStatus) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusIdle:
		return "idle"
	case StatusDone:
		return "done"
	default:
		return "unknown"
	}
}

// Symbol returns a single-character status indicator.
func (s AgentStatus) Symbol() string {
	switch s {
	case StatusRunning:
		return "●"
	case StatusIdle:
		return "◐"
	case StatusDone:
		return "✓"
	default:
		return "○"
	}
}

// TaskStatus represents the state of a task.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
)

// Session holds metadata about a Claude Code session, read from
// ~/.claude/sessions/{pid}.json.
type Session struct {
	PID        int       `json:"pid"`
	SessionID  string    `json:"sessionId"`
	CWD        string    `json:"cwd"`
	StartedAt  int64     `json:"startedAt"` // Unix milliseconds
	Kind       string    `json:"kind"`      // "interactive" | "background" | "batch"
	Entrypoint string    `json:"entrypoint"`
	Alive      bool      `json:"-"` // true if PID is running
	Uptime     string    `json:"-"` // computed human-readable uptime
	Project    string    `json:"-"` // resolved project name
	PathHash   string    `json:"-"` // CWD encoded as path hash
	StartTime  time.Time `json:"-"` // parsed from StartedAt
}

// Task holds metadata about a task, read from
// ~/.claude/tasks/{sessionId}/{taskId}.json.
type Task struct {
	ID          string     `json:"id"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	ActiveForm  string     `json:"activeForm"`
	Status      TaskStatus `json:"status"`
	Blocks      []string   `json:"blocks"`
	BlockedBy   []string   `json:"blockedBy"`
	SessionID   string     `json:"-"` // set by scanner
}

// AgentMeta holds the metadata from a subagent's meta.json file.
type AgentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
}

// TokenUsage aggregates token counts from JSONL entries.
type TokenUsage struct {
	InputTokens          int64 `json:"input_tokens"`
	OutputTokens         int64 `json:"output_tokens"`
	CacheCreationTokens  int64 `json:"cache_creation_input_tokens"`
	CacheReadTokens      int64 `json:"cache_read_input_tokens"`
}

// ToolCall represents a single tool invocation extracted from JSONL.
type ToolCall struct {
	Timestamp time.Time
	Name      string
	Input     string // short summary of input
}

// Agent represents a fully resolved subagent with computed fields.
type Agent struct {
	ID          string      // e.g. "a1370bcbcce67f63d"
	SessionID   string      // parent session ID
	AgentType   string      // from meta.json
	Description string      // from meta.json
	Status      AgentStatus // computed from JSONL mtime
	Model       string      // extracted from JSONL
	Tokens      TokenUsage  // aggregated from JSONL
	ToolCalls   []ToolCall  // latest tool calls (from tail of JSONL)
	Elapsed     string      // human-readable elapsed time
	JSONLPath   string      // path to .jsonl file for mtime checks
	MetaPath    string      // path to .meta.json
	StartTime   time.Time   // approximated from first JSONL entry or mtime
}

// ProjectEntry represents an entry from ~/.claude/homunculus/projects.json.
type ProjectEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	GitRemote string `json:"gitRemote"`
}

// Stats holds aggregate metrics for the summary bar.
type Stats struct {
	TotalSessions      int
	ActiveSessions     int
	TotalAgents        int
	RunningAgents      int
	IdleAgents         int
	DoneAgents         int
	TotalTasks         int
	CompletedTasks     int
	InProgressTasks    int
	PendingTasks       int
	TotalTokensIn      int64
	TotalTokensOut     int64
	TotalCacheCreation int64
	TotalCacheRead     int64
}
