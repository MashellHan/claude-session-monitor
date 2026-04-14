package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// FileChangeMsg is a Bubble Tea message sent when a watched file changes.
type FileChangeMsg struct {
	Path string
	Op   fsnotify.Op
}

// WatchErrorMsg is a Bubble Tea message sent when the watcher encounters an error.
type WatchErrorMsg struct {
	Err error
}

// Watcher wraps fsnotify to watch Claude Code data directories and
// send Bubble Tea messages on changes.
type Watcher struct {
	watcher   *fsnotify.Watcher
	events    chan FileChangeMsg
	errors    chan WatchErrorMsg
	done      chan struct{}
	closeOnce sync.Once
}

// NewWatcher creates a new filesystem watcher.
func NewWatcher() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		watcher: fsw,
		events:  make(chan FileChangeMsg, 100),
		errors:  make(chan WatchErrorMsg, 10),
		done:    make(chan struct{}),
	}

	go w.loop()
	return w, nil
}

// loop processes fsnotify events and forwards them.
func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Only forward write and create events — we don't care about
			// chmod or rename for our purposes.
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				select {
				case w.events <- FileChangeMsg{Path: event.Name, Op: event.Op}:
				default:
					// Drop events if channel is full to prevent blocking.
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- WatchErrorMsg{Err: err}:
			default:
			}
		case <-w.done:
			return
		}
	}
}

// WatchDir adds a directory (and its immediate subdirectories) to the watcher.
// Creates the directory if it doesn't exist.
func (w *Watcher) WatchDir(dir string) error {
	// Walk the directory tree (up to 3 levels deep) to add watches.
	return w.walkAndWatch(dir, 0, 3)
}

// walkAndWatch recursively adds directories to the watcher up to maxDepth.
func (w *Watcher) walkAndWatch(dir string, depth, maxDepth int) error {
	if depth > maxDepth {
		return nil
	}

	// Check if directory exists.
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Graceful degradation.
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("watch %s: %w", dir, err)
	}

	// Watch subdirectories.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // Skip unreadable directories.
	}

	for _, e := range entries {
		if e.IsDir() {
			subdir := filepath.Join(dir, e.Name())
			if err := w.walkAndWatch(subdir, depth+1, maxDepth); err != nil {
				continue // Skip errors in subdirectories.
			}
		}
	}

	return nil
}

// Events returns the channel of file change messages.
func (w *Watcher) Events() <-chan FileChangeMsg {
	return w.events
}

// Errors returns the channel of watcher errors.
func (w *Watcher) Errors() <-chan WatchErrorMsg {
	return w.errors
}

// Close shuts down the watcher. Safe to call multiple times.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() { close(w.done) })
	return w.watcher.Close()
}
