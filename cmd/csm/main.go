// Command csm is the Claude Session Monitor — a terminal TUI for monitoring
// all active Claude Code sessions and their subagents.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/MashellHan/claude-session-monitor/internal/data"
	"github.com/MashellHan/claude-session-monitor/internal/model"
	"github.com/MashellHan/claude-session-monitor/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time.
var version = "1.0.0"

func main() {
	// CLI flags.
	versionFlag := flag.Bool("version", false, "Print version and exit")
	claudeDirFlag := flag.String("claude-dir", "", "Override Claude data directory (default: ~/.claude)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("csm v%s\n", version)
		os.Exit(0)
	}

	// Resolve Claude data directory.
	claudeDir := *claudeDirFlag
	if claudeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		claudeDir = filepath.Join(home, ".claude")
	}

	// Check if Claude directory exists — warn but don't fail.
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: %s does not exist. No sessions will be shown.\n", claudeDir)
	}

	// Initialize scanner.
	scanner := data.NewScanner(claudeDir)

	// Perform initial scan.
	st := store.New()
	result, err := scanner.FullScan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: initial scan error: %v\n", err)
		// Continue anyway — we'll get data on the next tick.
	} else {
		st.LoadScanResult(result)
	}

	// Initialize filesystem watcher.
	watcher, err := data.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create file watcher: %v\n", err)
		// Continue without watcher — polling fallback will still work.
	} else {
		defer watcher.Close()

		// Watch the key directories.
		watchPaths := scanner.WatchPaths()
		for _, p := range watchPaths {
			if err := watcher.WatchDir(p); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot watch %s: %v\n", p, err)
			}
		}
	}

	// Create and run the Bubble Tea app.
	app := model.NewApp(st, scanner, watcher, claudeDir)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("error running csm: %v", err)
	}
}
