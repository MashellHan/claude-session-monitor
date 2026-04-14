package model

import (
	"fmt"
	"strings"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// SessionList renders the list of Claude sessions.
type SessionList struct {
	sessions []data.Session
	cursor   int
	height   int
	width    int
	offset   int // scroll offset for viewport
}

// NewSessionList creates a new SessionList.
func NewSessionList() SessionList {
	return SessionList{}
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

// View renders the session list.
func (sl *SessionList) View(focused bool) string {
	if len(sl.sessions) == 0 {
		return ui.MutedStyle.Render("  No sessions found")
	}

	var b strings.Builder

	// Header row.
	header := fmt.Sprintf("  %-6s %-22s %-10s %-9s %-14s",
		"PID", "Project", "Kind", "Uptime", "Agents")
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows.
	visibleRows := sl.visibleRows()
	endIdx := sl.offset + visibleRows
	if endIdx > len(sl.sessions) {
		endIdx = len(sl.sessions)
	}

	for i := sl.offset; i < endIdx; i++ {
		sess := sl.sessions[i]
		isSelected := i == sl.cursor && focused

		// Format PID with alive indicator.
		pidStr := fmt.Sprintf("%d", sess.PID)
		if sess.Alive {
			pidStr = ui.AliveStyle.Render(pidStr)
		} else {
			pidStr = ui.DeadStyle.Render(pidStr)
		}

		// Truncate project name.
		project := sess.Project
		if len(project) > 20 {
			project = project[:17] + "..."
		}

		// Truncate kind.
		kind := sess.Kind
		if len(kind) > 8 {
			kind = kind[:8]
		}

		// Agent count placeholder (will be enriched by app model).
		agentInfo := ""

		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		row := fmt.Sprintf("%s%-6s %-22s %-10s %-9s %s",
			cursor, pidStr, project, kind, sess.Uptime, agentInfo)

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < endIdx-1 {
			b.WriteByte('\n')
		}
	}

	// Truncate to width if needed.
	result := b.String()
	if sl.width > 0 {
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) > sl.width {
				lines[i] = line[:sl.width]
			}
		}
		result = strings.Join(lines, "\n")
	}

	return result
}
