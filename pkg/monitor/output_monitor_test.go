package monitor

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
)

// MockNotifier implements Notifier for testing
type MockNotifier struct {
	mu       sync.Mutex
	sent     []notification.Notification
	sendErr  error
	activity []time.Time
}

func (m *MockNotifier) Send(n notification.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, n)
	return nil
}

func (m *MockNotifier) MarkActivity() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activity = append(m.activity, time.Now())
}

func (m *MockNotifier) GetSent() []notification.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]notification.Notification, len(m.sent))
	copy(result, m.sent)
	return result
}

func (m *MockNotifier) GetActivityCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activity)
}

// MockBackstopNotifier implements backstop-specific methods
type MockBackstopNotifier struct {
	MockNotifier
	backstopSent     bool
	backstopDisabled bool
	sessionReset     int
}

func (m *MockBackstopNotifier) SetBackstopSent(sent bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backstopSent = sent
}

func (m *MockBackstopNotifier) ResetSession() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionReset++
}

func (m *MockBackstopNotifier) DisableBackstopTimer() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backstopDisabled = true
}

func TestContainsVisibleContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		// Visible content cases
		{
			name:     "regular text",
			data:     []byte("Hello world"),
			expected: true,
		},
		{
			name:     "text with newline",
			data:     []byte("Hello\nworld"),
			expected: true,
		},
		{
			name:     "just newline",
			data:     []byte("\n"),
			expected: true,
		},
		{
			name:     "tab character",
			data:     []byte("\t"),
			expected: true,
		},
		{
			name:     "carriage return",
			data:     []byte("\r"),
			expected: true,
		},
		{
			name:     "unicode text",
			data:     []byte("Hello 世界"),
			expected: true,
		},
		{
			name:     "mixed visible and escape sequences",
			data:     []byte("\x1b[31mRed text\x1b[0m"),
			expected: true,
		},
		// Non-visible content cases
		{
			name:     "just escape sequence",
			data:     []byte("\x1b[31m"),
			expected: false,
		},
		{
			name:     "cursor movement",
			data:     []byte("\x1b[1A"),
			expected: false,
		},
		{
			name:     "screen clear",
			data:     []byte("\x1b[2J"),
			expected: false,
		},
		{
			name:     "terminal title",
			data:     []byte("\x1b]0;Title\x07"),
			expected: false,
		},
		{
			name:     "multiple escape sequences",
			data:     []byte("\x1b[?25l\x1b[?25h"),
			expected: false,
		},
		{
			name:     "CSI sequence",
			data:     []byte("\x9b31m"),
			expected: false,
		},
		{
			name:     "control characters only",
			data:     []byte("\x01\x02\x03"),
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "bell character only",
			data:     []byte("\x07"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsVisibleContent(tt.data)
			if result != tt.expected {
				t.Errorf("containsVisibleContent(%q) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}
}

func TestOutputMonitor_HandleData(t *testing.T) {
	tests := []struct {
		name               string
		data               []byte
		expectActivity     bool
		expectBellDetected bool
	}{
		{
			name:           "regular output marks activity",
			data:           []byte("Hello world\n"),
			expectActivity: true,
		},
		{
			name:               "bell character with text disables backstop",
			data:               []byte("Bell: \x07\n"),
			expectActivity:     true,
			expectBellDetected: true,
		},
		{
			name:           "multiple lines processed",
			data:           []byte("Line 1\nLine 2\nLine 3\n"),
			expectActivity: true,
		},
		{
			name:           "partial line buffered",
			data:           []byte("Partial line without newline"),
			expectActivity: true,
		},
		{
			name:           "escape sequence only - no activity",
			data:           []byte("\x1b[31m\x1b[0m"),
			expectActivity: false,
		},
		{
			name:           "cursor movement only - no activity",
			data:           []byte("\x1b[1A\x1b[1B"),
			expectActivity: false,
		},
		{
			name:           "screen clear only - no activity",
			data:           []byte("\x1b[2J"),
			expectActivity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear debug env
			_ = os.Unsetenv("CLAUDE_NOTIFY_DEBUG")

			cfg := &config.Config{}
			mockNotifier := &MockBackstopNotifier{}
			om := NewOutputMonitor(cfg, mockNotifier)

			// Handle the data
			om.HandleData(tt.data)

			// Check activity was marked
			activityCount := mockNotifier.GetActivityCount()
			if tt.expectActivity && activityCount == 0 {
				t.Error("expected activity to be marked")
			} else if !tt.expectActivity && activityCount > 0 {
				t.Errorf("expected no activity to be marked, but got %d", activityCount)
			}

			// Check bell detection
			if tt.expectBellDetected {
				mockNotifier.mu.Lock()
				backstopSent := mockNotifier.backstopSent
				mockNotifier.mu.Unlock()
				if !backstopSent {
					t.Error("expected backstop to be marked as sent after bell")
				}
			}
		})
	}
}

func TestOutputMonitor_ScreenClear(t *testing.T) {
	cfg := &config.Config{}
	mockNotifier := &MockBackstopNotifier{}
	om := NewOutputMonitor(cfg, mockNotifier)

	// Simulate screen clear
	om.HandleScreenClear()

	// Check session was reset
	mockNotifier.mu.Lock()
	resets := mockNotifier.sessionReset
	mockNotifier.mu.Unlock()

	if resets != 1 {
		t.Errorf("expected 1 session reset, got %d", resets)
	}
}

func TestOutputMonitor_BellDetection(t *testing.T) {
	cfg := &config.Config{}
	mockNotifier := &MockBackstopNotifier{}
	om := NewOutputMonitor(cfg, mockNotifier)

	// Test various bell scenarios
	tests := []struct {
		name       string
		input      string
		expectBell bool
	}{
		{"bell in middle", "test\x07text", true},
		{"bell at start", "\x07text", true},
		{"bell at end", "text\x07", true},
		{"no bell", "regular text", false},
		{"bell across lines", "line1\nbell\x07\nline3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset notifier
			mockNotifier = &MockBackstopNotifier{}
			om.SetNotifier(mockNotifier)

			// Process the input
			om.HandleData([]byte(tt.input + "\n"))

			// Check if bell was detected
			mockNotifier.mu.Lock()
			backstopSent := mockNotifier.backstopSent
			mockNotifier.mu.Unlock()

			if tt.expectBell && !backstopSent {
				t.Error("expected bell to be detected")
			} else if !tt.expectBell && backstopSent {
				t.Error("bell detected when not expected")
			}
		})
	}
}

func TestOutputMonitor_LastOutputTime(t *testing.T) {
	cfg := &config.Config{}
	mockNotifier := &MockNotifier{}
	om := NewOutputMonitor(cfg, mockNotifier)

	// Get initial time
	initialTime := om.GetLastOutputTime()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Handle some data
	om.HandleData([]byte("test output\n"))

	// Check time was updated
	newTime := om.GetLastOutputTime()
	if !newTime.After(initialTime) {
		t.Error("expected last output time to be updated")
	}
}

func TestOutputMonitor_FlushPartialLine(t *testing.T) {
	cfg := &config.Config{}
	mockNotifier := &MockBackstopNotifier{}
	om := NewOutputMonitor(cfg, mockNotifier)

	// Send partial line with bell
	om.HandleData([]byte("partial with bell\x07"))

	// Initially bell shouldn't be detected (no newline)
	mockNotifier.mu.Lock()
	backstopSent := mockNotifier.backstopSent
	mockNotifier.mu.Unlock()
	if backstopSent {
		t.Error("bell should not be detected until line is complete")
	}

	// Flush should process the partial line
	om.Flush()

	// Now bell should be detected
	mockNotifier.mu.Lock()
	backstopSent = mockNotifier.backstopSent
	mockNotifier.mu.Unlock()
	if !backstopSent {
		t.Error("bell should be detected after flush")
	}
}
