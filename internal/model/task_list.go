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

	const (
		colID     = 5
		colStatus = 9
		colDeps   = 7
	)
	colSubject := tl.width - colID - colStatus - colDeps - 8
	if colSubject < 20 {
		colSubject = 20
	}
	if colSubject > 60 {
		colSubject = 60
	}

	var b strings.Builder

	// Header.
	header := fmt.Sprintf("  %s│%s│%s│%s",
		padRight("ID", colID),
		padRight("Status", colStatus),
		padRight("Subject", colSubject),
		padRight("Deps", colDeps))
	b.WriteString(ui.HeaderStyle.Render(header))
	b.WriteByte('\n')

	// Data rows.
	visibleRows := tl.visibleRows()
	endIdx := tl.offset + visibleRows
	if endIdx > len(tl.tasks) {
		endIdx = len(tl.tasks)
	}

	for i := tl.offset; i < endIdx; i++ {
		task := tl.tasks[i]
		isSelected := i == tl.cursor && focused

		statusStr := formatTaskStatus(task.Status)

		subject := truncateRune(task.Subject, colSubject-1)
		deps := formatDeps(task.BlockedBy)

		cursor := "  "
		if isSelected {
			cursor = ui.CursorStyle.Render("▸ ")
		}

		row := fmt.Sprintf("%s%s│%s│%s│%s",
			cursor,
			padRight(task.ID, colID),
			statusStr+strings.Repeat(" ", max(0, colStatus-statusStrWidth(task.Status))),
			padRight(subject, colSubject),
			padRight(deps, colDeps))

		if isSelected {
			row = ui.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < endIdx-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// formatTaskStatus returns a styled status indicator.
func formatTaskStatus(status data.TaskStatus) string {
	switch status {
	case data.TaskCompleted:
		return ui.StatusDoneStyle.Render("✓ done")
	case data.TaskInProgress:
		return ui.StatusRunningStyle.Render("● work")
	case data.TaskPending:
		return ui.MutedStyle.Render("○ pend")
	default:
		return ui.MutedStyle.Render("? " + string(status))
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

// statusStrWidth returns the display width of the formatted task status string (without ANSI).
func statusStrWidth(status data.TaskStatus) int {
	switch status {
	case data.TaskCompleted:
		return 6 // "✓ done"
	case data.TaskInProgress:
		return 6 // "● work"
	case data.TaskPending:
		return 6 // "○ pend"
	default:
		return 8
	}
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
