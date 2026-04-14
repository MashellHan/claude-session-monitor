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

	agentInfo := fmt.Sprintf("%d agents", s.TotalAgents)
	if s.TotalAgents > 0 {
		agentInfo = fmt.Sprintf("%d agents (%d running, %d idle, %d done)",
			s.TotalAgents, s.RunningAgents, s.IdleAgents, s.DoneAgents)
	}

	tokenInfo := ""
	totalTokens := s.TotalTokensIn + s.TotalTokensOut
	if totalTokens > 0 {
		tokenInfo = fmt.Sprintf(" │ %s tokens", data.FormatTokens(totalTokens))
	}

	taskInfo := ""
	if s.TotalTasks > 0 {
		taskInfo = fmt.Sprintf(" │ %d tasks", s.TotalTasks)
	}

	helpHints := " │ ↑↓ nav │ enter expand │ tab panel │ q quit │ ? help"

	content := fmt.Sprintf(" %s │ %s%s%s%s",
		sessInfo, agentInfo, tokenInfo, taskInfo, helpHints)

	return ui.SummaryBarStyle.Width(sb.width).Render(content)
}
