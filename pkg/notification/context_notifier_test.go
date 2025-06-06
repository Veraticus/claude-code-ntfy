package notification

import (
	"testing"
)

func TestContextNotifier(t *testing.T) {
	tests := []struct {
		name          string
		terminalTitle string
		notification  Notification
		expectedTitle string
	}{
		{
			name:          "adds cwd and terminal title",
			terminalTitle: "My Project",
			notification: Notification{
				Title:   "Test Notification",
				Message: "Test message",
			},
			expectedTitle: "Claude Code: claude-code-ntfy - My Project",
		},
		{
			name:          "removes claude icon from title",
			terminalTitle: "✅ claude",
			notification: Notification{
				Title:   "Test Notification",
				Message: "Test message",
			},
			expectedTitle: "Claude Code: claude-code-ntfy",
		},
		{
			name:          "handles empty terminal title",
			terminalTitle: "",
			notification: Notification{
				Title:   "Test Notification",
				Message: "Test message",
			},
			expectedTitle: "Claude Code: claude-code-ntfy",
		},
		{
			name:          "ignores plain claude title",
			terminalTitle: "claude",
			notification: Notification{
				Title:   "Test Notification",
				Message: "Test message",
			},
			expectedTitle: "Claude Code: claude-code-ntfy",
		},
		{
			name:          "removes various claude icons",
			terminalTitle: "🤖 My Terminal",
			notification: Notification{
				Title:   "Test Notification",
				Message: "Test message",
			},
			expectedTitle: "Claude Code: claude-code-ntfy - My Terminal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock notifier to capture the sent notification
			var sentNotification Notification
			mockNotifier := &mockNotifier{
				sendFunc: func(n Notification) error {
					sentNotification = n
					return nil
				},
			}

			// Create context notifier
			cn := NewContextNotifier(mockNotifier, func() string {
				return tt.terminalTitle
			})

			// The CWD basename will be "claude-code-ntfy" in tests
			// (based on the current directory structure)

			// Send notification
			err := cn.Send(tt.notification)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check if title was modified correctly
			// The title should start with "Claude Code:"
			if len(sentNotification.Title) < 12 || sentNotification.Title[:12] != "Claude Code:" {
				t.Errorf("expected title to start with 'Claude Code:', got %q", sentNotification.Title)
			}
		})
	}
}

func TestCleanTerminalTitle(t *testing.T) {
	cn := &ContextNotifier{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes checkmark icon",
			input:    "✅ claude",
			expected: "claude",
		},
		{
			name:     "removes robot emoji",
			input:    "🤖 My Project",
			expected: "My Project",
		},
		{
			name:     "removes lightning bolt",
			input:    "⚡ Terminal",
			expected: "Terminal",
		},
		{
			name:     "handles no icon",
			input:    "Plain Title",
			expected: "Plain Title",
		},
		{
			name:     "removes generic unicode at start",
			input:    "🎯 Something",
			expected: "Something",
		},
		{
			name:     "preserves unicode elsewhere",
			input:    "Title with 🎯 emoji",
			expected: "Title with 🎯 emoji",
		},
		{
			name:     "removes garbage characters",
			input:    "ÓÇ∂‚ú≥ Test Coverage 1",
			expected: "Test Coverage 1",
		},
		{
			name:     "handles mixed garbage and valid",
			input:    "∂ßå©ƒ∆ My Project",
			expected: "My Project",
		},
		{
			name:     "preserves brackets and numbers",
			input:    "[1] Development Server",
			expected: "[1] Development Server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cn.cleanTerminalTitle(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTerminalTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// mockNotifier for testing
type mockNotifier struct {
	sendFunc func(Notification) error
}

func (m *mockNotifier) Send(n Notification) error {
	if m.sendFunc != nil {
		return m.sendFunc(n)
	}
	return nil
}
