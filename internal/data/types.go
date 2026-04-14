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

	// v1.1 fields — extracted from session JSONL.
	Topic        string     `json:"-"` // first user message (truncated 60 chars)
	TopicFull    string     `json:"-"` // first user message (full)
	LastMessage  string     `json:"-"` // last user message (truncated)
	GitBranch    string     `json:"-"` // from JSONL gitBranch field
	MessageCount int        `json:"-"` // count of user messages
	Tokens       TokenUsage `json:"-"` // session-level token totals
	JSONLPath    string     `json:"-"` // path to session JSONL
	JSONLSize    int64      `json:"-"` // file size for display
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
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadTokens     int64 `json:"cache_read_input_tokens"`
}

// Total returns the sum of all token fields.
func (t TokenUsage) Total() int64 {
	return t.InputTokens + t.OutputTokens + t.CacheCreationTokens + t.CacheReadTokens
}

// Formatted returns a human-readable formatted total (e.g., "245K", "29.4M").
func (t TokenUsage) Formatted() string {
	return FormatTokenCount(t.Total())
}

// ToolCall represents a single tool invocation extracted from JSONL.
type ToolCall struct {
	Name  string
	Input string // short summary of input
}

// Agent represents a fully resolved subagent with computed fields.
type Agent struct {
	ID          string      // e.g. "a1370bcbcce67f63d"
	SessionID   string      // parent session ID
	AgentType   string      // from meta.json
	Description string      // from meta.json
	Status      AgentStatus // computed from JSONL mtime
	Model       string      // short model name: "sonnet", "haiku", "opus"
	ModelFull   string      // full model name: "claude-haiku-4-5-20251001"
	Tokens      TokenUsage  // aggregated from JSONL
	ToolCalls   []ToolCall  // latest tool calls (from tail of JSONL)
	ToolCallMap   map[string]int // tool name → call count
	ToolCallTotal int            // total tool calls
	FinalOutput   string         // last 200 chars of agent's result
	Elapsed       string      // human-readable time since last activity
	JSONLPath     string      // path to .jsonl file for mtime checks
	MetaPath      string      // path to .meta.json
	// LastActiveTime is the time of the last JSONL write (file mtime),
	// not the actual agent start time. Used to compute Elapsed.
	LastActiveTime time.Time
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
	// TotalTokensAll is the sum of all token types across all sessions.
	TotalTokensAll int64
}
