package model

import (
	"fmt"
	"strings"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
)

// DetailView renders an inline detail expansion for an agent.
// This is used when the user presses Enter on an agent row.
type DetailView struct {
	agent   *data.Agent
	visible bool
	width   int
}

// NewDetailView creates a new DetailView.
func NewDetailView() DetailView {
	return DetailView{}
}

// SetAgent sets the agent whose detail is being viewed.
func (dv *DetailView) SetAgent(agent *data.Agent) {
	dv.agent = agent
	dv.visible = agent != nil
}

// SetSize sets the available width.
func (dv *DetailView) SetSize(w int) {
	dv.width = w
}

// Toggle toggles visibility.
func (dv *DetailView) Toggle() {
	dv.visible = !dv.visible
}

// IsVisible returns whether the detail view is shown.
func (dv *DetailView) IsVisible() bool {
	return dv.visible && dv.agent != nil
}

// View renders the detail view content.
func (dv *DetailView) View() string {
	if !dv.visible || dv.agent == nil {
		return ""
	}

	agent := dv.agent
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

	detailWidth := dv.width - 6
	if detailWidth < 40 {
		detailWidth = 40
	}

	return ui.DetailBorderStyle.Width(detailWidth).Render(b.String())
}
