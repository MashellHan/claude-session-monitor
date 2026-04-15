// Package model implements the Bubble Tea models for the CSM TUI.
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/store"
	"github.com/MashellHan/claude-session-monitor/internal/ui"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Panel represents which panel has focus.
type Panel int

const (
	PanelSessions Panel = iota
	PanelAgents
	PanelTasks
)

// tickInterval is the polling fallback interval.
const tickInterval = 2 * time.Second

// TickMsg is sent periodically to trigger a refresh.
type TickMsg time.Time

// RefreshMsg triggers a full data refresh.
type RefreshMsg struct{}

// App is the root Bubble Tea model that manages all panels.
type App struct {
	store       *store.Store
	scanner     *data.Scanner
	watcher     *data.Watcher
	claudeDir   string

	// Sub-models for each panel.
	sessionList SessionList
	agentTable  AgentTable
	taskList    TaskList
	summaryBar  SummaryBar

	// UI state.
	activePanel    Panel
	showHelp       bool
	filterActive   bool // when true, hide dead sessions
	width          int
	height         int
	lastRefresh    time.Time
	ready          bool
}

// NewApp creates the root app model.
func NewApp(st *store.Store, scanner *data.Scanner, watcher *data.Watcher, claudeDir string) App {
	return App{
		store:       st,
		scanner:     scanner,
		watcher:     watcher,
		claudeDir:   claudeDir,
		sessionList: NewSessionList(),
		agentTable:  NewAgentTable(),
		taskList:    NewTaskList(),
		summaryBar:  NewSummaryBar(),
		activePanel: PanelSessions,
		lastRefresh: time.Now(),
	}
}

// ScanCompleteMsg carries the result of an async full scan.
type ScanCompleteMsg struct {
	Result *data.ScanResult
}

// Init implements tea.Model. Starts the tick timer, file watcher, and initial async scan.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		a.listenForFileChanges(),
		a.asyncScan(), // Non-blocking initial data load.
	)
}

// asyncScan runs FullScan in a goroutine and returns the result as a message.
func (a App) asyncScan() tea.Cmd {
	return func() tea.Msg {
		if a.scanner == nil {
			return ScanCompleteMsg{}
		}
		result, err := a.scanner.FullScan()
		if err != nil {
			return ScanCompleteMsg{}
		}
		return ScanCompleteMsg{Result: result}
	}
}

// tickCmd returns a command that sends a TickMsg after the polling interval.
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// listenForFileChanges returns a command that listens for watcher events.
// Uses a timeout to prevent permanently blocking the Bubble Tea event loop
// if the watcher stalls or is closed.
func (a App) listenForFileChanges() tea.Cmd {
	if a.watcher == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case msg := <-a.watcher.Events():
			return msg
		case errMsg := <-a.watcher.Errors():
			_ = errMsg // Log in the future.
			return RefreshMsg{}
		case <-time.After(30 * time.Second):
			// Re-arm the listener even if no events arrive, to prevent
			// the watcher goroutine from blocking forever.
			return RefreshMsg{}
		}
	}
}

// Update implements tea.Model. Handles all messages and key events.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keys := ui.DefaultKeyMap()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.updateLayout()
		return a, a.asyncScan()

	case ScanCompleteMsg:
		a.applyResult(msg.Result)
		return a, nil

	case TickMsg:
		return a, tea.Batch(tickCmd(), a.asyncScan())

	case data.FileChangeMsg:
		return a, tea.Batch(a.listenForFileChanges(), a.asyncScan())

	case RefreshMsg:
		return a, a.asyncScan()

	case tea.KeyMsg:
		// Global keys.
		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, keys.Help):
			a.showHelp = !a.showHelp
			return a, nil

		case key.Matches(msg, keys.Refresh):
			return a, a.asyncScan() // Non-blocking refresh instead of synchronous scan.

		case key.Matches(msg, keys.Filter):
			a.filterActive = !a.filterActive
			return a, a.asyncScan() // Non-blocking refresh.

		case key.Matches(msg, keys.Tab):
			a.activePanel = (a.activePanel + 1) % 3
			return a, nil

		case key.Matches(msg, keys.ShiftTab):
			a.activePanel = (a.activePanel + 2) % 3 // -1 mod 3
			return a, nil

		case key.Matches(msg, keys.Right):
			a.activePanel = (a.activePanel + 1) % 3
			return a, nil

		case key.Matches(msg, keys.Left):
			a.activePanel = (a.activePanel + 2) % 3
			return a, nil

		case key.Matches(msg, keys.Panel1):
			a.activePanel = PanelSessions
			return a, nil

		case key.Matches(msg, keys.Panel2):
			a.activePanel = PanelAgents
			return a, nil

		case key.Matches(msg, keys.Panel3):
			a.activePanel = PanelTasks
			return a, nil

		case key.Matches(msg, keys.Up):
			a.cursorUp()
			a.syncSelectedSession()
			return a, nil

		case key.Matches(msg, keys.Down):
			a.cursorDown()
			a.syncSelectedSession()
			return a, nil

		case key.Matches(msg, keys.Enter):
			switch a.activePanel {
			case PanelSessions:
				a.sessionList.ToggleDetail()
			case PanelAgents:
				a.agentTable.ToggleDetail()
			}
			return a, nil
		}
	}

	return a, nil
}

// cursorUp moves the cursor up in the active panel.
func (a *App) cursorUp() {
	switch a.activePanel {
	case PanelSessions:
		a.sessionList.CursorUp()
	case PanelAgents:
		a.agentTable.CursorUp()
	case PanelTasks:
		a.taskList.CursorUp()
	}
}

// cursorDown moves the cursor down in the active panel.
func (a *App) cursorDown() {
	switch a.activePanel {
	case PanelSessions:
		a.sessionList.CursorDown()
	case PanelAgents:
		a.agentTable.CursorDown()
	case PanelTasks:
		a.taskList.CursorDown()
	}
}

// syncSelectedSession updates the agent table and task list based on
// the currently selected session.
func (a *App) syncSelectedSession() {
	sess, ok := a.sessionList.SelectedSession()
	if !ok {
		a.agentTable.SetAgents(nil)
		a.agentTable.SetSessionContext(0, "")
		a.taskList.SetTasks(nil)
		return
	}
	agents := a.store.AgentsForSession(sess.SessionID)
	a.agentTable.SetAgents(agents)
	a.agentTable.SetSessionContext(sess.PID, sess.Topic)

	tasks := a.store.TasksForSession(sess.SessionID)
	a.taskList.SetTasks(tasks)
}

// applyResult loads scan results into the store and updates all views.
func (a *App) applyResult(result *data.ScanResult) {
	if result == nil {
		return
	}
	a.store.LoadScanResult(result)
	a.lastRefresh = time.Now()

	// Update session list.
	sessions := a.store.Sessions()
	if a.filterActive {
		filtered := make([]data.Session, 0, len(sessions))
		for _, s := range sessions {
			if s.Alive {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}
	a.sessionList.SetSessions(sessions)

	// Populate agent lookup on the session list for enriched rendering.
	a.sessionList.ClearAgents()
	for _, sess := range sessions {
		agents := a.store.AgentsForSession(sess.SessionID)
		if len(agents) > 0 {
			a.sessionList.SetAgentsForSession(sess.SessionID, agents)
		}
	}

	// Update agents and tasks for selected session.
	a.syncSelectedSession()

	// Update summary bar.
	a.summaryBar.SetStats(a.store.Stats())
}

// updateLayout recalculates panel sizes based on terminal dimensions.
func (a *App) updateLayout() {
	// Reserve space for title (2 lines), summary bar (1 line), and borders.
	availHeight := a.height - 4

	// Split available height: sessions 30%, agents 35%, tasks 25%, summary 10%.
	sessionH := availHeight * 30 / 100
	agentH := availHeight * 35 / 100
	taskH := availHeight - sessionH - agentH

	if sessionH < 4 {
		sessionH = 4
	}
	if agentH < 4 {
		agentH = 4
	}
	if taskH < 4 {
		taskH = 4
	}

	a.sessionList.SetSize(a.width, sessionH)
	a.agentTable.SetSize(a.width, agentH)
	a.taskList.SetSize(a.width, taskH)
	a.summaryBar.SetWidth(a.width)
}

// View implements tea.Model. Renders the complete TUI.
func (a App) View() string {
	if !a.ready {
		return "  Initializing..."
	}

	var sections []string

	// Title bar.
	title := ui.TitleStyle.Render("Claude Session Monitor")
	refreshInfo := ui.MutedStyle.Render(
		fmt.Sprintf(" (refreshed %s ago)", formatTimeSince(a.lastRefresh)))
	filterInfo := ""
	if a.filterActive {
		filterInfo = ui.StatusRunningStyle.Render(" [active only]")
	}
	titleLine := title + refreshInfo + filterInfo
	sections = append(sections, titleLine)

	// Sessions panel.
	sessCount := len(a.sessionList.sessions)
	activeCount := 0
	for _, s := range a.sessionList.sessions {
		if s.Alive {
			activeCount++
		}
	}
	sessionTitle := panelTitle("Sessions", fmt.Sprintf("%d active, %d total", activeCount, sessCount),
		a.activePanel == PanelSessions)
	sections = append(sections, sessionTitle)

	// Enrich session rows with agent info — delegated to SessionList.
	sessionView := a.sessionList.View(a.activePanel == PanelSessions)
	sections = append(sections, sessionView)

	// Agents panel.
	sess, hasSelected := a.sessionList.SelectedSession()
	agentTitle := ""
	if hasSelected {
		agents := a.store.AgentsForSession(sess.SessionID)
		topicLabel := sess.Topic
		if topicLabel == "" {
			topicLabel = sess.Project
		}
		agentTitle = panelTitle("Subagents",
			fmt.Sprintf("PID %d: \"%s\" (%d agents)", sess.PID, topicLabel, len(agents)),
			a.activePanel == PanelAgents)
	} else {
		agentTitle = panelTitle("Subagents", "no session selected",
			a.activePanel == PanelAgents)
	}
	sections = append(sections, agentTitle)
	sections = append(sections, a.agentTable.View(a.activePanel == PanelAgents))

	// Tasks panel.
	if hasSelected {
		tasks := a.store.TasksForSession(sess.SessionID)
		taskTitle := panelTitle("Tasks", TaskSummary(tasks),
			a.activePanel == PanelTasks)
		sections = append(sections, taskTitle)
	} else {
		sections = append(sections, panelTitle("Tasks", "no session selected",
			a.activePanel == PanelTasks))
	}
	sections = append(sections, a.taskList.View(a.activePanel == PanelTasks))

	// Summary bar.
	sections = append(sections, a.summaryBar.View())

	content := strings.Join(sections, "\n")

	// Help overlay.
	if a.showHelp {
		helpOverlay := ui.HelpView(a.width)
		// Place help overlay on the right side.
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, "\n"+helpOverlay)
	}

	return content
}

// panelTitle creates a section title with active indicator.
func panelTitle(name, info string, active bool) string {
	label := fmt.Sprintf("%s (%s)", name, info)
	if active {
		return ui.ActivePanelTitleStyle.Render("▸ " + label)
	}
	return ui.PanelTitleStyle.Render("  " + label)
}

// formatTimeSince returns a human-readable "N seconds/minutes ago" string.
func formatTimeSince(t time.Time) string {
	d := time.Since(t)
	if d < time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
