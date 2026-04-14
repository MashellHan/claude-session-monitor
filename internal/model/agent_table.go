package model

import (
	"fmt"
	"strings"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
)

// AgentTable renders the agent table for the selected session.
type AgentTable struct {
	agents       []data.Agent
	cursor       int
	expandedIdx  int // -1 means no expansion
	height       int
	width        int
	offset       int
}

// NewAgentTable creates a new AgentTable.
func NewAgentTable() AgentTable {
	return AgentTable{expandedIdx: -1}
}

// SetAgents updates the agents to display.
func (at *AgentTable) SetAgents(agents []data.Agent) {
	at.agents = agents
	if at.cursor >= len(agents) && len(agents) > 0 {
		at.cursor = len(agents) - 1
	}
	if at.cursor < 0 {
		at.cursor = 0
	}
	// Collapse detail if agents changed.
	if at.expandedIdx >= len(agents) {
		at.expandedIdx = -1
	}
}

// SetSize updates the viewport dimensions.
func (at *AgentTable) SetSize(w, h int) {
	at.width = w
	at.height = h
}

// CursorUp moves the cursor up.
func (at *AgentTable) CursorUp() {
	if at.cursor > 0 {
		at.cursor--
		if at.cursor < at.offset {
			at.offset = at.cursor
		}
	}
}

// CursorDown moves the cursor down.
func (at *AgentTable) CursorDown() {
	if at.cursor < len(at.agents)-1 {
		at.cursor++
		visibleRows := at.visibleRows()
		if at.cursor >= at.offset+visibleRows {
			at.offset = at.cursor - visibleRows + 1
		}
	}
}

// visibleRows returns how many rows can be displayed.
func (at *AgentTable) visibleRows() int {
	rows := at.height - 3
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ToggleDetail toggles the inline detail expansion for the current agent.
func (at *AgentTable) ToggleDetail() {
	if at.expandedIdx == at.cursor {
		at.expandedIdx = -1 // Collapse.
	} else {
		at.expandedIdx = at.cursor // Expand.
	}
}

// SelectedAgent returns the currently selected agent, if any.
func (at *AgentTable) SelectedAgent() (data.Agent, bool) {
	if at.cursor >= 0 && at.cursor < len(at.agents) {
		return at.agents[at.cursor], true
	}
	return data.Agent{}, false
}

// View renders the agent table.
func (at *AgentTable) View(focused bool) string {
	if len(at.agents) == 0 {
		return ui.MutedStyle.Render("  No agents found")
	}

	var b strings.Builder

	// Header row.
	header := fmt.Sprintf("  %-18s %-10s %-9s %-30s",
		"Agent Type", "Status", "Elapsed", "Description")
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows.
	for i, agent := range at.agents {
		isSelected := i == at.cursor && focused

		// Status with color.
		statusStr := formatAgentStatus(agent.Status)

		// Truncate description.
		desc := agent.Description
		if len(desc) > 28 {
			desc = desc[:25] + "..."
		}

		// Truncate agent type.
		agentType := agent.AgentType
		if len(agentType) > 16 {
			agentType = agentType[:13] + "..."
		}

		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		row := fmt.Sprintf("%s%-18s %s %-9s %-30s",
			cursor, agentType, statusStr, agent.Elapsed, desc)

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteByte('\n')

		// Inline detail expansion.
		if i == at.expandedIdx {
			detail := renderAgentDetail(agent, at.width)
			b.WriteString(detail)
			b.WriteByte('\n')
		}
	}

	result := b.String()
	if sl := strings.TrimRight(result, "\n"); sl != "" {
		result = sl
	}

	return result
}

// formatAgentStatus returns a styled status string with symbol.
func formatAgentStatus(status data.AgentStatus) string {
	symbol := status.Symbol()
	label := status.String()
	display := fmt.Sprintf("%-8s", symbol+" "+label)

	switch status {
	case data.StatusRunning:
		return ui.StatusRunningStyle.Render(display)
	case data.StatusIdle:
		return ui.StatusIdleStyle.Render(display)
	case data.StatusDone:
		return ui.StatusDoneStyle.Render(display)
	default:
		return ui.MutedStyle.Render(display)
	}
}

// renderAgentDetail renders the inline detail view for an agent.
func renderAgentDetail(agent data.Agent, maxWidth int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Agent ID: %s\n", agent.ID))
	if agent.Model != "" {
		b.WriteString(fmt.Sprintf("Model: %s\n", agent.Model))
	}
	b.WriteString(fmt.Sprintf("Tokens: %s in / %s out\n",
		data.FormatTokens(agent.Tokens.InputTokens),
		data.FormatTokens(agent.Tokens.OutputTokens)))

	if len(agent.ToolCalls) > 0 {
		b.WriteString("\nLatest activity:\n")
		// Show last 5 tool calls.
		start := 0
		if len(agent.ToolCalls) > 5 {
			start = len(agent.ToolCalls) - 5
		}
		for _, tc := range agent.ToolCalls[start:] {
			line := fmt.Sprintf("  → %s", tc.Name)
			if tc.Input != "" {
				inputStr := tc.Input
				if len(inputStr) > 50 {
					inputStr = inputStr[:47] + "..."
				}
				line += ": " + inputStr
			}
			b.WriteString(line + "\n")
		}
	}

	detailWidth := maxWidth - 6
	if detailWidth < 40 {
		detailWidth = 40
	}

	return ui.DetailBorderStyle.Width(detailWidth).Render(b.String())
}

// AgentSummary returns a short summary string for display in the session list.
func AgentSummary(agents []data.Agent) string {
	if len(agents) == 0 {
		return ui.MutedStyle.Render("0 agents")
	}

	running := 0
	idle := 0
	for _, a := range agents {
		switch a.Status {
		case data.StatusRunning:
			running++
		case data.StatusIdle:
			idle++
		}
	}

	total := len(agents)
	if running > 0 {
		return ui.StatusRunningStyle.Render(fmt.Sprintf("%d (%d running)", total, running))
	}
	if idle > 0 {
		return ui.StatusIdleStyle.Render(fmt.Sprintf("%d (%d idle)", total, idle))
	}
	return ui.MutedStyle.Render(fmt.Sprintf("%d done", total))
}
