package chrome_test

import (
	"testing"

	chrome "github.com/masfu/chrome-go"
)

var normalizeKeyTests = []struct {
	input string
	want  string
}{
	{"ctrl", "Control"},
	{"cmd", "Meta"},
	{"command", "Meta"},
	{"alt", "Alt"},
	{"shift", "Shift"},
	{"enter", "Enter"},
	{"return", "Enter"},
	{"tab", "Tab"},
	{"backspace", "Backspace"},
	{"delete", "Delete"},
	{"del", "Delete"},
	{"escape", "Escape"},
	{"esc", "Escape"},
	{"arrowup", "ArrowUp"},
	{"arrowdown", "ArrowDown"},
	{"arrowleft", "ArrowLeft"},
	{"arrowright", "ArrowRight"},
	{"home", "Home"},
	{"end", "End"},
	{"pageup", "PageUp"},
	{"pagedown", "PageDown"},
	{"space", " "},
	// Passthrough for unknown keys.
	{"F5", "F5"},
	{"A", "A"},
	{"1", "1"},
	{"Enter", "Enter"},
}

func TestNormalizeKey(t *testing.T) {
	for _, tt := range normalizeKeyTests {
		t.Run(tt.input, func(t *testing.T) {
			got := chrome.NormalizeKey(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
