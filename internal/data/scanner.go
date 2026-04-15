package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	// maxJSONLTailBytes is how many bytes to read from the end of large JSONL files.
	maxJSONLTailBytes = 8 * 1024 // 8KB
)

// Scanner performs a full scan of the ~/.claude/ directory tree to discover
// sessions, tasks, and agents.
type Scanner struct {
	claudeDir  string            // e.g. ~/.claude
	projectMap map[string]string // CWD → project name
}

// NewScanner creates a Scanner rooted at the given Claude data directory.
func NewScanner(claudeDir string) *Scanner {
	return &Scanner{
		claudeDir:  claudeDir,
		projectMap: make(map[string]string),
	}
}

// ScanResult holds the complete result of a full filesystem scan.
type ScanResult struct {
	Sessions []Session
	Agents   map[string][]Agent // sessionID → agents
	Tasks    map[string][]Task  // sessionID → tasks
}

// FullScan performs the complete startup scan:
//  1. Load project name map
//  2. Scan all session files
//  3. Check PID liveness for each session
//  4. Discover tasks and agents for active sessions
func (s *Scanner) FullScan() (*ScanResult, error) {
	result := &ScanResult{
		Agents: make(map[string][]Agent),
		Tasks:  make(map[string][]Task),
	}

	// Step 1: Load project names from homunculus/projects.json.
	projectsPath := filepath.Join(s.claudeDir, "homunculus", "projects.json")
	if pm, err := ParseProjectsJSON(projectsPath); err == nil {
		s.projectMap = pm
	}

	// Step 2: Scan session files.
	sessionsDir := filepath.Join(s.claudeDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // No sessions directory — graceful degradation.
		}
		return result, fmt.Errorf("read sessions dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		sess, err := ParseSession(filepath.Join(sessionsDir, e.Name()))
		if err != nil {
			continue // Skip malformed session files.
		}

		// Step 3: Check PID liveness.
		sess.Alive = IsPIDAlive(sess.PID)

		// Compute uptime.
		if !sess.StartTime.IsZero() {
			sess.Uptime = FormatUptime(time.Since(sess.StartTime))
		}

		// Compute path hash.
		sess.PathHash = PathToHash(sess.CWD)

		// Resolve project name.
		sess.Project = s.resolveProjectName(sess.CWD)

		// v1.1: Discover session JSONL and extract topic, branch, tokens, message count.
		jsonlPath := filepath.Join(s.claudeDir, "projects", sess.PathHash, sess.SessionID+".jsonl")
		if info, err := os.Stat(jsonlPath); err == nil {
			sess.JSONLPath = jsonlPath
			sess.JSONLSize = info.Size()

			topic, topicFull := ParseSessionTopic(jsonlPath)
			sess.Topic = topic
			sess.TopicFull = topicFull

			sess.GitBranch = ParseGitBranch(jsonlPath)
			sess.MessageCount = ParseMessageCount(jsonlPath)
			sess.Tokens = ParseSessionTokens(jsonlPath)
		}

		// If project name is still generic, try to extract from topic.
		// Mark it as topic-derived so the UI can avoid showing duplicates.
		if isGenericProjectName(sess.Project) && sess.Topic != "" {
			derived := truncateProjectFromTopic(sess.Topic)
			// Only use topic-derived name if it's meaningfully different from generic.
			if derived != "" && !isGenericProjectName(derived) {
				sess.Project = derived
				sess.ProjectFromTopic = true
			}
		}

		result.Sessions = append(result.Sessions, sess)

		// Step 4: Load tasks and agents for all sessions (not just alive ones,
		// since we want to show recently dead sessions too).
		tasks := s.scanTasks(sess.SessionID)
		if len(tasks) > 0 {
			result.Tasks[sess.SessionID] = tasks
		}

		agents := s.scanAgents(sess)
		if len(agents) > 0 {
			result.Agents[sess.SessionID] = agents
		}
	}

	return result, nil
}

// scanTasks discovers all task files for a given session.
func (s *Scanner) scanTasks(sessionID string) []Task {
	tasksDir := filepath.Join(s.claudeDir, "tasks", sessionID)
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil
	}

	var tasks []Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t, err := ParseTask(filepath.Join(tasksDir, e.Name()))
		if err != nil {
			continue
		}
		t.SessionID = sessionID
		tasks = append(tasks, t)
	}
	return tasks
}

// scanAgents discovers all subagents for a session by looking in the
// projects/{pathHash}/{sessionId}/subagents/ directory.
func (s *Scanner) scanAgents(sess Session) []Agent {
	if sess.PathHash == "" {
		return nil
	}

	agentsDir := filepath.Join(s.claudeDir, "projects", sess.PathHash, sess.SessionID, "subagents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}

	// Collect agent IDs from meta.json filenames.
	agentIDs := make(map[string]bool)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "agent-") && strings.HasSuffix(name, ".meta.json") {
			id := strings.TrimPrefix(name, "agent-")
			id = strings.TrimSuffix(id, ".meta.json")
			agentIDs[id] = true
		}
	}

	var agents []Agent
	for id := range agentIDs {
		agent := s.buildAgent(id, sess.SessionID, agentsDir)
		// Skip compact agents — they are internal compaction artifacts.
		if agent.AgentType == "compact" {
			continue
		}
		agents = append(agents, agent)
	}

	// Sort agents: running first, then idle, then done.
	// Within same status, sort by last active time descending (most recent first).
	sort.Slice(agents, func(i, j int) bool {
		if agents[i].Status != agents[j].Status {
			return agents[i].Status < agents[j].Status // Running(1) < Idle(2) < Done(3)
		}
		return agents[i].LastActiveTime.After(agents[j].LastActiveTime)
	})

	return agents
}

// buildAgent constructs a full Agent struct from its meta.json and JSONL files.
func (s *Scanner) buildAgent(id, sessionID, agentsDir string) Agent {
	agent := Agent{
		ID:        id,
		SessionID: sessionID,
		MetaPath:  filepath.Join(agentsDir, "agent-"+id+".meta.json"),
		JSONLPath: filepath.Join(agentsDir, "agent-"+id+".jsonl"),
	}

	// Skip compact agents — they are internal compaction artifacts.
	if strings.HasPrefix(id, "compact-") {
		agent.AgentType = "compact"
		agent.Description = "(compaction)"
		agent.Status = StatusDone
		return agent
	}

	// Parse meta.json for type and description.
	if meta, err := ParseAgentMeta(agent.MetaPath); err == nil {
		agent.AgentType = meta.AgentType
		agent.Description = meta.Description
	}

	// Determine agent status from JSONL mtime.
	agent.Status = DetectAgentStatus(agent.JSONLPath)

	// Single-pass parse of agent JSONL for all data: tokens, model, tool calls, final output.
	// This avoids opening the same file 5 times.
	agentInfo := ParseAgentJSONLFull(agent.JSONLPath)
	agent.Tokens = agentInfo.Tokens
	if agentInfo.ModelFull != "" {
		agent.Model = agentInfo.Model
		agent.ModelFull = agentInfo.ModelFull
	}
	agent.ToolCalls = agentInfo.RecentToolCalls
	agent.ToolCallMap = agentInfo.ToolCallMap
	agent.ToolCallTotal = agentInfo.ToolCallTotal
	agent.FinalOutput = agentInfo.FinalOutput

	// Approximate last active time from JSONL file mtime.
	// Note: this is the time of the last JSONL write, not the agent's start time.
	if info, err := os.Stat(agent.JSONLPath); err == nil {
		agent.LastActiveTime = info.ModTime()
		agent.Elapsed = FormatRelativeTime(time.Since(info.ModTime()))
	}

	return agent
}

// DetectAgentStatus determines an agent's status based on its JSONL file's mtime.
func DetectAgentStatus(jsonlPath string) AgentStatus {
	info, err := os.Stat(jsonlPath)
	if err != nil {
		return StatusUnknown
	}

	age := time.Since(info.ModTime())
	switch {
	case age < 30*time.Second:
		return StatusRunning
	case age < 5*time.Minute:
		return StatusIdle
	default:
		return StatusDone
	}
}

// IsPIDAlive checks if a process with the given PID is still running.
// Uses kill(pid, 0) which checks for process existence without sending a signal.
func IsPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. We need to send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// resolveProjectName maps a CWD to a human-readable project name.
// It first checks the homunculus projects.json map, then falls back to
// the last meaningful path component (skipping generic names like "Tmp").
func (s *Scanner) resolveProjectName(cwd string) string {
	if name, ok := s.projectMap[cwd]; ok {
		return name
	}
	// Fallback: use last path component, but skip generic directory names.
	base := filepath.Base(cwd)
	if genericNames[base] {
		// Try parent directory name instead.
		parent := filepath.Base(filepath.Dir(cwd))
		if parent != "" && parent != "." && parent != "/" && !genericNames[parent] {
			return parent
		}
	}
	return base
}

// WatchPaths returns the list of directories that should be watched
// for filesystem events.
func (s *Scanner) WatchPaths() []string {
	return []string{
		filepath.Join(s.claudeDir, "sessions"),
		filepath.Join(s.claudeDir, "tasks"),
		filepath.Join(s.claudeDir, "projects"),
	}
}

// genericNames is a set of directory names that are too generic to use as project names.
var genericNames = map[string]bool{
	"Tmp": true, "tmp": true, "temp": true, "Temp": true,
	"src": true, "projects": true, "Projects": true,
	"code": true, "Code": true, "workspace": true, "Workspace": true,
	"dev": true, "Dev": true, "work": true, "Work": true,
	"home": true, "Home": true, "Desktop": true, "desktop": true,
	"Documents": true, "documents": true,
}

// isGenericProjectName returns true if the name is too generic to be useful.
func isGenericProjectName(name string) bool {
	return genericNames[name]
}

// truncateProjectFromTopic extracts a short project identifier from the session topic.
// Handles special cases like URLs (extracts repo name) and CJK text.
func truncateProjectFromTopic(topic string) string {
	// If topic starts with a URL, extract the repo/path name.
	words := strings.Fields(topic)
	if len(words) == 0 {
		return topic
	}

	firstWord := words[0]
	if strings.HasPrefix(firstWord, "http://") || strings.HasPrefix(firstWord, "https://") {
		parts := strings.Split(firstWord, "/")
		// Try to find a meaningful segment (last non-empty path segment).
		for i := len(parts) - 1; i >= 3; i-- {
			seg := strings.TrimSpace(parts[i])
			if seg != "" && seg != "." {
				if len(seg) > 14 {
					return seg[:14]
				}
				return seg
			}
		}
		// Fall back to hostname.
		if len(parts) > 2 {
			host := parts[2]
			if len(host) > 14 {
				return host[:14]
			}
			return host
		}
	}

	// For regular text, use first 2-3 meaningful words.
	n := len(words)
	if n > 3 {
		n = 3
	}
	short := strings.Join(words[:n], " ")

	// Truncate to 14 runes.
	runes := []rune(short)
	if len(runes) > 14 {
		return string(runes[:14])
	}
	return short
}
