// Package data provides JSON/JSONL parsers for Claude Code session files.
package data

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
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

	// Collect and sort keys for deterministic output.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		s := fmt.Sprintf("%v", m[k])
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

// FormatRelativeTime returns a compact relative time string like "2m ago", "3h ago", "1d ago".
// Used for agent "last active" display where the value represents time since last file write.
func FormatRelativeTime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%dd ago", days)
}

// FormatTokens formats a token count with K/M suffixes.
// Deprecated: Use FormatTokenCount instead.
func FormatTokens(n int64) string {
	return FormatTokenCount(n)
}

// FormatTokenCount formats a token count into a compact human-readable string.
// 0 → "0", < 1000 → "999", < 1M → "245K", < 1B → "29.4M", ≥ 1B → "1.2B"
func FormatTokenCount(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		v := float64(n) / 1000
		if v < 10 {
			return fmt.Sprintf("%.1fK", v)
		}
		return fmt.Sprintf("%.0fK", v)
	}
	if n < 1_000_000_000 {
		v := float64(n) / 1_000_000
		if v < 10 {
			return fmt.Sprintf("%.1fM", v)
		}
		return fmt.Sprintf("%.0fM", v)
	}
	v := float64(n) / 1_000_000_000
	return fmt.Sprintf("%.1fB", v)
}

// jsonlUserEntry is used for parsing user-type JSONL lines.
type jsonlUserEntry struct {
	Type      string `json:"type"`
	GitBranch string `json:"gitBranch"`
	Message   *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// ParseSessionTopic reads the FIRST user message from the JSONL file.
// Returns (truncated 60 chars, full text).
// Only reads the first ~100 lines to avoid scanning the whole file.
func ParseSessionTopic(jsonlPath string) (topic string, topicFull string) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // 256KB line buffer

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > 100 {
			break
		}
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"type":"user"`)) {
			continue
		}

		var entry jsonlUserEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "user" || entry.Message == nil || entry.Message.Role != "user" {
			continue
		}

		topicFull = extractTextContent(entry.Message.Content)
		if topicFull == "" {
			continue
		}

		// Sanitize: replace newlines/tabs with spaces, collapse whitespace.
		topicFull = sanitizeOneLine(topicFull)
		topic = truncateString(topicFull, 60)
		return topic, topicFull
	}
	return "", ""
}

// sanitizeOneLine replaces newlines, tabs, and collapses whitespace into a single line.
// Uses strings.Fields to avoid quadratic behavior on long whitespace runs.
func sanitizeOneLine(s string) string {
	s = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// ParseGitBranch extracts the gitBranch field from the first user message.
func ParseGitBranch(jsonlPath string) string {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > 100 {
			break
		}
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"type":"user"`)) {
			continue
		}

		var entry jsonlUserEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "user" {
			continue
		}
		if entry.GitBranch != "" {
			return entry.GitBranch
		}
	}
	return ""
}

// ParseMessageCount counts occurrences of "type":"user" in the JSONL file.
// Uses byte-level scanning for efficiency — does NOT parse full JSON per line.
func ParseMessageCount(jsonlPath string) int {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	marker := []byte(`"type":"user"`)

	buf := make([]byte, 64*1024) // 64KB chunks
	for {
		n, err := f.Read(buf)
		if n > 0 {
			count += bytes.Count(buf[:n], marker)
		}
		if err != nil {
			break
		}
	}
	return count
}

// ParseSessionTokens sums ALL message.usage fields across the JSONL file.
func ParseSessionTokens(jsonlPath string) TokenUsage {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return TokenUsage{}
	}
	defer f.Close()

	var usage TokenUsage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	for scanner.Scan() {
		line := scanner.Bytes()
		// Fast check: skip lines without usage data.
		if !bytes.Contains(line, []byte(`"usage"`)) {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.Message.Usage == nil {
			continue
		}
		usage.InputTokens += entry.Message.Usage.InputTokens
		usage.OutputTokens += entry.Message.Usage.OutputTokens
		usage.CacheCreationTokens += entry.Message.Usage.CacheCreationInputTokens
		usage.CacheReadTokens += entry.Message.Usage.CacheReadInputTokens
	}
	return usage
}

// ParseAgentModel reads the first assistant message from JSONL, extracts
// message.model, and returns (shortName, fullName).
// Short name mapping: claude-sonnet-4-* → "sonnet", claude-haiku-4-5-* → "haiku",
// claude-opus-4-* → "opus".
func ParseAgentModel(jsonlPath string) (short string, full string) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"assistant"`)) {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.Message.Role != "assistant" || entry.Message.Model == "" {
			continue
		}
		full = entry.Message.Model
		short = modelShortName(full)
		return short, full
	}
	return "", ""
}

// ParseAgentToolCallSummary counts tool_use entries by name from the JSONL.
// Returns (toolName → count, totalCount).
func ParseAgentToolCallSummary(jsonlPath string) (map[string]int, int) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, 0
	}
	defer f.Close()

	toolCounts := make(map[string]int)
	total := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"tool_use"`)) {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.Message.Role != "assistant" || len(entry.Message.Content) == 0 {
			continue
		}

		var blocks []toolUseContent
		if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" && b.Name != "" {
				toolCounts[b.Name]++
				total++
			}
		}
	}
	if len(toolCounts) == 0 {
		return nil, 0
	}
	return toolCounts, total
}

// ParseAgentFinalOutput reads the last assistant text message from JSONL.
// Returns the last 200 chars of the text content.
func ParseAgentFinalOutput(jsonlPath string) string {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read the entire file to find the last assistant text.
	// For large files, read last 64KB.
	info, err := f.Stat()
	if err != nil {
		return ""
	}

	var data []byte
	if info.Size() > 64*1024 {
		if _, err := f.Seek(-64*1024, io.SeekEnd); err != nil {
			return ""
		}
		data, err = io.ReadAll(f)
		if err != nil {
			return ""
		}
	} else {
		data, err = io.ReadAll(f)
		if err != nil {
			return ""
		}
	}

	lines := bytes.Split(data, []byte("\n"))
	// Walk backwards to find the last assistant message with text content.
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		if !bytes.Contains(line, []byte(`"assistant"`)) {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Message == nil || entry.Message.Role != "assistant" || len(entry.Message.Content) == 0 {
			continue
		}

		text := extractAssistantText(entry.Message.Content)
		if text == "" {
			continue
		}

		if len(text) > 200 {
			text = text[len(text)-200:]
		}
		return text
	}
	return ""
}

// extractTextContent extracts text from message.content, handling both
// string content and array-of-blocks content.
func extractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Try as array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	}
	return ""
}

// extractAssistantText extracts text blocks from an assistant message's content.
// Delegates to extractTextContent — the logic is identical.
func extractAssistantText(raw json.RawMessage) string {
	return extractTextContent(raw)
}

// modelShortName maps a full model name to a short display name.
func modelShortName(full string) string {
	switch {
	case strings.HasPrefix(full, "claude-sonnet") || strings.Contains(full, "sonnet"):
		return "sonnet"
	case strings.HasPrefix(full, "claude-haiku") || strings.Contains(full, "haiku"):
		return "haiku"
	case strings.HasPrefix(full, "claude-opus") || strings.Contains(full, "opus"):
		return "opus"
	default:
		return full
	}
}

// truncateString truncates a string to maxLen chars, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// AgentJSONLInfo holds all info extracted from a single-pass parse of an agent JSONL.
type AgentJSONLInfo struct {
	Tokens          TokenUsage
	Model           string // short name
	ModelFull       string // full model string
	RecentToolCalls []ToolCall
	ToolCallMap     map[string]int
	ToolCallTotal   int
	FinalOutput     string
}

// ParseAgentJSONLFull does a single-pass read of an agent JSONL file and extracts
// all useful information: tokens, model, tool calls, and final output.
// For files > 2MB, only reads the last 512KB for tool calls/output but scans fully for tokens.
func ParseAgentJSONLFull(path string) AgentJSONLInfo {
	info := AgentJSONLInfo{
		ToolCallMap: make(map[string]int),
	}

	f, err := os.Open(path)
	if err != nil {
		return info
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)

	var lastAssistantText string
	var recentCalls []ToolCall

	for scanner.Scan() {
		line := scanner.Bytes()

		// Fast pre-check: skip lines that don't have useful data.
		hasUsage := bytes.Contains(line, []byte(`"usage"`))
		hasModel := bytes.Contains(line, []byte(`"model"`))
		hasToolUse := bytes.Contains(line, []byte(`"tool_use"`))
		hasAssistant := bytes.Contains(line, []byte(`"assistant"`))

		if !hasUsage && !hasModel && !hasToolUse && !hasAssistant {
			continue
		}

		// Parse any line that passed the pre-filter.
		// Use a single struct that covers all fields we need.
		var entry struct {
			Message *struct {
				Usage *struct {
					InputTokens              int64 `json:"input_tokens"`
					OutputTokens             int64 `json:"output_tokens"`
					CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
					CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
				} `json:"usage"`
				Model   string          `json:"model"`
				Content json.RawMessage `json:"content"`
				Role    string          `json:"role"`
			} `json:"message"`
		}
		if err := json.Unmarshal(line, &entry); err != nil || entry.Message == nil {
			continue
		}

		// Extract tokens.
		if entry.Message.Usage != nil {
			info.Tokens.InputTokens += entry.Message.Usage.InputTokens
			info.Tokens.OutputTokens += entry.Message.Usage.OutputTokens
			info.Tokens.CacheCreationTokens += entry.Message.Usage.CacheCreationInputTokens
			info.Tokens.CacheReadTokens += entry.Message.Usage.CacheReadInputTokens
		}

		// Extract model (first occurrence wins).
		if info.ModelFull == "" && entry.Message.Model != "" {
			info.ModelFull = entry.Message.Model
			info.Model = modelShortName(entry.Message.Model)
		}

		// Extract tool calls from content.
		if hasToolUse && entry.Message.Content != nil {
			calls := extractToolCalls(entry.Message.Content)
			for _, c := range calls {
				info.ToolCallMap[c.Name]++
				info.ToolCallTotal++
			}
			recentCalls = append(recentCalls, calls...)
			// Keep only last 10 tool calls in memory.
			if len(recentCalls) > 10 {
				recentCalls = recentCalls[len(recentCalls)-10:]
			}
		}

		// Track last assistant text for final output.
		if entry.Message.Role == "assistant" && entry.Message.Content != nil {
			text := extractAssistantText(entry.Message.Content)
			if text != "" {
				lastAssistantText = text
			}
		}
	}

	info.RecentToolCalls = recentCalls

	// Final output: last 200 chars.
	if len(lastAssistantText) > 200 {
		runes := []rune(lastAssistantText)
		if len(runes) > 200 {
			info.FinalOutput = string(runes[len(runes)-200:])
		} else {
			info.FinalOutput = lastAssistantText
		}
	} else {
		info.FinalOutput = lastAssistantText
	}

	return info
}
