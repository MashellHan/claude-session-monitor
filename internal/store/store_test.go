package store

import (
	"testing"

	"github.com/MashellHan/claude-session-monitor/internal/data"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.SessionCount() != 0 {
		t.Errorf("SessionCount() = %d, want 0", s.SessionCount())
	}
}

func TestLoadScanResult(t *testing.T) {
	s := New()

	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Kind: "interactive", Alive: true, StartedAt: 2000},
			{PID: 200, SessionID: "s2", Kind: "background", Alive: false, StartedAt: 1000},
		},
		Agents: map[string][]data.Agent{
			"s1": {
				{ID: "a1", AgentType: "Explore", Status: data.StatusRunning,
					Tokens: data.TokenUsage{InputTokens: 1000, OutputTokens: 500}},
				{ID: "a2", AgentType: "Review", Status: data.StatusDone,
					Tokens: data.TokenUsage{InputTokens: 2000, OutputTokens: 800}},
			},
		},
		Tasks: map[string][]data.Task{
			"s1": {
				{ID: "1", Subject: "Task 1", Status: data.TaskCompleted},
				{ID: "2", Subject: "Task 2", Status: data.TaskInProgress},
				{ID: "3", Subject: "Task 3", Status: data.TaskPending},
			},
		},
	}

	s.LoadScanResult(result)

	if s.SessionCount() != 2 {
		t.Errorf("SessionCount() = %d, want 2", s.SessionCount())
	}

	// Alive sessions should come first.
	sessions := s.Sessions()
	if len(sessions) != 2 {
		t.Fatalf("Sessions() length = %d, want 2", len(sessions))
	}
	if sessions[0].PID != 100 {
		t.Errorf("first session PID = %d, want 100 (alive first)", sessions[0].PID)
	}
}

func TestSessionByIndex(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true, StartedAt: 2000},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	sess, ok := s.SessionByIndex(0)
	if !ok {
		t.Fatal("SessionByIndex(0) should return ok=true")
	}
	if sess.PID != 100 {
		t.Errorf("PID = %d, want 100", sess.PID)
	}

	_, ok = s.SessionByIndex(5)
	if ok {
		t.Error("SessionByIndex(5) should return ok=false for out of bounds")
	}

	_, ok = s.SessionByIndex(-1)
	if ok {
		t.Error("SessionByIndex(-1) should return ok=false for negative index")
	}
}

func TestUpdateSession(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true, StartedAt: 2000},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	// Update existing session.
	s.UpdateSession(data.Session{PID: 100, SessionID: "s1", Alive: false, StartedAt: 2000})
	sess, ok := s.SessionByIndex(0)
	if !ok {
		t.Fatal("session should still exist after update")
	}
	if sess.Alive {
		t.Error("session should be dead after update")
	}

	// Add new session.
	s.UpdateSession(data.Session{PID: 200, SessionID: "s2", Alive: true, StartedAt: 3000})
	if s.SessionCount() != 2 {
		t.Errorf("SessionCount() = %d, want 2", s.SessionCount())
	}
}

func TestRemoveSession(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true, StartedAt: 2000},
			{PID: 200, SessionID: "s2", Alive: true, StartedAt: 1000},
		},
		Agents: map[string][]data.Agent{
			"s1": {{ID: "a1"}},
		},
		Tasks: map[string][]data.Task{
			"s1": {{ID: "t1"}},
		},
	}
	s.LoadScanResult(result)

	s.RemoveSession("s1")

	if s.SessionCount() != 1 {
		t.Errorf("SessionCount() = %d, want 1 after removal", s.SessionCount())
	}

	agents := s.AgentsForSession("s1")
	if len(agents) != 0 {
		t.Errorf("agents for removed session = %d, want 0", len(agents))
	}

	tasks := s.TasksForSession("s1")
	if len(tasks) != 0 {
		t.Errorf("tasks for removed session = %d, want 0", len(tasks))
	}
}

func TestAgentsForSession(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true},
		},
		Agents: map[string][]data.Agent{
			"s1": {
				{ID: "a1", AgentType: "Explore"},
				{ID: "a2", AgentType: "Review"},
			},
		},
		Tasks: make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	agents := s.AgentsForSession("s1")
	if len(agents) != 2 {
		t.Errorf("agents count = %d, want 2", len(agents))
	}

	// Verify it's a copy (modifying shouldn't affect store).
	agents[0].AgentType = "Modified"
	original := s.AgentsForSession("s1")
	if original[0].AgentType == "Modified" {
		t.Error("AgentsForSession should return a copy, not a reference")
	}

	// Non-existent session.
	noAgents := s.AgentsForSession("nonexistent")
	if noAgents != nil {
		t.Errorf("agents for nonexistent session should be nil, got %v", noAgents)
	}
}

func TestTasksForSession(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true},
		},
		Agents: make(map[string][]data.Agent),
		Tasks: map[string][]data.Task{
			"s1": {
				{ID: "1", Subject: "Task 1", Status: data.TaskCompleted},
				{ID: "2", Subject: "Task 2", Status: data.TaskPending},
			},
		},
	}
	s.LoadScanResult(result)

	tasks := s.TasksForSession("s1")
	if len(tasks) != 2 {
		t.Errorf("tasks count = %d, want 2", len(tasks))
	}

	// Non-existent session.
	noTasks := s.TasksForSession("nonexistent")
	if noTasks != nil {
		t.Errorf("tasks for nonexistent session should be nil, got %v", noTasks)
	}
}

func TestUpdateAgents(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	newAgents := []data.Agent{
		{ID: "a1", AgentType: "Explore"},
	}
	s.UpdateAgents("s1", newAgents)

	agents := s.AgentsForSession("s1")
	if len(agents) != 1 {
		t.Errorf("agents count = %d, want 1", len(agents))
	}
}

func TestUpdateTasks(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	newTasks := []data.Task{
		{ID: "1", Subject: "New task", Status: data.TaskPending},
	}
	s.UpdateTasks("s1", newTasks)

	tasks := s.TasksForSession("s1")
	if len(tasks) != 1 {
		t.Errorf("tasks count = %d, want 1", len(tasks))
	}
}

func TestStats(t *testing.T) {
	s := New()
	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true, StartedAt: 2000},
			{PID: 200, SessionID: "s2", Alive: false, StartedAt: 1000},
		},
		Agents: map[string][]data.Agent{
			"s1": {
				{ID: "a1", Status: data.StatusRunning, Tokens: data.TokenUsage{InputTokens: 1000, OutputTokens: 500}},
				{ID: "a2", Status: data.StatusDone, Tokens: data.TokenUsage{InputTokens: 2000, OutputTokens: 800}},
			},
			"s2": {
				{ID: "a3", Status: data.StatusIdle, Tokens: data.TokenUsage{InputTokens: 500, OutputTokens: 200}},
			},
		},
		Tasks: map[string][]data.Task{
			"s1": {
				{ID: "1", Status: data.TaskCompleted},
				{ID: "2", Status: data.TaskInProgress},
				{ID: "3", Status: data.TaskPending},
			},
		},
	}
	s.LoadScanResult(result)

	stats := s.Stats()

	if stats.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", stats.TotalSessions)
	}
	if stats.ActiveSessions != 1 {
		t.Errorf("ActiveSessions = %d, want 1", stats.ActiveSessions)
	}
	if stats.TotalAgents != 3 {
		t.Errorf("TotalAgents = %d, want 3", stats.TotalAgents)
	}
	if stats.RunningAgents != 1 {
		t.Errorf("RunningAgents = %d, want 1", stats.RunningAgents)
	}
	if stats.IdleAgents != 1 {
		t.Errorf("IdleAgents = %d, want 1", stats.IdleAgents)
	}
	if stats.DoneAgents != 1 {
		t.Errorf("DoneAgents = %d, want 1", stats.DoneAgents)
	}
	if stats.TotalTasks != 3 {
		t.Errorf("TotalTasks = %d, want 3", stats.TotalTasks)
	}
	if stats.CompletedTasks != 1 {
		t.Errorf("CompletedTasks = %d, want 1", stats.CompletedTasks)
	}
	if stats.InProgressTasks != 1 {
		t.Errorf("InProgressTasks = %d, want 1", stats.InProgressTasks)
	}
	if stats.PendingTasks != 1 {
		t.Errorf("PendingTasks = %d, want 1", stats.PendingTasks)
	}
	if stats.TotalTokensIn != 3500 { // 1000 + 2000 + 500
		t.Errorf("TotalTokensIn = %d, want 3500", stats.TotalTokensIn)
	}
	if stats.TotalTokensOut != 1500 { // 500 + 800 + 200
		t.Errorf("TotalTokensOut = %d, want 1500", stats.TotalTokensOut)
	}
}

func TestStats_Empty(t *testing.T) {
	s := New()
	stats := s.Stats()

	if stats.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", stats.TotalSessions)
	}
	if stats.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", stats.TotalAgents)
	}
}

func TestLoadScanResult_ClearsOldData(t *testing.T) {
	s := New()

	// Load first scan.
	s.LoadScanResult(&data.ScanResult{
		Sessions: []data.Session{
			{PID: 100, SessionID: "s1", Alive: true},
		},
		Agents: map[string][]data.Agent{
			"s1": {{ID: "a1"}},
		},
		Tasks: map[string][]data.Task{
			"s1": {{ID: "t1"}},
		},
	})

	// Load second scan with different data.
	s.LoadScanResult(&data.ScanResult{
		Sessions: []data.Session{
			{PID: 200, SessionID: "s2", Alive: true},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	})

	if s.SessionCount() != 1 {
		t.Errorf("SessionCount() = %d, want 1", s.SessionCount())
	}

	// Old session's data should be gone.
	agents := s.AgentsForSession("s1")
	if len(agents) != 0 {
		t.Error("old session agents should be cleared")
	}
}

func TestSortingOrder(t *testing.T) {
	s := New()

	result := &data.ScanResult{
		Sessions: []data.Session{
			{PID: 1, SessionID: "old-dead", Alive: false, StartedAt: 1000},
			{PID: 2, SessionID: "new-alive", Alive: true, StartedAt: 3000},
			{PID: 3, SessionID: "old-alive", Alive: true, StartedAt: 2000},
			{PID: 4, SessionID: "new-dead", Alive: false, StartedAt: 4000},
		},
		Agents: make(map[string][]data.Agent),
		Tasks:  make(map[string][]data.Task),
	}
	s.LoadScanResult(result)

	sessions := s.Sessions()
	if len(sessions) != 4 {
		t.Fatalf("Sessions count = %d, want 4", len(sessions))
	}

	// Alive sessions first (newest first), then dead (newest first).
	if sessions[0].SessionID != "new-alive" {
		t.Errorf("first session = %q, want new-alive", sessions[0].SessionID)
	}
	if sessions[1].SessionID != "old-alive" {
		t.Errorf("second session = %q, want old-alive", sessions[1].SessionID)
	}
	if sessions[2].SessionID != "new-dead" {
		t.Errorf("third session = %q, want new-dead", sessions[2].SessionID)
	}
	if sessions[3].SessionID != "old-dead" {
		t.Errorf("fourth session = %q, want old-dead", sessions[3].SessionID)
	}
}
