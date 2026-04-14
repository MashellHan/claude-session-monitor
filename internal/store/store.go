// Package store provides a thread-safe in-memory state store for
// Claude session monitoring data.
package store

import (
	"sort"
	"sync"

	"github.com/MashellHan/claude-session-monitor/internal/data"
)

// Store holds the in-memory state of all sessions, agents, and tasks.
// All methods are thread-safe.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]data.Session  // sessionID → Session
	agents   map[string][]data.Agent  // sessionID → agents
	tasks    map[string][]data.Task   // sessionID → tasks

	// Ordered session IDs for consistent display.
	sessionOrder []string
}

// New creates an empty Store.
func New() *Store {
	return &Store{
		sessions: make(map[string]data.Session),
		agents:   make(map[string][]data.Agent),
		tasks:    make(map[string][]data.Task),
	}
}

// LoadScanResult populates the store from a full scan result.
func (s *Store) LoadScanResult(result *data.ScanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data.
	s.sessions = make(map[string]data.Session)
	s.agents = make(map[string][]data.Agent)
	s.tasks = make(map[string][]data.Task)
	s.sessionOrder = nil

	for _, sess := range result.Sessions {
		s.sessions[sess.SessionID] = sess
		s.sessionOrder = append(s.sessionOrder, sess.SessionID)
	}

	for sid, agents := range result.Agents {
		s.agents[sid] = agents
	}

	for sid, tasks := range result.Tasks {
		s.tasks[sid] = tasks
	}

	// Sort sessions: alive first, then by start time descending.
	s.sortSessionsLocked()
}

// sortSessionsLocked sorts sessionOrder with alive sessions first,
// then by start time (newest first). Caller must hold mu.
func (s *Store) sortSessionsLocked() {
	sort.Slice(s.sessionOrder, func(i, j int) bool {
		si := s.sessions[s.sessionOrder[i]]
		sj := s.sessions[s.sessionOrder[j]]

		// Alive sessions come first.
		if si.Alive != sj.Alive {
			return si.Alive
		}
		// Then sort by start time descending (newest first).
		return si.StartedAt > sj.StartedAt
	})
}

// Sessions returns all sessions in display order.
func (s *Store) Sessions() []data.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]data.Session, 0, len(s.sessionOrder))
	for _, sid := range s.sessionOrder {
		if sess, ok := s.sessions[sid]; ok {
			result = append(result, sess)
		}
	}
	return result
}

// SessionByIndex returns the session at the given display index.
func (s *Store) SessionByIndex(idx int) (data.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx < 0 || idx >= len(s.sessionOrder) {
		return data.Session{}, false
	}
	sess, ok := s.sessions[s.sessionOrder[idx]]
	return sess, ok
}

// SessionCount returns the number of sessions.
func (s *Store) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessionOrder)
}

// AgentsForSession returns agents for the given session ID.
func (s *Store) AgentsForSession(sessionID string) []data.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents, ok := s.agents[sessionID]
	if !ok {
		return nil
	}
	// Return a copy.
	result := make([]data.Agent, len(agents))
	copy(result, agents)
	return result
}

// TasksForSession returns tasks for the given session ID.
func (s *Store) TasksForSession(sessionID string) []data.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks, ok := s.tasks[sessionID]
	if !ok {
		return nil
	}
	result := make([]data.Task, len(tasks))
	copy(result, tasks)
	return result
}

// UpdateSession updates or adds a session in the store.
func (s *Store) UpdateSession(sess data.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sess.SessionID]; !exists {
		s.sessionOrder = append(s.sessionOrder, sess.SessionID)
	}
	s.sessions[sess.SessionID] = sess
	s.sortSessionsLocked()
}

// RemoveSession removes a session and its associated agents/tasks.
func (s *Store) RemoveSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	delete(s.agents, sessionID)
	delete(s.tasks, sessionID)

	// Remove from order.
	newOrder := make([]string, 0, len(s.sessionOrder))
	for _, sid := range s.sessionOrder {
		if sid != sessionID {
			newOrder = append(newOrder, sid)
		}
	}
	s.sessionOrder = newOrder
}

// UpdateAgents replaces all agents for a session.
func (s *Store) UpdateAgents(sessionID string, agents []data.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[sessionID] = agents
}

// UpdateTasks replaces all tasks for a session.
func (s *Store) UpdateTasks(sessionID string, tasks []data.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[sessionID] = tasks
}

// Stats computes aggregate metrics across all sessions.
func (s *Store) Stats() data.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stats data.Stats
	stats.TotalSessions = len(s.sessions)

	for _, sess := range s.sessions {
		if sess.Alive {
			stats.ActiveSessions++
		}
	}

	for _, agents := range s.agents {
		for _, a := range agents {
			stats.TotalAgents++
			switch a.Status {
			case data.StatusRunning:
				stats.RunningAgents++
			case data.StatusIdle:
				stats.IdleAgents++
			case data.StatusDone:
				stats.DoneAgents++
			}
			stats.TotalTokensIn += a.Tokens.InputTokens
			stats.TotalTokensOut += a.Tokens.OutputTokens
			stats.TotalCacheCreation += a.Tokens.CacheCreationTokens
			stats.TotalCacheRead += a.Tokens.CacheReadTokens
		}
	}

	for _, tasks := range s.tasks {
		for _, t := range tasks {
			stats.TotalTasks++
			switch t.Status {
			case data.TaskCompleted:
				stats.CompletedTasks++
			case data.TaskInProgress:
				stats.InProgressTasks++
			case data.TaskPending:
				stats.PendingTasks++
			}
		}
	}

	return stats
}
