package model

import (
	"fmt"
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
	height          int
	width           int
	offset          int // scroll offset for viewport
}

// NewSessionList creates a new SessionList.
func NewSessionList() SessionList {
	return SessionList{
		agentsBySession: make(map[string][]data.Agent),
	}
}

// SetSessions updates the sessions to display.
func (sl *SessionList) SetSessions(sessions []data.Session) {
	sl.sessions = sessions
	if sl.cursor >= len(sessions) && len(sessions) > 0 {
		sl.cursor = len(sessions) - 1
	}
	if sl.cursor < 0 {
		sl.cursor = 0
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
		colPID     = 7
		colProject = 16
		colBranch  = 9
		colUptime  = 10
		colTokens  = 8
		colAgts    = 7
		colFixed   = colPID + colProject + colBranch + colUptime + colTokens + colAgts + 6 // separators + cursor
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
	header := fmt.Sprintf("  %s│%s│%s│%s│%s│%s│%s",
		padRight("PID", colPID),
		padRight("Project", colProject),
		padRight("Topic", colTopic),
		padRight("Branch", colBranch),
		padRight("Uptime", colUptime),
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

		// Prepare each cell.
		pidStr := fmt.Sprintf("%d", sess.PID)
		project := truncateRune(sess.Project, colProject-1)
		topic := truncateRune(sess.Topic, colTopic-1)
		branch := truncateRune(sess.GitBranch, colBranch-1)
		tokens := sess.Tokens.Formatted()

		uptime := sess.Uptime
		if !sess.StartTime.IsZero() {
			uptime = data.FormatUptime(time.Since(sess.StartTime))
		}

		agents := sl.agentsBySession[sess.SessionID]

		// Build cursor prefix.
		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		// Color PID.
		if sess.Alive {
			pidStr = ui.AliveStyle.Render(padRight(pidStr, colPID))
		} else {
			pidStr = ui.DeadStyle.Render(padRight(pidStr, colPID))
		}

		// Agent count — rendered with color, so pad the plain text first.
		agentInfo := AgentCountSummary(agents)

		// Build row with separators for column alignment.
		row := fmt.Sprintf("%s%s│%s│%s│%s│%s│%s│%s",
			cursor,
			pidStr,
			padRight(project, colProject),
			padRight(topic, colTopic),
			padRight(branch, colBranch),
			padRight(uptime, colUptime),
			padRight(tokens, colTokens),
			agentInfo)

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < endIdx-1 {
			b.WriteByte('\n')
		}
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
