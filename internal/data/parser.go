// Package data provides JSON/JSONL parsers for Claude Code session files.
package data

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ParseSession reads a session JSON file and returns a Session struct.
// Returns an error if the file cannot be read or parsed.
func ParseSession(path string) (Session, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("read session file %s: %w", path, err)
	}
	return ParseSessionJSON(raw)
}

// ParseSessionJSON parses session JSON bytes into a Session struct.
func ParseSessionJSON(raw []byte) (Session, error) {
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return Session{}, fmt.Errorf("parse session JSON: %w", err)
	}
	// Convert StartedAt (Unix ms) to Time.
	if s.StartedAt > 0 {
		s.StartTime = time.UnixMilli(s.StartedAt)
	}
	return s, nil
}

// ParseTask reads a task JSON file and returns a Task struct.
func ParseTask(path string) (Task, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Task{}, fmt.Errorf("read task file %s: %w", path, err)
	}
	return ParseTaskJSON(raw)
}

// ParseTaskJSON parses task JSON bytes into a Task struct.
func ParseTaskJSON(raw []byte) (Task, error) {
	var t Task
	if err := json.Unmarshal(raw, &t); err != nil {
		return Task{}, fmt.Errorf("parse task JSON: %w", err)
	}
	return t, nil
}

// ParseAgentMeta reads a subagent meta.json file.
func ParseAgentMeta(path string) (AgentMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AgentMeta{}, fmt.Errorf("read agent meta %s: %w", path, err)
	}
	return ParseAgentMetaJSON(raw)
}

// ParseAgentMetaJSON parses agent meta JSON bytes.
func ParseAgentMetaJSON(raw []byte) (AgentMeta, error) {
	var m AgentMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return AgentMeta{}, fmt.Errorf("parse agent meta JSON: %w", err)
	}
	return m, nil
}

// jsonlEntry is an intermediate struct for parsing JSONL lines.
// We only extract the fields we need for token usage, tool calls, and model.
type jsonlEntry struct {
	AgentID string `json:"agentId"`
	Message *struct {
		Role    string          `json:"role"`
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// toolUseContent is used to extract tool_use blocks from message content.
type toolUseContent struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ParseJSONLTail reads the last maxBytes of a JSONL file and extracts
// token usage, the model name, and recent tool calls.
// This avoids loading entire large JSONL files into memory.
func ParseJSONLTail(path string, maxBytes int64) (TokenUsage, string, []ToolCall, error) {
	f, err := os.Open(path)
	if err != nil {
		return TokenUsage{}, "", nil, fmt.Errorf("open JSONL %s: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return TokenUsage{}, "", nil, fmt.Errorf("stat JSONL %s: %w", path, err)
	}

	size := info.Size()
	readSize := size
	if readSize > maxBytes {
		readSize = maxBytes
		// Seek to (size - maxBytes) from the start.
		if _, err := f.Seek(size-maxBytes, io.SeekStart); err != nil {
			return TokenUsage{}, "", nil, fmt.Errorf("seek JSONL %s: %w", path, err)
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return TokenUsage{}, "", nil, fmt.Errorf("read JSONL tail %s: %w", path, err)
	}

	return parseJSONLBytes(data)
}

// ParseJSONLFull reads an entire JSONL file. Used for small files and testing.
func ParseJSONLFull(path string) (TokenUsage, string, []ToolCall, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TokenUsage{}, "", nil, fmt.Errorf("read JSONL %s: %w", path, err)
	}
	return parseJSONLBytes(data)
}

// parseJSONLBytes processes raw JSONL bytes and extracts token usage,
// model name, and tool calls.
func parseJSONLBytes(data []byte) (TokenUsage, string, []ToolCall, error) {
	var usage TokenUsage
	var model string
	var calls []ToolCall

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines — might be a partial line from tail read.
			continue
		}

		if entry.Message == nil {
			continue
		}

		// Extract model name (use the latest non-empty one).
		if entry.Message.Model != "" {
			model = entry.Message.Model
		}

		// Aggregate token usage.
		if entry.Message.Usage != nil {
			usage.InputTokens += entry.Message.Usage.InputTokens
			usage.OutputTokens += entry.Message.Usage.OutputTokens
			usage.CacheCreationTokens += entry.Message.Usage.CacheCreationInputTokens
			usage.CacheReadTokens += entry.Message.Usage.CacheReadInputTokens
		}

		// Extract tool calls from assistant messages.
		if entry.Message.Role == "assistant" && len(entry.Message.Content) > 0 {
			calls = append(calls, extractToolCalls(entry.Message.Content)...)
		}
	}

	// Keep only the last 10 tool calls for display.
	if len(calls) > 10 {
		calls = calls[len(calls)-10:]
	}

	return usage, model, calls, nil
}

// extractToolCalls parses the content array looking for tool_use blocks.
func extractToolCalls(contentRaw json.RawMessage) []ToolCall {
	// Content can be a string or an array of content blocks.
	// Try array first.
	var blocks []toolUseContent
	if err := json.Unmarshal(contentRaw, &blocks); err != nil {
		return nil
	}

	var calls []ToolCall
	for _, b := range blocks {
		if b.Type == "tool_use" && b.Name != "" {
			inputSummary := summarizeInput(b.Input)
			calls = append(calls, ToolCall{
				Name:  b.Name,
				Input: inputSummary,
			})
		}
	}
	return calls
}

// summarizeInput creates a short human-readable summary of tool input.
func summarizeInput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}

	var parts []string
	for k, v := range m {
		s := fmt.Sprintf("%v", v)
		if len(s) > 50 {
			s = s[:47] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, s))
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, " ")
}

// ParseProjectsJSON reads the homunculus projects.json and returns a
// map from CWD path to project name.
func ParseProjectsJSON(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read projects.json %s: %w", path, err)
	}
	return ParseProjectsJSONBytes(raw)
}

// ParseProjectsJSONBytes parses projects.json bytes. The format is an array
// of project entries, or possibly a map — we handle both.
func ParseProjectsJSONBytes(raw []byte) (map[string]string, error) {
	result := make(map[string]string)

	// Try array format first: [{"name":"...", "path":"...", ...}, ...]
	var entries []ProjectEntry
	if err := json.Unmarshal(raw, &entries); err == nil {
		for _, e := range entries {
			if e.Path != "" && e.Name != "" {
				result[e.Path] = e.Name
			}
		}
		return result, nil
	}

	// Try map format: {"path": {"name": "...", ...}}
	var mapFormat map[string]ProjectEntry
	if err := json.Unmarshal(raw, &mapFormat); err == nil {
		for path, e := range mapFormat {
			name := e.Name
			if name == "" {
				name = path
			}
			result[path] = name
		}
		return result, nil
	}

	// Try simple map format: {"path": "name", ...}
	var simpleMap map[string]string
	if err := json.Unmarshal(raw, &simpleMap); err == nil {
		return simpleMap, nil
	}

	return nil, fmt.Errorf("unrecognized projects.json format")
}

// PathToHash converts a filesystem path to the path hash format used by
// Claude Code in ~/.claude/projects/. Both forward slashes and dots are
// replaced with dashes.
// Example: "/Users/foo/.bar/baz" → "-Users-foo--bar-baz"
func PathToHash(path string) string {
	r := strings.ReplaceAll(path, "/", "-")
	r = strings.ReplaceAll(r, ".", "-")
	return r
}

// FormatUptime returns a human-readable duration string like "2h 15m".
func FormatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %02dm", hours, mins)
}

// FormatTokens formats a token count with K/M suffixes.
func FormatTokens(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
