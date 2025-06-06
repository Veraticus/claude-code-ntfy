package notification

import (
	"os"
	"path/filepath"
	"testing"
)

// mockNotifier for testing context notifier
type mockNotifier struct {
	notifications []Notification
}

func (m *mockNotifier) Send(n Notification) error {
	m.notifications = append(m.notifications, n)
	return nil
}

func TestContextNotifier_AddsContext(t *testing.T) {
	// Get current directory basename for testing
	cwd, _ := os.Getwd()
	expectedCwd := filepath.Base(cwd)

	tests := []struct {
		name          string
		terminalTitle string
		notification  Notification
		expectedTitle string
	}{
		{
			name:          "with terminal title and Claude icon",
			terminalTitle: "✅ Test Terminal",
			notification: Notification{
				Title:   "Claude Code: Match",
				Pattern: "test",
			},
			expectedTitle: "Claude Code: " + expectedCwd + " - Test Terminal",
		},
		{
			name:          "with terminal title but claude name",
			terminalTitle: "✅ claude",
			notification: Notification{
				Title:   "Claude Code: Match",
				Pattern: "test",
			},
			expectedTitle: "Claude Code: " + expectedCwd,
		},
		{
			name:          "without terminal title",
			terminalTitle: "",
			notification: Notification{
				Title:   "Claude Code: Match",
				Pattern: "test",
			},
			expectedTitle: "Claude Code: " + expectedCwd,
		},
		{
			name:          "with different Claude icon",
			terminalTitle: "🤖 Robot Assistant",
			notification: Notification{
				Title:   "Claude Code: Match",
				Pattern: "test",
			},
			expectedTitle: "Claude Code: " + expectedCwd + " - Robot Assistant",
		},
		{
			name:          "with non-ASCII character",
			terminalTitle: "🔥 Fire Mode",
			notification: Notification{
				Title:   "Claude Code: Match",
				Pattern: "test",
			},
			expectedTitle: "Claude Code: " + expectedCwd + " - Fire Mode",
		},
		{
			name:          "no context preserves original title",
			terminalTitle: "",
			notification: Notification{
				Title:   "Claude Code: Backstop",
				Pattern: "backstop",
			},
			expectedTitle: "Claude Code: " + expectedCwd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNotifier{}

			// Create context notifier with terminal info function
			cn := NewContextNotifier(mock, func() string {
				return tt.terminalTitle
			})

			// Send notification
			err := cn.Send(tt.notification)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}

			// Check that notification was sent with context
			if len(mock.notifications) != 1 {
				t.Fatalf("Expected 1 notification, got %d", len(mock.notifications))
			}

			sent := mock.notifications[0]
			if sent.Title != tt.expectedTitle {
				t.Errorf("Expected title %q, got %q", tt.expectedTitle, sent.Title)
			}
		})
	}
}

func TestContextNotifier_CleanTerminalTitle(t *testing.T) {
	cn := &ContextNotifier{}

	tests := []struct {
		input    string
		expected string
	}{
		{"✅ Test Terminal", "Test Terminal"},
		{"✅Test Terminal", "Test Terminal"},
		{"🤖 Robot Assistant", "Robot Assistant"},
		{"⚡ Lightning Fast", "Lightning Fast"},
		{"✨ Sparkle Mode", "Sparkle Mode"},
		{"🔮 Crystal Ball", "Crystal Ball"},
		{"💫 Dizzy", "Dizzy"},
		{"☁️ Cloud Nine", "Cloud Nine"},
		{"🌟 Star Power", "Star Power"},
		{"No Icon Here", "No Icon Here"},
		{"", ""},
		{"   Spaces   ", "Spaces"},
		{"🔥 Non-listed icon", "Non-listed icon"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cn.cleanTerminalTitle(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTerminalTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
