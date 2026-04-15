package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		{45200, "45K"},
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
		if agent.Model != "haiku" {
			t.Errorf("Model = %q, want haiku", agent.Model)
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

// --- v1.1 New Parser Tests ---

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{9999, "10.0K"},
		{45200, "45K"},
		{999999, "1000K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
		{29400000, "29M"},
		{999999999, "1000M"},
		{1000000000, "1.0B"},
		{1200000000, "1.2B"},
	}

	for _, tt := range tests {
		got := FormatTokenCount(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestTokenUsage_Total(t *testing.T) {
	usage := TokenUsage{
		InputTokens:         1000,
		OutputTokens:        500,
		CacheCreationTokens: 200,
		CacheReadTokens:     300,
	}
	if usage.Total() != 2000 {
		t.Errorf("Total() = %d, want 2000", usage.Total())
	}
}

func TestTokenUsage_Formatted(t *testing.T) {
	usage := TokenUsage{
		InputTokens:  200000,
		OutputTokens: 45000,
	}
	got := usage.Formatted()
	if got != "245K" {
		t.Errorf("Formatted() = %q, want 245K", got)
	}
}

func TestParseSessionTopic_StringContent(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl")

	topic, topicFull := ParseSessionTopic(jsonlPath)
	if topic == "" {
		t.Fatal("topic should not be empty")
	}
	if topicFull == "" {
		t.Fatal("topicFull should not be empty")
	}
	if !strings.Contains(topicFull, "Build a REST API server") {
		t.Errorf("topicFull = %q, should contain 'Build a REST API server'", topicFull)
	}
	// Topic should be truncated to <= 60 chars.
	if len([]rune(topic)) > 60 {
		t.Errorf("topic length = %d, should be <= 60", len([]rune(topic)))
	}
}

func TestParseSessionTopic_ArrayContent(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-otherproject", "test-session-002.jsonl")

	topic, topicFull := ParseSessionTopic(jsonlPath)
	if topic == "" {
		t.Fatal("topic should not be empty")
	}
	if !strings.Contains(topicFull, "Analyze the project structure") {
		t.Errorf("topicFull = %q, should contain 'Analyze the project structure'", topicFull)
	}
	if !strings.Contains(topicFull, "suggest improvements") {
		t.Errorf("topicFull = %q, should contain 'suggest improvements'", topicFull)
	}
}

func TestParseSessionTopic_MissingFile(t *testing.T) {
	topic, topicFull := ParseSessionTopic("/nonexistent.jsonl")
	if topic != "" || topicFull != "" {
		t.Errorf("missing file should return empty, got topic=%q, topicFull=%q", topic, topicFull)
	}
}

func TestParseGitBranch(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl")

	branch := ParseGitBranch(jsonlPath)
	if branch != "feature/api-server" {
		t.Errorf("GitBranch = %q, want 'feature/api-server'", branch)
	}
}

func TestParseGitBranch_Main(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-otherproject", "test-session-002.jsonl")

	branch := ParseGitBranch(jsonlPath)
	if branch != "main" {
		t.Errorf("GitBranch = %q, want 'main'", branch)
	}
}

func TestParseGitBranch_MissingFile(t *testing.T) {
	branch := ParseGitBranch("/nonexistent.jsonl")
	if branch != "" {
		t.Errorf("missing file should return empty, got %q", branch)
	}
}

func TestParseMessageCount(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl")

	count := ParseMessageCount(jsonlPath)
	// The fixture has 3 user messages.
	if count != 3 {
		t.Errorf("MessageCount = %d, want 3", count)
	}
}

func TestParseMessageCount_SingleUser(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-otherproject", "test-session-002.jsonl")

	count := ParseMessageCount(jsonlPath)
	if count != 1 {
		t.Errorf("MessageCount = %d, want 1", count)
	}
}

func TestParseMessageCount_MissingFile(t *testing.T) {
	count := ParseMessageCount("/nonexistent.jsonl")
	if count != 0 {
		t.Errorf("missing file should return 0, got %d", count)
	}
}

func TestParseSessionTokens(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl")

	usage := ParseSessionTokens(jsonlPath)
	// 500 + 800 + 300 = 1600 input tokens
	if usage.InputTokens != 1600 {
		t.Errorf("InputTokens = %d, want 1600", usage.InputTokens)
	}
	// 200 + 400 + 150 = 750 output tokens
	if usage.OutputTokens != 750 {
		t.Errorf("OutputTokens = %d, want 750", usage.OutputTokens)
	}
	// 100 + 0 + 0 = 100 cache creation
	if usage.CacheCreationTokens != 100 {
		t.Errorf("CacheCreationTokens = %d, want 100", usage.CacheCreationTokens)
	}
	// 50 + 200 + 100 = 350 cache read
	if usage.CacheReadTokens != 350 {
		t.Errorf("CacheReadTokens = %d, want 350", usage.CacheReadTokens)
	}
}

func TestParseSessionTokens_MissingFile(t *testing.T) {
	usage := ParseSessionTokens("/nonexistent.jsonl")
	if usage.Total() != 0 {
		t.Errorf("missing file should return zero tokens, got %d", usage.Total())
	}
}

func TestParseAgentModel(t *testing.T) {
	dir := fixtureDir(t)

	tests := []struct {
		name      string
		path      string
		wantShort string
		wantFull  string
	}{
		{
			name:      "haiku agent",
			path:      filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001", "subagents", "agent-abc123.jsonl"),
			wantShort: "haiku",
			wantFull:  "claude-haiku-4-5-20251001",
		},
		{
			name:      "sonnet session",
			path:      filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl"),
			wantShort: "sonnet",
			wantFull:  "claude-sonnet-4-20250514",
		},
		{
			name:      "opus session",
			path:      filepath.Join(dir, "projects", "-Users-test-otherproject", "test-session-002.jsonl"),
			wantShort: "opus",
			wantFull:  "claude-opus-4-20250514",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			short, full := ParseAgentModel(tt.path)
			if short != tt.wantShort {
				t.Errorf("short = %q, want %q", short, tt.wantShort)
			}
			if full != tt.wantFull {
				t.Errorf("full = %q, want %q", full, tt.wantFull)
			}
		})
	}
}

func TestParseAgentModel_MissingFile(t *testing.T) {
	short, full := ParseAgentModel("/nonexistent.jsonl")
	if short != "" || full != "" {
		t.Errorf("missing file should return empty, got short=%q, full=%q", short, full)
	}
}

func TestParseAgentToolCallSummary(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001", "subagents", "agent-abc123.jsonl")

	toolMap, total := ParseAgentToolCallSummary(jsonlPath)
	if toolMap == nil {
		t.Fatal("toolMap should not be nil")
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if toolMap["Grep"] != 1 {
		t.Errorf("Grep count = %d, want 1", toolMap["Grep"])
	}
	if toolMap["Read"] != 1 {
		t.Errorf("Read count = %d, want 1", toolMap["Read"])
	}
	if toolMap["Glob"] != 1 {
		t.Errorf("Glob count = %d, want 1", toolMap["Glob"])
	}
}

func TestParseAgentToolCallSummary_MissingFile(t *testing.T) {
	toolMap, total := ParseAgentToolCallSummary("/nonexistent.jsonl")
	if toolMap != nil {
		t.Errorf("missing file should return nil toolMap, got %v", toolMap)
	}
	if total != 0 {
		t.Errorf("missing file should return 0 total, got %d", total)
	}
}

func TestParseAgentFinalOutput(t *testing.T) {
	dir := fixtureDir(t)

	// Session JSONL with assistant text.
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject", "test-session-001.jsonl")
	output := ParseAgentFinalOutput(jsonlPath)
	if output == "" {
		t.Error("final output should not be empty")
	}
	if !strings.Contains(output, "rate limiter") {
		t.Errorf("final output = %q, should contain 'rate limiter'", output)
	}
}

func TestParseAgentFinalOutput_MissingFile(t *testing.T) {
	output := ParseAgentFinalOutput("/nonexistent.jsonl")
	if output != "" {
		t.Errorf("missing file should return empty, got %q", output)
	}
}

func TestModelShortName(t *testing.T) {
	tests := []struct {
		full string
		want string
	}{
		{"claude-sonnet-4-20250514", "sonnet"},
		{"claude-haiku-4-5-20251001", "haiku"},
		{"claude-opus-4-20250514", "opus"},
		{"unknown-model", "unknown-model"},
	}

	for _, tt := range tests {
		got := modelShortName(tt.full)
		if got != tt.want {
			t.Errorf("modelShortName(%q) = %q, want %q", tt.full, got, tt.want)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"ab", 2, "ab"},
		{"abcde", 5, "abcde"},
		{"abcdef", 5, "ab..."},
	}

	for _, tt := range tests {
		got := truncateString(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestExtractTextContent(t *testing.T) {
	// String content.
	stringContent := []byte(`"Hello world"`)
	got := extractTextContent(stringContent)
	if got != "Hello world" {
		t.Errorf("string content = %q, want 'Hello world'", got)
	}

	// Array content.
	arrayContent := []byte(`[{"type":"text","text":"Hello "},{"type":"text","text":"world"}]`)
	got = extractTextContent(arrayContent)
	if got != "Hello world" {
		t.Errorf("array content = %q, want 'Hello world'", got)
	}

	// Empty.
	got = extractTextContent(nil)
	if got != "" {
		t.Errorf("nil content = %q, want empty", got)
	}
}

// --- v1.2 New Tests ---

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s ago"},
		{45 * time.Second, "45s ago"},
		{90 * time.Second, "1m ago"},
		{5 * time.Minute, "5m ago"},
		{time.Hour + 30*time.Minute, "1h ago"},
		{5 * time.Hour, "5h ago"},
		{25 * time.Hour, "1d ago"},
		{72 * time.Hour, "3d ago"},
	}

	for _, tt := range tests {
		got := FormatRelativeTime(tt.d)
		if got != tt.want {
			t.Errorf("FormatRelativeTime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestIsGenericProjectName(t *testing.T) {
	generics := []string{"Tmp", "tmp", "temp", "Temp", "src", "projects", "Projects",
		"code", "Code", "workspace", "Workspace", "dev", "Dev", "Desktop", "Documents"}
	for _, name := range generics {
		if !isGenericProjectName(name) {
			t.Errorf("isGenericProjectName(%q) = false, want true", name)
		}
	}

	specifics := []string{"myproject", "claude-monitor", "api-server", "frontend"}
	for _, name := range specifics {
		if isGenericProjectName(name) {
			t.Errorf("isGenericProjectName(%q) = true, want false", name)
		}
	}
}

func TestTruncateProjectFromTopic(t *testing.T) {
	tests := []struct {
		topic string
		want  string
	}{
		// URL extraction
		{"https://github.com/org/my-repo something", "my-repo"},
		{"https://github.com/org/repo", "repo"},
		// Regular text — first 3 words
		{"Build a REST API server", "Build a REST"},
		// CJK text — truncated to 14 runes
		{"分析当前项目下面的所有文件，给出建议", "分析当前项目下面的所有文件，"},
		// Short topic
		{"hi", "hi"},
		// Empty
		{"", ""},
	}

	for _, tt := range tests {
		got := truncateProjectFromTopic(tt.topic)
		if got != tt.want {
			t.Errorf("truncateProjectFromTopic(%q) = %q, want %q", tt.topic, got, tt.want)
		}
	}
}

func TestSanitizeOneLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello\nworld", "hello world"},
		{"hello\r\nworld", "hello world"},
		{"hello\t\tworld", "hello world"},
		{"  hello   world  ", "hello world"},
		{"normal text", "normal text"},
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizeOneLine(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeOneLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseAgentJSONLFull(t *testing.T) {
	dir := fixtureDir(t)
	jsonlPath := filepath.Join(dir, "projects", "-Users-test-myproject",
		"test-session-001", "subagents", "agent-abc123.jsonl")

	info := ParseAgentJSONLFull(jsonlPath)

	if info.Tokens.InputTokens != 2124 {
		t.Errorf("InputTokens = %d, want 2124", info.Tokens.InputTokens)
	}
	if info.Model != "haiku" {
		t.Errorf("Model = %q, want haiku", info.Model)
	}
	if info.ModelFull != "claude-haiku-4-5-20251001" {
		t.Errorf("ModelFull = %q, want claude-haiku-4-5-20251001", info.ModelFull)
	}
	if info.ToolCallTotal != 3 {
		t.Errorf("ToolCallTotal = %d, want 3", info.ToolCallTotal)
	}
	if info.ToolCallMap["Grep"] != 1 {
		t.Errorf("Grep count = %d, want 1", info.ToolCallMap["Grep"])
	}
}

func TestParseAgentJSONLFull_MissingFile(t *testing.T) {
	info := ParseAgentJSONLFull("/nonexistent.jsonl")
	if info.Tokens.Total() != 0 {
		t.Errorf("missing file should have zero tokens, got %d", info.Tokens.Total())
	}
	if info.Model != "" {
		t.Errorf("missing file should have empty model, got %q", info.Model)
	}
}

// --- v1.3 New Tests ---

func TestExtractAssistantText_DelegatesToExtractTextContent(t *testing.T) {
	// extractAssistantText should produce identical results to extractTextContent.
	// Test with string content.
	raw := json.RawMessage(`"hello world"`)
	a := extractAssistantText(raw)
	b := extractTextContent(raw)
	if a != b {
		t.Errorf("extractAssistantText(%q) = %q, extractTextContent = %q, should be identical", string(raw), a, b)
	}

	// Test with array content.
	raw = json.RawMessage(`[{"type":"text","text":"block1"},{"type":"text","text":"block2"}]`)
	a = extractAssistantText(raw)
	b = extractTextContent(raw)
	if a != b {
		t.Errorf("extractAssistantText(%q) = %q, extractTextContent = %q, should be identical", string(raw), a, b)
	}
}

func TestTruncateProjectFromTopic_EdgeCases(t *testing.T) {
	tests := []struct {
		topic string
		want  string
	}{
		// URL with no path segments.
		{"https://example.com", "example.com"},
		// URL with trailing slash.
		{"https://github.com/org/repo/", "repo"},
		// Single word.
		{"deploy", "deploy"},
		// Four words — truncated to 3.
		{"Build a REST API", "Build a REST"},
		// Exactly 14 runes.
		{"12345678901234", "12345678901234"},
		// 15 runes — truncated.
		{"123456789012345", "12345678901234"},
	}

	for _, tt := range tests {
		got := truncateProjectFromTopic(tt.topic)
		if got != tt.want {
			t.Errorf("truncateProjectFromTopic(%q) = %q, want %q", tt.topic, got, tt.want)
		}
	}
}

func TestFormatTokenCount_Boundaries(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{9999, "10.0K"},
		{10000, "10K"},
		{999999, "1000K"},
		{1000000, "1.0M"},
		{9999999, "10.0M"},
		{999999999, "1000M"},
		{1000000000, "1.0B"},
		{2500000000, "2.5B"},
	}

	for _, tt := range tests {
		got := FormatTokenCount(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestSanitizeOneLine_Complex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Multiple newlines and tabs.
		{"line1\n\n\nline2\t\tline3", "line1 line2 line3"},
		// Carriage return + newline.
		{"hello\r\nworld\r\n", "hello world"},
		// Only whitespace.
		{"   \t\n\r   ", ""},
	}

	for _, tt := range tests {
		got := sanitizeOneLine(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeOneLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestModelShortName_Variants(t *testing.T) {
	tests := []struct {
		full string
		want string
	}{
		{"claude-sonnet-4-20250514", "sonnet"},
		{"claude-haiku-4-5-20251001", "haiku"},
		{"claude-opus-4-20250514", "opus"},
		{"gpt-4", "gpt-4"},                    // unknown model passes through
		{"custom-sonnet-model", "sonnet"},      // contains "sonnet"
		{"claude-haiku-3.5", "haiku"},          // older haiku
	}

	for _, tt := range tests {
		got := modelShortName(tt.full)
		if got != tt.want {
			t.Errorf("modelShortName(%q) = %q, want %q", tt.full, got, tt.want)
		}
	}
}
