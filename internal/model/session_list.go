package model

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// SessionList renders the list of Claude sessions.
type SessionList struct {
	sessions []data.Session
	// agentsBySession maps sessionID to agents for enriched rendering.
	agentsBySession map[string][]data.Agent
	cursor          int
	expandedIdx     int // -1 means no expansion
	height          int
	width           int
	offset          int // scroll offset for viewport
}

// NewSessionList creates a new SessionList.
func NewSessionList() SessionList {
	return SessionList{
		agentsBySession: make(map[string][]data.Agent),
		expandedIdx:     -1,
	}
}

// SetSessions updates the sessions to display.
// Sorts: alive sessions first, then by start time descending (newest first).
func (sl *SessionList) SetSessions(sessions []data.Session) {
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Alive != sessions[j].Alive {
			return sessions[i].Alive // alive first
		}
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})
	sl.sessions = sessions
	if sl.cursor >= len(sessions) && len(sessions) > 0 {
		sl.cursor = len(sessions) - 1
	}
	if sl.cursor < 0 {
		sl.cursor = 0
	}
	if sl.expandedIdx >= len(sessions) {
		sl.expandedIdx = -1
	}
}

// ToggleDetail toggles inline detail expansion for the selected session.
func (sl *SessionList) ToggleDetail() {
	if sl.expandedIdx == sl.cursor {
		sl.expandedIdx = -1
	} else {
		sl.expandedIdx = sl.cursor
	}
}

// SetAgentsForSession stores the agent list for a given session, used to
// enrich the session row with agent count and status info.
func (sl *SessionList) SetAgentsForSession(sessionID string, agents []data.Agent) {
	sl.agentsBySession[sessionID] = agents
}

// ClearAgents resets the agent lookup map for a fresh data cycle.
func (sl *SessionList) ClearAgents() {
	sl.agentsBySession = make(map[string][]data.Agent)
}

// SetSize updates the viewport dimensions.
func (sl *SessionList) SetSize(w, h int) {
	sl.width = w
	sl.height = h
}

// CursorUp moves the cursor up.
func (sl *SessionList) CursorUp() {
	if sl.cursor > 0 {
		sl.cursor--
		if sl.cursor < sl.offset {
			sl.offset = sl.cursor
		}
	}
}

// CursorDown moves the cursor down.
func (sl *SessionList) CursorDown() {
	if sl.cursor < len(sl.sessions)-1 {
		sl.cursor++
		visibleRows := sl.visibleRows()
		if sl.cursor >= sl.offset+visibleRows {
			sl.offset = sl.cursor - visibleRows + 1
		}
	}
}

// visibleRows returns how many rows can be displayed.
func (sl *SessionList) visibleRows() int {
	// Account for header (2 lines) and some padding.
	rows := sl.height - 3
	if rows < 1 {
		rows = 1
	}
	return rows
}

// SelectedSession returns the currently selected session, if any.
func (sl *SessionList) SelectedSession() (data.Session, bool) {
	if sl.cursor >= 0 && sl.cursor < len(sl.sessions) {
		return sl.sessions[sl.cursor], true
	}
	return data.Session{}, false
}

// padRight pads s to exactly width display columns using runewidth,
// handling CJK characters correctly. If s exceeds width, it is truncated.
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return runewidth.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

// truncateRune truncates s to maxWidth display columns, appending "…" if cut.
func truncateRune(s string, maxWidth int) string {
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	return runewidth.Truncate(s, maxWidth-1, "…")
}

// View renders the session list with agent info and proper scroll offset.
func (sl *SessionList) View(focused bool) string {
	if len(sl.sessions) == 0 {
		return ui.MutedStyle.Render("  No sessions found")
	}

	// Column widths — dynamically allocate Topic based on terminal width.
	const (
		colStatus  = 4  // "●" / "○"
		colPID     = 7
		colProject = 16
		colBranch  = 9
		colUptime  = 10
		colMsgs    = 5
		colTokens  = 8
		colAgts    = 7
		colFixed   = colStatus + colPID + colProject + colBranch + colUptime + colMsgs + colTokens + colAgts + 9 // separators + cursor
	)
	colTopic := sl.width - colFixed - 4 // remaining space for topic
	if colTopic < 20 {
		colTopic = 20
	}
	if colTopic > 60 {
		colTopic = 60
	}

	var b strings.Builder

	// Header row with fixed columns.
	header := fmt.Sprintf("  %s│%s│%s│%s│%s│%s│%s│%s│%s",
		padRight("", colStatus),
		padRight("PID", colPID),
		padRight("Project", colProject),
		padRight("Topic", colTopic),
		padRight("Branch", colBranch),
		padRight("Uptime", colUptime),
		padRight("Msgs", colMsgs),
		padRight("Tokens", colTokens),
		padRight("Agts", colAgts))
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows — respect scroll offset.
	visibleRows := sl.visibleRows()
	endIdx := sl.offset + visibleRows
	if endIdx > len(sl.sessions) {
		endIdx = len(sl.sessions)
	}

	for i := sl.offset; i < endIdx; i++ {
		sess := sl.sessions[i]
		isSelected := i == sl.cursor && focused

		// Status indicator.
		var statusStr string
		if sess.Alive {
			statusStr = ui.StatusRunningStyle.Render(padRight("●", colStatus))
		} else {
			statusStr = ui.DeadStyle.Render(padRight("○", colStatus))
		}

		// Prepare each cell.
		pidStr := fmt.Sprintf("%d", sess.PID)
		project := truncateRune(sess.Project, colProject-1)
		topic := truncateRune(sess.Topic, colTopic-1)
		// If project was derived from topic, show "~" prefix and suppress
		// topic column to avoid showing the same text in both columns.
		if sess.ProjectFromTopic {
			project = "~" + truncateRune(sess.Project, colProject-2)
			topic = "" // avoid duplicate display
		}
		branch := truncateRune(sess.GitBranch, colBranch-1)
		tokens := sess.Tokens.Formatted()

		uptime := sess.Uptime
		if !sess.StartTime.IsZero() {
			uptime = data.FormatUptime(time.Since(sess.StartTime))
		}

		msgs := ""
		if sess.MessageCount > 0 {
			msgs = fmt.Sprintf("%d", sess.MessageCount)
		}

		agents := sl.agentsBySession[sess.SessionID]

		// Build cursor prefix.
		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		// Agent count — rendered with color, so pad the plain text first.
		agentInfo := AgentCountSummary(agents)

		// Build row with separators for column alignment.
		row := fmt.Sprintf("%s%s│%s│%s│%s│%s│%s│%s│%s│%s",
			cursor,
			statusStr,
			padRight(pidStr, colPID),
			padRight(project, colProject),
			padRight(topic, colTopic),
			padRight(branch, colBranch),
			padRight(uptime, colUptime),
			padRight(msgs, colMsgs),
			padRight(tokens, colTokens),
			agentInfo)

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)

		// Inline session detail expansion.
		if i == sl.expandedIdx {
			detail := renderSessionDetail(sess, sl.agentsBySession[sess.SessionID], sl.width)
			b.WriteByte('\n')
			b.WriteString(detail)
		}

		if i < endIdx-1 {
			b.WriteByte('\n')
		}
	}

	// Scroll indicator.
	if len(sl.sessions) > visibleRows {
		scrollPct := 0
		if len(sl.sessions)-visibleRows > 0 {
			scrollPct = sl.offset * 100 / (len(sl.sessions) - visibleRows)
		}
		b.WriteByte('\n')
		b.WriteString(ui.MutedStyle.Render(
			fmt.Sprintf("  ↕ %d/%d (scroll %d%%)", sl.offset+1, len(sl.sessions), scrollPct)))
	}

	// ANSI-safe width truncation.
	result := b.String()
	if sl.width > 0 {
		truncStyle := lipgloss.NewStyle().MaxWidth(sl.width)
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) > sl.width {
				lines[i] = truncStyle.Render(line)
			}
		}
		result = strings.Join(lines, "\n")
	}

	return result
}

// renderSessionDetail renders inline detail for an expanded session row.
func renderSessionDetail(sess data.Session, agents []data.Agent, maxWidth int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Session ID:  %s\n", sess.SessionID))
	b.WriteString(fmt.Sprintf("PID:         %d (%s)\n", sess.PID, statusLabel(sess.Alive)))
	b.WriteString(fmt.Sprintf("CWD:         %s\n", sess.CWD))
	if sess.Kind != "" {
		b.WriteString(fmt.Sprintf("Kind:        %s\n", sess.Kind))
	}
	if sess.Project != "" {
		b.WriteString(fmt.Sprintf("Project:     %s\n", sess.Project))
	}
	if sess.GitBranch != "" {
		b.WriteString(fmt.Sprintf("Branch:      %s\n", sess.GitBranch))
	}
	if !sess.StartTime.IsZero() {
		b.WriteString(fmt.Sprintf("Started:     %s (%s)\n",
			sess.StartTime.Format("2006-01-02 15:04:05"),
			data.FormatUptime(time.Since(sess.StartTime))))
	}
	if sess.MessageCount > 0 {
		b.WriteString(fmt.Sprintf("Messages:    %d\n", sess.MessageCount))
	}

	// Token breakdown.
	if sess.Tokens.Total() > 0 {
		b.WriteString(fmt.Sprintf("Tokens:      In: %s  Out: %s  Cache Read: %s  (Total: %s)\n",
			data.FormatTokenCount(sess.Tokens.InputTokens),
			data.FormatTokenCount(sess.Tokens.OutputTokens),
			data.FormatTokenCount(sess.Tokens.CacheReadTokens),
			sess.Tokens.Formatted()))
	}

	// Full topic.
	if sess.TopicFull != "" {
		topicDisplay := sess.TopicFull
		if len([]rune(topicDisplay)) > 120 {
			topicDisplay = string([]rune(topicDisplay)[:120]) + "..."
		}
		b.WriteString(fmt.Sprintf("Topic:       %s\n", topicDisplay))
	}

	// Agent summary.
	if len(agents) > 0 {
		running, idle, done := 0, 0, 0
		for _, a := range agents {
			switch a.Status {
			case data.StatusRunning:
				running++
			case data.StatusIdle:
				idle++
			case data.StatusDone:
				done++
			}
		}
		b.WriteString(fmt.Sprintf("Agents:      %d total (%d running, %d idle, %d done)\n",
			len(agents), running, idle, done))
	}

	if sess.JSONLSize > 0 {
		sizeStr := data.FormatTokenCount(sess.JSONLSize)
		b.WriteString(fmt.Sprintf("JSONL Size:  %sB\n", sizeStr))
	}

	detailWidth := maxWidth - 6
	if detailWidth < 40 {
		detailWidth = 40
	}

	return ui.DetailBorderStyle.Width(detailWidth).Render(b.String())
}

// statusLabel returns a human-readable alive/dead label.
func statusLabel(alive bool) string {
	if alive {
		return "alive"
	}
	return "dead"
}
