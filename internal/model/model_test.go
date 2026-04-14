package model

import (
	"strings"
	"testing"

	"github.com/MashellHan/claude-session-monitor/internal/data"
)

// --- SessionList Tests ---

func TestSessionList_SetSessions(t *testing.T) {
	sl := NewSessionList()
	sessions := []data.Session{
		{PID: 100, SessionID: "s1", Alive: true},
		{PID: 200, SessionID: "s2", Alive: false},
	}
	sl.SetSessions(sessions)

	sess, ok := sl.SelectedSession()
	if !ok {
		t.Fatal("should have selected session")
	}
	if sess.PID != 100 {
		t.Errorf("PID = %d, want 100", sess.PID)
	}
}

func TestSessionList_EmptySelection(t *testing.T) {
	sl := NewSessionList()
	_, ok := sl.SelectedSession()
	if ok {
		t.Error("empty list should have no selection")
	}
}

func TestSessionList_CursorNavigation(t *testing.T) {
	sl := NewSessionList()
	sl.SetSize(80, 20)
	sessions := []data.Session{
		{PID: 100, SessionID: "s1"},
		{PID: 200, SessionID: "s2"},
		{PID: 300, SessionID: "s3"},
	}
	sl.SetSessions(sessions)

	// Start at 0.
	sess, _ := sl.SelectedSession()
	if sess.PID != 100 {
		t.Errorf("initial PID = %d, want 100", sess.PID)
	}

	// Move down.
	sl.CursorDown()
	sess, _ = sl.SelectedSession()
	if sess.PID != 200 {
		t.Errorf("after down PID = %d, want 200", sess.PID)
	}

	// Move down again.
	sl.CursorDown()
	sess, _ = sl.SelectedSession()
	if sess.PID != 300 {
		t.Errorf("after 2x down PID = %d, want 300", sess.PID)
	}

	// Can't move past end.
	sl.CursorDown()
	sess, _ = sl.SelectedSession()
	if sess.PID != 300 {
		t.Errorf("past end PID = %d, want 300", sess.PID)
	}

	// Move up.
	sl.CursorUp()
	sess, _ = sl.SelectedSession()
	if sess.PID != 200 {
		t.Errorf("after up PID = %d, want 200", sess.PID)
	}

	// Can't move past start.
	sl.CursorUp()
	sl.CursorUp()
	sess, _ = sl.SelectedSession()
	if sess.PID != 100 {
		t.Errorf("past start PID = %d, want 100", sess.PID)
	}
}

func TestSessionList_CursorClamp(t *testing.T) {
	sl := NewSessionList()
	sl.SetSize(80, 20)

	// Set sessions with cursor at 2.
	sessions := []data.Session{
		{PID: 100, SessionID: "s1"},
		{PID: 200, SessionID: "s2"},
		{PID: 300, SessionID: "s3"},
	}
	sl.SetSessions(sessions)
	sl.CursorDown()
	sl.CursorDown() // cursor = 2

	// Now set fewer sessions — cursor should clamp.
	sl.SetSessions([]data.Session{
		{PID: 100, SessionID: "s1"},
	})
	sess, ok := sl.SelectedSession()
	if !ok {
		t.Fatal("should have selection")
	}
	if sess.PID != 100 {
		t.Errorf("clamped PID = %d, want 100", sess.PID)
	}
}

func TestSessionList_View_Empty(t *testing.T) {
	sl := NewSessionList()
	sl.SetSize(80, 20)
	view := sl.View(true)
	if !strings.Contains(view, "No sessions found") {
		t.Errorf("empty view should show 'No sessions found', got: %s", view)
	}
}

func TestSessionList_View_WithData(t *testing.T) {
	sl := NewSessionList()
	sl.SetSize(100, 20)
	sessions := []data.Session{
		{PID: 100, SessionID: "s1", Project: "myproject", Kind: "interactive", Uptime: "2h 15m", Alive: true},
	}
	sl.SetSessions(sessions)
	view := sl.View(true)
	if !strings.Contains(view, "PID") {
		t.Error("view should contain header with PID")
	}
}

// --- AgentTable Tests ---

func TestAgentTable_SetAgents(t *testing.T) {
	at := NewAgentTable()
	agents := []data.Agent{
		{ID: "a1", AgentType: "Explore", Status: data.StatusRunning},
		{ID: "a2", AgentType: "Review", Status: data.StatusDone},
	}
	at.SetAgents(agents)

	agent, ok := at.SelectedAgent()
	if !ok {
		t.Fatal("should have selected agent")
	}
	if agent.ID != "a1" {
		t.Errorf("ID = %q, want a1", agent.ID)
	}
}

func TestAgentTable_EmptySelection(t *testing.T) {
	at := NewAgentTable()
	_, ok := at.SelectedAgent()
	if ok {
		t.Error("empty table should have no selection")
	}
}

func TestAgentTable_CursorNavigation(t *testing.T) {
	at := NewAgentTable()
	at.SetSize(80, 20)
	agents := []data.Agent{
		{ID: "a1"}, {ID: "a2"}, {ID: "a3"},
	}
	at.SetAgents(agents)

	at.CursorDown()
	agent, _ := at.SelectedAgent()
	if agent.ID != "a2" {
		t.Errorf("after down ID = %q, want a2", agent.ID)
	}

	at.CursorUp()
	agent, _ = at.SelectedAgent()
	if agent.ID != "a1" {
		t.Errorf("after up ID = %q, want a1", agent.ID)
	}
}

func TestAgentTable_ToggleDetail(t *testing.T) {
	at := NewAgentTable()
	at.SetSize(80, 20)
	agents := []data.Agent{
		{ID: "a1", AgentType: "Explore", Model: "claude-haiku", Tokens: data.TokenUsage{InputTokens: 1000, OutputTokens: 500}},
	}
	at.SetAgents(agents)

	if at.expandedIdx != -1 {
		t.Error("initially should have no expansion")
	}

	at.ToggleDetail()
	if at.expandedIdx != 0 {
		t.Errorf("after toggle expandedIdx = %d, want 0", at.expandedIdx)
	}

	at.ToggleDetail()
	if at.expandedIdx != -1 {
		t.Errorf("after second toggle expandedIdx = %d, want -1", at.expandedIdx)
	}
}

func TestAgentTable_View_Empty(t *testing.T) {
	at := NewAgentTable()
	at.SetSize(80, 20)
	view := at.View(true)
	if !strings.Contains(view, "No agents found") {
		t.Errorf("empty view should show 'No agents found', got: %s", view)
	}
}

func TestAgentTable_View_WithData(t *testing.T) {
	at := NewAgentTable()
	at.SetSize(100, 20)
	agents := []data.Agent{
		{ID: "a1", AgentType: "Explore", Status: data.StatusRunning, Elapsed: "45s", Description: "Test agent"},
	}
	at.SetAgents(agents)
	view := at.View(true)
	if !strings.Contains(view, "Agent Type") {
		t.Error("view should contain header")
	}
	if !strings.Contains(view, "Explore") {
		t.Error("view should contain agent type")
	}
}

func TestAgentTable_View_WithExpansion(t *testing.T) {
	at := NewAgentTable()
	at.SetSize(100, 20)
	agents := []data.Agent{
		{
			ID: "a1", AgentType: "Explore", Status: data.StatusRunning,
			Model: "claude-haiku", Elapsed: "45s",
			Tokens: data.TokenUsage{InputTokens: 1234, OutputTokens: 567},
			ToolCalls: []data.ToolCall{
				{Name: "Grep", Input: "pattern=test"},
			},
		},
	}
	at.SetAgents(agents)
	at.ToggleDetail()

	view := at.View(true)
	if !strings.Contains(view, "Agent ID:") || !strings.Contains(view, "a1") {
		t.Error("expanded view should contain agent ID")
	}
	if !strings.Contains(view, "claude-haiku") {
		t.Error("expanded view should contain model")
	}
}

// --- AgentSummary Tests ---

func TestAgentSummary_Empty(t *testing.T) {
	result := AgentSummary(nil)
	if !strings.Contains(result, "0 agents") {
		t.Errorf("empty summary = %q, should contain '0 agents'", result)
	}
}

func TestAgentSummary_Running(t *testing.T) {
	agents := []data.Agent{
		{Status: data.StatusRunning},
		{Status: data.StatusDone},
	}
	result := AgentSummary(agents)
	if !strings.Contains(result, "running") {
		t.Errorf("summary = %q, should contain 'running'", result)
	}
}

func TestAgentSummary_AllDone(t *testing.T) {
	agents := []data.Agent{
		{Status: data.StatusDone},
		{Status: data.StatusDone},
	}
	result := AgentSummary(agents)
	if !strings.Contains(result, "done") {
		t.Errorf("summary = %q, should contain 'done'", result)
	}
}

// --- AgentCountSummary Tests ---

func TestAgentCountSummary_Empty(t *testing.T) {
	result := AgentCountSummary(nil)
	if !strings.Contains(result, "0") {
		t.Errorf("empty summary = %q, should contain '0'", result)
	}
}

func TestAgentCountSummary_Running(t *testing.T) {
	agents := []data.Agent{
		{Status: data.StatusRunning},
		{Status: data.StatusDone},
		{Status: data.StatusDone},
	}
	result := AgentCountSummary(agents)
	if !strings.Contains(result, "3") {
		t.Errorf("summary = %q, should contain '3'", result)
	}
	if !strings.Contains(result, "●1") {
		t.Errorf("summary = %q, should contain '●1'", result)
	}
}

func TestAgentCountSummary_NoneRunning(t *testing.T) {
	agents := []data.Agent{
		{Status: data.StatusDone},
		{Status: data.StatusDone},
	}
	result := AgentCountSummary(agents)
	if !strings.Contains(result, "2") {
		t.Errorf("summary = %q, should contain '2'", result)
	}
}

// --- formatToolCallSummary Tests ---

func TestFormatToolCallSummary(t *testing.T) {
	toolMap := map[string]int{
		"Grep": 4,
		"Read": 3,
		"Glob": 3,
		"Bash": 2,
	}
	result := formatToolCallSummary(toolMap, 12)
	if !strings.Contains(result, "12 total") {
		t.Errorf("result = %q, should contain '12 total'", result)
	}
	if !strings.Contains(result, "Grep: 4") {
		t.Errorf("result = %q, should contain 'Grep: 4'", result)
	}
}

// --- formatTokenComma Tests ---

func TestFormatTokenComma(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{6234, "6,234"},
		{1000000, "1,000,000"},
	}

	for _, tt := range tests {
		got := formatTokenComma(tt.n)
		if got != tt.want {
			t.Errorf("formatTokenComma(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// --- TaskList Tests ---

func TestTaskList_SetTasks(t *testing.T) {
	tl := NewTaskList()
	tasks := []data.Task{
		{ID: "1", Subject: "Task 1", Status: data.TaskCompleted},
		{ID: "2", Subject: "Task 2", Status: data.TaskPending},
	}
	tl.SetTasks(tasks)
}

func TestTaskList_View_Empty(t *testing.T) {
	tl := NewTaskList()
	tl.SetSize(80, 20)
	view := tl.View(true)
	if !strings.Contains(view, "No tasks found") {
		t.Errorf("empty view should show 'No tasks found', got: %s", view)
	}
}

func TestTaskList_View_WithData(t *testing.T) {
	tl := NewTaskList()
	tl.SetSize(100, 20)
	tasks := []data.Task{
		{ID: "1", Subject: "Research data sources", Status: data.TaskCompleted, BlockedBy: []string{}},
		{ID: "2", Subject: "Architecture design", Status: data.TaskInProgress, BlockedBy: []string{"1"}},
		{ID: "3", Subject: "Implementation", Status: data.TaskPending, BlockedBy: []string{"2"}},
	}
	tl.SetTasks(tasks)
	view := tl.View(true)

	if !strings.Contains(view, "ID") {
		t.Error("view should contain header")
	}
	if !strings.Contains(view, "Research") {
		t.Error("view should contain task subject")
	}
}

func TestTaskList_CursorNavigation(t *testing.T) {
	tl := NewTaskList()
	tl.SetSize(80, 20)
	tasks := []data.Task{
		{ID: "1"}, {ID: "2"}, {ID: "3"},
	}
	tl.SetTasks(tasks)

	tl.CursorDown()
	tl.CursorDown()
	tl.CursorUp()
}

// --- TaskSummary Tests ---

func TestTaskSummary_Empty(t *testing.T) {
	result := TaskSummary(nil)
	if !strings.Contains(result, "0 total") {
		t.Errorf("empty summary = %q, should contain '0 total'", result)
	}
}

func TestTaskSummary_WithTasks(t *testing.T) {
	tasks := []data.Task{
		{Status: data.TaskCompleted},
		{Status: data.TaskInProgress},
		{Status: data.TaskPending},
	}
	result := TaskSummary(tasks)
	if !strings.Contains(result, "3 total") {
		t.Errorf("summary = %q, should contain '3 total'", result)
	}
}

// --- SummaryBar Tests ---

func TestSummaryBar_View(t *testing.T) {
	sb := NewSummaryBar()
	sb.SetWidth(120)
	sb.SetStats(data.Stats{
		TotalSessions:  5,
		ActiveSessions: 3,
		TotalAgents:    10,
		RunningAgents:  4,
		IdleAgents:     2,
		DoneAgents:     4,
		TotalTasks:     8,
		TotalTokensIn:  45000,
		TotalTokensOut: 12000,
		TotalTokensAll: 57000,
	})

	view := sb.View()
	if !strings.Contains(view, "3 sessions") {
		t.Errorf("summary should contain '3 sessions', got: %s", view)
	}
	if !strings.Contains(view, "10 agents") {
		t.Errorf("summary should contain '10 agents', got: %s", view)
	}
}

func TestSummaryBar_View_Empty(t *testing.T) {
	sb := NewSummaryBar()
	sb.SetWidth(120)
	sb.SetStats(data.Stats{})

	view := sb.View()
	if !strings.Contains(view, "0 sessions") {
		t.Errorf("empty summary should contain '0 sessions', got: %s", view)
	}
}

// --- DetailView Tests ---

func TestDetailView_Visibility(t *testing.T) {
	dv := NewDetailView()
	if dv.IsVisible() {
		t.Error("should not be visible initially")
	}

	agent := &data.Agent{ID: "a1", Model: "claude-haiku"}
	dv.SetAgent(agent)
	if !dv.IsVisible() {
		t.Error("should be visible after SetAgent")
	}

	dv.Toggle()
	if dv.IsVisible() {
		t.Error("should not be visible after toggle")
	}

	dv.Toggle()
	if !dv.IsVisible() {
		t.Error("should be visible after second toggle")
	}
}

func TestDetailView_View_Empty(t *testing.T) {
	dv := NewDetailView()
	dv.SetSize(80)
	view := dv.View()
	if view != "" {
		t.Errorf("hidden detail should be empty, got: %s", view)
	}
}

func TestDetailView_View_WithAgent(t *testing.T) {
	dv := NewDetailView()
	dv.SetSize(80)

	agent := &data.Agent{
		ID:    "a1",
		Model: "claude-haiku-4-5",
		Tokens: data.TokenUsage{
			InputTokens:  1234,
			OutputTokens: 567,
		},
		ToolCalls: []data.ToolCall{
			{Name: "Grep", Input: "pattern=test"},
		},
	}
	dv.SetAgent(agent)

	view := dv.View()
	if !strings.Contains(view, "a1") {
		t.Error("detail should contain agent ID")
	}
	if !strings.Contains(view, "claude-haiku") {
		t.Error("detail should contain model")
	}
}

// --- Helper Function Tests ---

func TestFormatAgentStatus(t *testing.T) {
	tests := []struct {
		status data.AgentStatus
		want   string
	}{
		{data.StatusRunning, "running"},
		{data.StatusIdle, "idle"},
		{data.StatusDone, "done"},
		{data.StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		result := formatAgentStatus(tt.status)
		if !strings.Contains(result, tt.want) {
			t.Errorf("formatAgentStatus(%v) = %q, should contain %q", tt.status, result, tt.want)
		}
	}
}

func TestFormatTaskStatus(t *testing.T) {
	tests := []struct {
		status data.TaskStatus
		want   string
	}{
		{data.TaskCompleted, "done"},
		{data.TaskInProgress, "work"},
		{data.TaskPending, "pend"},
	}

	for _, tt := range tests {
		result := formatTaskStatus(tt.status)
		if !strings.Contains(result, tt.want) {
			t.Errorf("formatTaskStatus(%v) = %q, should contain %q", tt.status, result, tt.want)
		}
	}
}

func TestFormatDeps(t *testing.T) {
	tests := []struct {
		deps []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"1"}, "←1"},
		{[]string{"1", "2"}, "←1,←2"},
	}

	for _, tt := range tests {
		result := formatDeps(tt.deps)
		if result != tt.want {
			t.Errorf("formatDeps(%v) = %q, want %q", tt.deps, result, tt.want)
		}
	}
}
