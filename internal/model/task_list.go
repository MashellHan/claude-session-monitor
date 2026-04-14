package model

import (
	"fmt"
	"strings"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
)

// TaskList renders the task list for the selected session.
type TaskList struct {
	tasks  []data.Task
	cursor int
	height int
	width  int
	offset int
}

// NewTaskList creates a new TaskList.
func NewTaskList() TaskList {
	return TaskList{}
}

// SetTasks updates the tasks to display.
func (tl *TaskList) SetTasks(tasks []data.Task) {
	tl.tasks = tasks
	if tl.cursor >= len(tasks) && len(tasks) > 0 {
		tl.cursor = len(tasks) - 1
	}
	if tl.cursor < 0 {
		tl.cursor = 0
	}
}

// SetSize updates the viewport dimensions.
func (tl *TaskList) SetSize(w, h int) {
	tl.width = w
	tl.height = h
}

// CursorUp moves the cursor up.
func (tl *TaskList) CursorUp() {
	if tl.cursor > 0 {
		tl.cursor--
		if tl.cursor < tl.offset {
			tl.offset = tl.cursor
		}
	}
}

// CursorDown moves the cursor down.
func (tl *TaskList) CursorDown() {
	if tl.cursor < len(tl.tasks)-1 {
		tl.cursor++
		visibleRows := tl.visibleRows()
		if tl.cursor >= tl.offset+visibleRows {
			tl.offset = tl.cursor - visibleRows + 1
		}
	}
}

// visibleRows returns how many rows can be displayed.
func (tl *TaskList) visibleRows() int {
	rows := tl.height - 3
	if rows < 1 {
		rows = 1
	}
	return rows
}

// View renders the task list.
func (tl *TaskList) View(focused bool) string {
	if len(tl.tasks) == 0 {
		return ui.MutedStyle.Render("  No tasks found")
	}

	var b strings.Builder

	// Header.
	header := fmt.Sprintf("  %-4s %-8s %-44s %-6s",
		"ID", "Status", "Subject", "Deps")
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows.
	for i, task := range tl.tasks {
		isSelected := i == tl.cursor && focused

		statusStr := formatTaskStatus(task.Status)

		subject := task.Subject
		if len(subject) > 42 {
			subject = subject[:39] + "..."
		}

		deps := formatDeps(task.BlockedBy)

		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		row := fmt.Sprintf("%s%-4s %s %-44s %-6s",
			cursor, task.ID, statusStr, subject, deps)

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < len(tl.tasks)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// formatTaskStatus returns a styled status indicator.
func formatTaskStatus(status data.TaskStatus) string {
	switch status {
	case data.TaskCompleted:
		return ui.StatusDoneStyle.Render(fmt.Sprintf("%-8s", "✓ done"))
	case data.TaskInProgress:
		return ui.StatusRunningStyle.Render(fmt.Sprintf("%-8s", "● work"))
	case data.TaskPending:
		return ui.MutedStyle.Render(fmt.Sprintf("%-8s", "○ pend"))
	default:
		return ui.MutedStyle.Render(fmt.Sprintf("%-8s", "? "+string(status)))
	}
}

// formatDeps formats the blockedBy list as a short string.
func formatDeps(blockedBy []string) string {
	if len(blockedBy) == 0 {
		return ""
	}
	parts := make([]string, len(blockedBy))
	for i, id := range blockedBy {
		parts[i] = "←" + id
	}
	return strings.Join(parts, ",")
}

// TaskSummary returns a summary string for the task panel title.
func TaskSummary(tasks []data.Task) string {
	if len(tasks) == 0 {
		return "0 total"
	}
	completed := 0
	inProgress := 0
	pending := 0
	for _, t := range tasks {
		switch t.Status {
		case data.TaskCompleted:
			completed++
		case data.TaskInProgress:
			inProgress++
		case data.TaskPending:
			pending++
		}
	}
	return fmt.Sprintf("%d total: %d completed, %d in_progress, %d pending",
		len(tasks), completed, inProgress, pending)
}
