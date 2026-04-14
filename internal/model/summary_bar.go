package model

import (
	"fmt"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
)

// SummaryBar renders the fixed bottom bar with aggregate metrics.
type SummaryBar struct {
	stats data.Stats
	width int
}

// NewSummaryBar creates a new SummaryBar.
func NewSummaryBar() SummaryBar {
	return SummaryBar{}
}

// SetStats updates the displayed statistics.
func (sb *SummaryBar) SetStats(stats data.Stats) {
	sb.stats = stats
}

// SetWidth updates the available width.
func (sb *SummaryBar) SetWidth(w int) {
	sb.width = w
}

// View renders the summary bar.
func (sb *SummaryBar) View() string {
	s := sb.stats

	sessInfo := fmt.Sprintf("%d sessions", s.ActiveSessions)

	// Compact agent status: 122 agents (2● 4◐ 116✓)
	agentInfo := fmt.Sprintf("%d agents", s.TotalAgents)
	if s.TotalAgents > 0 {
		agentInfo = fmt.Sprintf("%d agents (%d● %d◐ %d✓)",
			s.TotalAgents, s.RunningAgents, s.IdleAgents, s.DoneAgents)
	}

	// Total tokens from all sessions.
	tokenInfo := ""
	if s.TotalTokensAll > 0 {
		tokenInfo = fmt.Sprintf(" │ %s tokens", data.FormatTokenCount(s.TotalTokensAll))
	}

	taskInfo := ""
	if s.TotalTasks > 0 {
		taskInfo = fmt.Sprintf(" │ %d tasks", s.TotalTasks)
	}

	helpHints := " │ ↑↓ nav │ enter expand │ tab panel │ ? help │ q quit"

	content := fmt.Sprintf(" %s │ %s%s%s%s",
		sessInfo, agentInfo, tokenInfo, taskInfo, helpHints)

	return ui.SummaryBarStyle.Width(sb.width).Render(content)
}
