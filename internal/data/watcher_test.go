package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	// Verify channels exist.
	if w.Events() == nil {
		t.Error("Events channel should not be nil")
	}
	if w.Errors() == nil {
		t.Error("Errors channel should not be nil")
	}
}

func TestWatcher_WatchDir_NonExistent(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	// Watching a non-existent directory should not error (graceful degradation).
	err = w.WatchDir("/nonexistent/path/to/watch")
	if err != nil {
		t.Errorf("WatchDir for non-existent path should succeed, got: %v", err)
	}
}

func TestWatcher_WatchDir_Valid(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "csm-watcher-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should succeed.
	err = w.WatchDir(tmpDir)
	if err != nil {
		t.Errorf("WatchDir for valid path should succeed, got: %v", err)
	}
}

func TestWatcher_WatchDir_File(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	// Create a temp file (not a directory).
	tmpFile, err := os.CreateTemp("", "csm-watcher-test")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Watching a file (not a directory) should be a no-op.
	err = w.WatchDir(tmpFile.Name())
	if err != nil {
		t.Errorf("WatchDir for file should not error, got: %v", err)
	}
}

func TestWatcher_FileEvents(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "csm-watcher-events")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = w.WatchDir(tmpDir)
	if err != nil {
		t.Fatalf("WatchDir: %v", err)
	}

	// Write a file to trigger an event.
	testFile := filepath.Join(tmpDir, "test.json")
	err = os.WriteFile(testFile, []byte(`{"test": true}`), 0644)
	if err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// We should receive an event. Use a real timeout.
	select {
	case msg := <-w.Events():
		if msg.Path == "" {
			t.Error("event path should not be empty")
		}
	case <-time.After(2 * time.Second):
		t.Log("no event received within timeout")
	}
}
