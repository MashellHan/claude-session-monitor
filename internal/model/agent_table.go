package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
	"github.com/mattn/go-runewidth"
)

// AgentTable renders the agent table for the selected session.
type AgentTable struct {
	agents       []data.Agent
	sessionTopic string // topic of the parent session for the header
	sessionPID   int    // PID of the parent session
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

// SetSessionContext sets the parent session info for the header.
func (at *AgentTable) SetSessionContext(pid int, topic string) {
	at.sessionPID = pid
	at.sessionTopic = topic
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

	// Column widths for agent table.
	const (
		colIdx     = 4
		colType    = 18
		colStatus  = 10
		colElapsed = 10
		colModel   = 8
		colTokens  = 8
		colFixed   = colIdx + colType + colStatus + colElapsed + colModel + colTokens + 8
	)
	colDesc := at.width - colFixed
	if colDesc < 20 {
		colDesc = 20
	}
	if colDesc > 60 {
		colDesc = 60
	}

	var b strings.Builder

	// Header.
	header := fmt.Sprintf("  %s│%s│%s│%s│%s│%s│%s",
		padRight("#", colIdx),
		padRight("Agent Type", colType),
		padRight("Status", colStatus),
		padRight("Last Act", colElapsed),
		padRight("Model", colModel),
		padRight("Tokens", colTokens),
		padRight("Description", colDesc))
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows.
	for i, agent := range at.agents {
		isSelected := i == at.cursor && focused

		// Status with color — rendered with ANSI, so we render it separately.
		statusStr := formatAgentStatus(agent.Status)

		// Prepare plain text cells.
		idx := fmt.Sprintf("%d", i+1)
		agentType := runewidth.Truncate(agent.AgentType, colType-1, "…")
		model := runewidth.Truncate(agent.Model, colModel-1, "…")
		tokens := agent.Tokens.Formatted()
		desc := runewidth.Truncate(agent.Description, colDesc-1, "…")

		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		row := fmt.Sprintf("%s%s│%s│%s│%s│%s│%s│%s",
			cursor,
			padRight(idx, colIdx),
			padRight(agentType, colType),
			statusStr+strings.Repeat(" ", max(0, colStatus-runewidth.StringWidth(statusStr))),
			padRight(agent.Elapsed, colElapsed),
			padRight(model, colModel),
			padRight(tokens, colTokens),
			desc)

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatAgentStatus returns a styled status string with symbol, padded to 8 display columns.
func formatAgentStatus(status data.AgentStatus) string {
	symbol := status.Symbol()
	label := status.String()
	display := symbol + " " + label

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

	b.WriteString(fmt.Sprintf("Agent ID:    %s\n", agent.ID))
	b.WriteString(fmt.Sprintf("Type:        %s\n", agent.AgentType))
	if agent.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", agent.Description))
	}
	if agent.ModelFull != "" {
		b.WriteString(fmt.Sprintf("Model:       %s\n", agent.ModelFull))
	} else if agent.Model != "" {
		b.WriteString(fmt.Sprintf("Model:       %s\n", agent.Model))
	}
	if !agent.LastActiveTime.IsZero() {
		b.WriteString(fmt.Sprintf("Last Active: %s\n", agent.LastActiveTime.Format("2006-01-02 15:04:05")))
	}

	// Token breakdown: In: 6,234  Out: 1,988  Cache Read: 890  Cache Create: 0
	if agent.Tokens.Total() > 0 {
		b.WriteString(fmt.Sprintf("Tokens:      In: %s  Out: %s  Cache Read: %s  Cache Create: %s  (Total: %s)\n",
			formatTokenComma(agent.Tokens.InputTokens),
			formatTokenComma(agent.Tokens.OutputTokens),
			formatTokenComma(agent.Tokens.CacheReadTokens),
			formatTokenComma(agent.Tokens.CacheCreationTokens),
			agent.Tokens.Formatted()))
	}

	// Tool call summary: 12 total (Grep: 4, Read: 3, Glob: 3, Bash: 2)
	if agent.ToolCallTotal > 0 && agent.ToolCallMap != nil {
		summary := formatToolCallSummary(agent.ToolCallMap, agent.ToolCallTotal)
		b.WriteString(fmt.Sprintf("Tool Calls:  %s\n", summary))
	}

	if len(agent.ToolCalls) > 0 {
		b.WriteString("\nRecent Activity:\n")
		// Show last 5 tool calls.
		start := 0
		if len(agent.ToolCalls) > 5 {
			start = len(agent.ToolCalls) - 5
		}
		for _, tc := range agent.ToolCalls[start:] {
			line := fmt.Sprintf("  → %s", tc.Name)
			if tc.Input != "" {
				inputStr := tc.Input
				if len(inputStr) > 60 {
					inputStr = inputStr[:57] + "..."
				}
				line += ": " + inputStr
			}
			b.WriteString(line + "\n")
		}
	}

	// Final output.
	if agent.FinalOutput != "" {
		output := agent.FinalOutput
		// Sanitize for display.
		output = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(output)
		if len(output) > 200 {
			output = output[:200]
		}
		b.WriteString(fmt.Sprintf("\nResult: \"%s\"\n", output))
	}

	detailWidth := maxWidth - 6
	if detailWidth < 40 {
		detailWidth = 40
	}

	return ui.DetailBorderStyle.Width(detailWidth).Render(b.String())
}

// formatTokenComma formats a token count with comma separators.
func formatTokenComma(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	// Insert commas from the right.
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}

// formatToolCallSummary creates "12 total (Grep: 4, Read: 3, Glob: 3, Bash: 2)"
func formatToolCallSummary(toolMap map[string]int, total int) string {
	// Sort by count descending.
	type toolCount struct {
		name  string
		count int
	}
	sorted := make([]toolCount, 0, len(toolMap))
	for name, count := range toolMap {
		sorted = append(sorted, toolCount{name, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].name < sorted[j].name
	})

	var parts []string
	for _, tc := range sorted {
		parts = append(parts, fmt.Sprintf("%s: %d", tc.name, tc.count))
	}
	return fmt.Sprintf("%d total (%s)", total, strings.Join(parts, ", "))
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

// AgentCountSummary returns a compact agent count for the session table.
// Format: "4" or "4 ●1" (4 total, 1 running).
func AgentCountSummary(agents []data.Agent) string {
	if len(agents) == 0 {
		return ui.MutedStyle.Render("0")
	}

	total := len(agents)
	running := 0
	for _, a := range agents {
		if a.Status == data.StatusRunning {
			running++
		}
	}

	if running > 0 {
		return ui.StatusRunningStyle.Render(fmt.Sprintf("%d ●%d", total, running))
	}
	return ui.MutedStyle.Render(fmt.Sprintf("%d", total))
}
