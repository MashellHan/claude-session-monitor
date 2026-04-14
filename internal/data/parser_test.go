package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureDir returns the path to the test fixtures directory.
func fixtureDir(t *testing.T) string {
	t.Helper()
	// Navigate from internal/data/ to project root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	return filepath.Join(wd, "..", "..", "testdata", "fixtures")
}

// --- Session Parsing Tests ---

func TestParseSessionJSON_Valid(t *testing.T) {
	raw := []byte(`{
		"pid": 4394,
		"sessionId": "cc7b2bf4-d66e-4fda-a240-6c57c1e0fd4a",
		"cwd": "/Users/test/project",
		"startedAt": 1776174878779,
		"kind": "interactive",
		"entrypoint": "cli"
	}`)

	sess, err := ParseSessionJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sess.PID != 4394 {
		t.Errorf("PID = %d, want 4394", sess.PID)
	}
	if sess.SessionID != "cc7b2bf4-d66e-4fda-a240-6c57c1e0fd4a" {
		t.Errorf("SessionID = %q, want cc7b2bf4-d66e-4fda-a240-6c57c1e0fd4a", sess.SessionID)
	}
	if sess.CWD != "/Users/test/project" {
		t.Errorf("CWD = %q, want /Users/test/project", sess.CWD)
	}
	if sess.Kind != "interactive" {
		t.Errorf("Kind = %q, want interactive", sess.Kind)
	}
	if sess.StartedAt != 1776174878779 {
		t.Errorf("StartedAt = %d, want 1776174878779", sess.StartedAt)
	}
	if sess.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
}

func TestParseSessionJSON_Invalid(t *testing.T) {
	raw := []byte(`{broken json`)
	_, err := ParseSessionJSON(raw)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseSessionJSON_Empty(t *testing.T) {
	raw := []byte(`{}`)
	sess, err := ParseSessionJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.PID != 0 {
		t.Errorf("PID = %d, want 0", sess.PID)
	}
	if sess.StartTime.IsZero() != true {
		t.Error("StartTime should be zero for 0 startedAt")
	}
}

func TestParseSession_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	sess, err := ParseSession(filepath.Join(dir, "sessions", "12345.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.PID != 12345 {
		t.Errorf("PID = %d, want 12345", sess.PID)
	}
	if sess.SessionID != "test-session-001" {
		t.Errorf("SessionID = %q, want test-session-001", sess.SessionID)
	}
}

func TestParseSession_MissingFile(t *testing.T) {
	_, err := ParseSession("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// --- Task Parsing Tests ---

func TestParseTaskJSON_Valid(t *testing.T) {
	raw := []byte(`{
		"id": "1",
		"subject": "Test task",
		"description": "Test description",
		"activeForm": "Testing",
		"status": "in_progress",
		"blocks": ["2", "3"],
		"blockedBy": ["0"]
	}`)

	task, err := ParseTaskJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "1" {
		t.Errorf("ID = %q, want 1", task.ID)
	}
	if task.Subject != "Test task" {
		t.Errorf("Subject = %q, want 'Test task'", task.Subject)
	}
	if task.Status != TaskInProgress {
		t.Errorf("Status = %q, want in_progress", task.Status)
	}
	if len(task.Blocks) != 2 {
		t.Errorf("Blocks len = %d, want 2", len(task.Blocks))
	}
	if len(task.BlockedBy) != 1 {
		t.Errorf("BlockedBy len = %d, want 1", len(task.BlockedBy))
	}
}

func TestParseTaskJSON_Invalid(t *testing.T) {
	_, err := ParseTaskJSON([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseTask_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	task, err := ParseTask(filepath.Join(dir, "tasks", "test-session-001", "1.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "1" {
		t.Errorf("ID = %q, want 1", task.ID)
	}
	if task.Status != TaskCompleted {
		t.Errorf("Status = %q, want completed", task.Status)
	}
}

// --- Agent Meta Parsing Tests ---

func TestParseAgentMetaJSON_Valid(t *testing.T) {
	raw := []byte(`{
		"agentType": "Explore",
		"description": "Explore the codebase"
	}`)

	meta, err := ParseAgentMetaJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.AgentType != "Explore" {
		t.Errorf("AgentType = %q, want Explore", meta.AgentType)
	}
	if meta.Description != "Explore the codebase" {
		t.Errorf("Description = %q, want 'Explore the codebase'", meta.Description)
	}
}

func TestParseAgentMetaJSON_Invalid(t *testing.T) {
	_, err := ParseAgentMetaJSON([]byte(`{broken`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseAgentMeta_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	meta, err := ParseAgentMeta(filepath.Join(dir, "projects", "-Users-test-myproject",
		"test-session-001", "subagents", "agent-abc123.meta.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.AgentType != "Explore" {
		t.Errorf("AgentType = %q, want Explore", meta.AgentType)
	}
}

// --- JSONL Parsing Tests ---

func TestParseJSONLBytes_Valid(t *testing.T) {
	data := []byte(`{"parentUuid":null,"agentId":"abc123","type":"user","message":{"role":"user","content":"hello"}}
{"parentUuid":"p1","agentId":"abc123","message":{"type":"message","role":"assistant","content":[{"type":"tool_use","name":"Grep","input":{"pattern":"test"}}],"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":1000,"output_tokens":500}}}
{"parentUuid":"p2","agentId":"abc123","message":{"type":"message","role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/a.ts"}}],"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":2000,"output_tokens":800}}}
`)

	usage, model, calls, err := parseJSONLBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.InputTokens != 3000 {
		t.Errorf("InputTokens = %d, want 3000", usage.InputTokens)
	}
	if usage.OutputTokens != 1300 {
		t.Errorf("OutputTokens = %d, want 1300", usage.OutputTokens)
	}
	if model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want claude-haiku-4-5-20251001", model)
	}
	if len(calls) != 2 {
		t.Errorf("ToolCalls count = %d, want 2", len(calls))
	}
	if len(calls) > 0 && calls[0].Name != "Grep" {
		t.Errorf("First tool call = %q, want Grep", calls[0].Name)
	}
	if len(calls) > 1 && calls[1].Name != "Read" {
		t.Errorf("Second tool call = %q, want Read", calls[1].Name)
	}
}

func TestParseJSONLBytes_MalformedLines(t *testing.T) {
	// Malformed lines should be skipped, not cause errors.
	data := []byte(`{broken line
{"parentUuid":"p1","agentId":"abc123","message":{"type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50}}}
another broken line
`)

	usage, model, _, err := parseJSONLBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", usage.InputTokens)
	}
	if model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want claude-sonnet-4-20250514", model)
	}
}

func TestParseJSONLBytes_Empty(t *testing.T) {
	usage, model, calls, err := parseJSONLBytes([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", usage.InputTokens)
	}
	if model != "" {
		t.Errorf("Model = %q, want empty", model)
	}
	if len(calls) != 0 {
		t.Errorf("ToolCalls count = %d, want 0", len(calls))
	}
}

func TestParseJSONLFull_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject",
		"test-session-001", "subagents", "agent-abc123.jsonl")

	usage, model, calls, err := ParseJSONLFull(jsonlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.InputTokens != 2124 { // 1234 + 890
		t.Errorf("InputTokens = %d, want 2124", usage.InputTokens)
	}
	if usage.OutputTokens != 912 { // 567 + 345
		t.Errorf("OutputTokens = %d, want 912", usage.OutputTokens)
	}
	if model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want claude-haiku-4-5-20251001", model)
	}
	if len(calls) != 3 { // Grep, Read, Glob
		t.Errorf("ToolCalls count = %d, want 3", len(calls))
	}
}

func TestParseJSONLTail_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject",
		"test-session-001", "subagents", "agent-abc123.jsonl")

	// The fixture file is small enough to be fully read with 8KB.
	usage, model, _, err := ParseJSONLTail(jsonlPath, 8192)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.InputTokens != 2124 {
		t.Errorf("InputTokens = %d, want 2124", usage.InputTokens)
	}
	if model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want claude-haiku-4-5-20251001", model)
	}
}

func TestParseJSONLTail_MissingFile(t *testing.T) {
	_, _, _, err := ParseJSONLTail("/nonexistent.jsonl", 8192)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- Projects JSON Parsing Tests ---

func TestParseProjectsJSONBytes_Array(t *testing.T) {
	raw := []byte(`[
		{"name": "myproject", "path": "/Users/test/myproject", "gitRemote": ""},
		{"name": "other", "path": "/Users/test/other", "gitRemote": ""}
	]`)

	m, err := ParseProjectsJSONBytes(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Errorf("map length = %d, want 2", len(m))
	}
	if m["/Users/test/myproject"] != "myproject" {
		t.Errorf("project name = %q, want myproject", m["/Users/test/myproject"])
	}
}

func TestParseProjectsJSONBytes_SimpleMap(t *testing.T) {
	raw := []byte(`{"/Users/test/myproject": "myproject"}`)

	m, err := ParseProjectsJSONBytes(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["/Users/test/myproject"] != "myproject" {
		t.Errorf("project name = %q, want myproject", m["/Users/test/myproject"])
	}
}

func TestParseProjectsJSON_FromFile(t *testing.T) {
	dir := fixtureDir(t)
	m, err := ParseProjectsJSON(filepath.Join(dir, "homunculus", "projects.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["/Users/test/myproject"] != "myproject" {
		t.Errorf("project name = %q, want myproject", m["/Users/test/myproject"])
	}
}

func TestParseProjectsJSON_MissingFile(t *testing.T) {
	_, err := ParseProjectsJSON("/nonexistent/projects.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- Utility Function Tests ---

func TestPathToHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Users/test/project", "-Users-test-project"},
		{"/Users/mengxionghan/.superset/projects/Tmp", "-Users-mengxionghan--superset-projects-Tmp"}, // dot → dash too
		{"/", "-"},
		{"", ""},
	}

	for _, tt := range tests {
		got := PathToHash(tt.input)
		if got != tt.want {
			t.Errorf("PathToHash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m 0s"},
		{time.Hour + 30*time.Minute, "1h 30m"},
		{5*time.Hour + 2*time.Minute, "5h 02m"},
	}

	for _, tt := range tests {
		got := FormatUptime(tt.d)
		if got != tt.want {
			t.Errorf("FormatUptime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{45200, "45.2K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		got := FormatTokens(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// --- Agent Status Tests ---

func TestAgentStatus_String(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   string
	}{
		{StatusRunning, "running"},
		{StatusIdle, "idle"},
		{StatusDone, "done"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("AgentStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestAgentStatus_Symbol(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   string
	}{
		{StatusRunning, "●"},
		{StatusIdle, "◐"},
		{StatusDone, "✓"},
		{StatusUnknown, "○"},
	}

	for _, tt := range tests {
		if got := tt.status.Symbol(); got != tt.want {
			t.Errorf("AgentStatus(%d).Symbol() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// --- Scanner Tests ---

func TestScanner_FullScan(t *testing.T) {
	dir := fixtureDir(t)
	scanner := NewScanner(dir)

	result, err := scanner.FullScan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Sessions) < 2 {
		t.Errorf("Sessions count = %d, want >= 2", len(result.Sessions))
	}

	// Find the session with PID 12345.
	var found bool
	for _, sess := range result.Sessions {
		if sess.PID == 12345 {
			found = true
			if sess.SessionID != "test-session-001" {
				t.Errorf("SessionID = %q, want test-session-001", sess.SessionID)
			}
			if sess.Project != "myproject" {
				t.Errorf("Project = %q, want myproject", sess.Project)
			}
			if sess.PathHash != "-Users-test-myproject" {
				t.Errorf("PathHash = %q, want -Users-test-myproject", sess.PathHash)
			}
			break
		}
	}
	if !found {
		t.Error("session with PID 12345 not found")
	}

	// Check tasks were loaded.
	tasks, hasTasks := result.Tasks["test-session-001"]
	if !hasTasks {
		t.Error("no tasks found for test-session-001")
	} else if len(tasks) < 2 {
		t.Errorf("task count = %d, want >= 2", len(tasks))
	}

	// Check agents were loaded.
	agents, hasAgents := result.Agents["test-session-001"]
	if !hasAgents {
		t.Error("no agents found for test-session-001")
	} else if len(agents) < 1 {
		t.Errorf("agent count = %d, want >= 1", len(agents))
	} else {
		agent := agents[0]
		if agent.AgentType != "Explore" {
			t.Errorf("AgentType = %q, want Explore", agent.AgentType)
		}
		if agent.Model != "claude-haiku-4-5-20251001" {
			t.Errorf("Model = %q, want claude-haiku-4-5-20251001", agent.Model)
		}
	}
}

func TestScanner_FullScan_MissingDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path/to/claude")
	result, err := scanner.FullScan()
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if len(result.Sessions) != 0 {
		t.Errorf("Sessions count = %d, want 0", len(result.Sessions))
	}
}

func TestIsPIDAlive(t *testing.T) {
	// Our own PID should be alive.
	if !IsPIDAlive(os.Getpid()) {
		t.Error("current process PID should be alive")
	}

	// PID 0 should not be alive (invalid).
	if IsPIDAlive(0) {
		t.Error("PID 0 should not be alive")
	}

	// Negative PID should not be alive.
	if IsPIDAlive(-1) {
		t.Error("negative PID should not be alive")
	}
}

func TestDetectAgentStatus_MissingFile(t *testing.T) {
	status := DetectAgentStatus("/nonexistent/agent.jsonl")
	if status != StatusUnknown {
		t.Errorf("status = %v, want StatusUnknown", status)
	}
}

func TestScanner_WatchPaths(t *testing.T) {
	scanner := NewScanner("/test/.claude")
	paths := scanner.WatchPaths()
	if len(paths) != 3 {
		t.Errorf("WatchPaths count = %d, want 3", len(paths))
	}
}

// --- Tool Call Extraction Tests ---

func TestExtractToolCalls(t *testing.T) {
	raw := []byte(`[
		{"type": "text", "text": "some analysis"},
		{"type": "tool_use", "name": "Grep", "input": {"pattern": "test"}},
		{"type": "tool_use", "name": "Read", "input": {"file_path": "/a.ts"}}
	]`)

	calls := extractToolCalls(raw)
	if len(calls) != 2 {
		t.Fatalf("calls count = %d, want 2", len(calls))
	}
	if calls[0].Name != "Grep" {
		t.Errorf("first call = %q, want Grep", calls[0].Name)
	}
	if calls[1].Name != "Read" {
		t.Errorf("second call = %q, want Read", calls[1].Name)
	}
}

func TestExtractToolCalls_StringContent(t *testing.T) {
	// Content that is just a string, not an array.
	raw := []byte(`"just a string response"`)
	calls := extractToolCalls(raw)
	if len(calls) != 0 {
		t.Errorf("calls count = %d, want 0", len(calls))
	}
}

func TestSummarizeInput(t *testing.T) {
	raw := []byte(`{"pattern": "test", "type": "ts"}`)
	summary := summarizeInput(raw)
	if summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestSummarizeInput_Empty(t *testing.T) {
	summary := summarizeInput(nil)
	if summary != "" {
		t.Errorf("summary = %q, want empty", summary)
	}
}
