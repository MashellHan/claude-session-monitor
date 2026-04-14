package ui

import (
	"strings"
	"testing"
)

func TestHelpView(t *testing.T) {
	view := HelpView(80)
	if view == "" {
		t.Error("HelpView should not be empty")
	}
	if !strings.Contains(view, "Key Bindings") {
		t.Error("HelpView should contain 'Key Bindings'")
	}
	if !strings.Contains(view, "Quit") {
		t.Error("HelpView should mention Quit")
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Verify key bindings are defined.
	if len(km.Up.Keys()) == 0 {
		t.Error("Up key should have bindings")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("Down key should have bindings")
	}
	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit key should have bindings")
	}
	if len(km.Tab.Keys()) == 0 {
		t.Error("Tab key should have bindings")
	}
	if len(km.Enter.Keys()) == 0 {
		t.Error("Enter key should have bindings")
	}
}
