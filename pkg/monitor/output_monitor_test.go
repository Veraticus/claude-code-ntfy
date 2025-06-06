package monitor

import (
	"sync"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
)

// MockPatternMatcher implements PatternMatcher for testing
type MockPatternMatcher struct {
	matches     []MatchResult
	matchFunc   func(string) []MatchResult
	called      bool
	callCount   int
	calledLines []string
}

func (m *MockPatternMatcher) Match(text string) []MatchResult {
	m.called = true
	m.callCount++
	m.calledLines = append(m.calledLines, text)

	if m.matchFunc != nil {
		return m.matchFunc(text)
	}
	return m.matches
}

// MockIdleDetector implements IdleDetector for testing
type MockIdleDetector struct {
	isIdle       bool
	idleError    error
	lastActivity time.Time
}

func (m *MockIdleDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	if m.idleError != nil {
		return false, m.idleError
	}
	return m.isIdle, nil
}

func (m *MockIdleDetector) LastActivity() time.Time {
	return m.lastActivity
}

// MockNotifier implements Notifier for testing
type MockNotifier struct {
	mu            sync.Mutex
	notifications []notification.Notification
	sendError     error
}

func (m *MockNotifier) Send(notification notification.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *MockNotifier) GetNotifications() []notification.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]notification.Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func TestOutputMonitor_HandleData(t *testing.T) {
	tests := []struct {
		name          string
		data          [][]byte
		matches       []MatchResult
		config        *config.Config
		isIdle        bool
		wantNotifs    int
		wantProcessed bool
	}{
		{
			name: "single line with match",
			data: [][]byte{[]byte("Error occurred\n")},
			matches: []MatchResult{
				{PatternName: "error", Text: "Error", Position: 0},
			},
			config:        &config.Config{IdleTimeout: 2 * time.Minute},
			isIdle:        true,
			wantNotifs:    1,
			wantProcessed: true,
		},
		{
			name: "multiple lines",
			data: [][]byte{
				[]byte("Line 1\n"),
				[]byte("Error line\n"),
				[]byte("Line 3\n"),
			},
			matches:       nil, // Will use matchFunc instead
			config:        &config.Config{IdleTimeout: 2 * time.Minute},
			isIdle:        true,
			wantNotifs:    1,
			wantProcessed: true,
		},
		{
			name: "incomplete line buffering",
			data: [][]byte{
				[]byte("Partial "),
				[]byte("line with "),
				[]byte("Error\n"),
			},
			matches: []MatchResult{
				{PatternName: "error", Text: "Error", Position: 15},
			},
			config:        &config.Config{IdleTimeout: 2 * time.Minute},
			isIdle:        true,
			wantNotifs:    1,
			wantProcessed: true,
		},
		{
			name:          "quiet mode - no notifications",
			data:          [][]byte{[]byte("Error occurred\n")},
			matches:       []MatchResult{{PatternName: "error", Text: "Error", Position: 0}},
			config:        &config.Config{Quiet: true},
			isIdle:        true,
			wantNotifs:    0,
			wantProcessed: false, // Pattern matcher should not be called in quiet mode
		},
		{
			name:          "user active - no notifications",
			data:          [][]byte{[]byte("Error occurred\n")},
			matches:       []MatchResult{{PatternName: "error", Text: "Error", Position: 0}},
			config:        &config.Config{IdleTimeout: 2 * time.Minute},
			isIdle:        false,
			wantNotifs:    0,
			wantProcessed: true,
		},
		{
			name:          "force notify - ignore idle",
			data:          [][]byte{[]byte("Error occurred\n")},
			matches:       []MatchResult{{PatternName: "error", Text: "Error", Position: 0}},
			config:        &config.Config{ForceNotify: true},
			isIdle:        false,
			wantNotifs:    1,
			wantProcessed: true,
		},
		{
			name:          "no matches",
			data:          [][]byte{[]byte("Normal line\n")},
			matches:       []MatchResult{},
			config:        &config.Config{},
			isIdle:        true,
			wantNotifs:    0,
			wantProcessed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMatcher := &MockPatternMatcher{matches: tt.matches}

			// Set up match function for specific test cases
			switch tt.name {
			case "multiple lines":
				mockMatcher.matchFunc = func(line string) []MatchResult {
					if line == "Error line" {
						return []MatchResult{
							{PatternName: "error", Text: "Error", Position: 0},
						}
					}
					return nil
				}
			case "incomplete line buffering":
				mockMatcher.matchFunc = func(line string) []MatchResult {
					if line == "Partial line with Error" {
						return []MatchResult{
							{PatternName: "error", Text: "Error", Position: 18},
						}
					}
					return nil
				}
			}

			mockIdle := &MockIdleDetector{isIdle: tt.isIdle}
			mockNotifier := &MockNotifier{}

			monitor := NewOutputMonitor(tt.config, mockMatcher, mockIdle, mockNotifier)

			// Process data
			for _, data := range tt.data {
				monitor.HandleData(data)
			}

			// Check results
			notifs := mockNotifier.GetNotifications()
			if len(notifs) != tt.wantNotifs {
				t.Errorf("expected %d notifications but got %d", tt.wantNotifs, len(notifs))
			}

			if tt.wantProcessed && !mockMatcher.called {
				t.Error("pattern matcher was not called")
			}
		})
	}
}

// mockScreenEventHandler tracks screen clear events for testing
type mockScreenEventHandlerOM struct {
	mu               sync.Mutex
	screenClearCount int
	titleChanges     []string
	focusInCount     int
	focusOutCount    int
}

func (m *mockScreenEventHandlerOM) HandleScreenClear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.screenClearCount++
}

func (m *mockScreenEventHandlerOM) HandleTitleChange(title string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.titleChanges = append(m.titleChanges, title)
}

func (m *mockScreenEventHandlerOM) HandleFocusIn() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.focusInCount++
}

func (m *mockScreenEventHandlerOM) HandleFocusOut() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.focusOutCount++
}

func TestOutputMonitorScreenEventHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedClears int
	}{
		{
			name:           "screen clear sequence triggers handler",
			input:          []byte("text\033[2Jmore text"),
			expectedClears: 1,
		},
		{
			name:           "multiple clear sequences",
			input:          []byte("\033[2J\033[3J\033[H"),
			expectedClears: 1, // Only triggers once per batch
		},
		{
			name:           "no clear sequences",
			input:          []byte("normal output\nwith newlines\n"),
			expectedClears: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Patterns: []config.Pattern{{Regex: "test", Name: "test"}},
			}

			matcher := &MockPatternMatcher{}
			idleDetector := &MockIdleDetector{isIdle: true}
			notifier := &MockNotifier{}

			monitor := NewOutputMonitor(cfg, matcher, idleDetector, notifier)

			// Set up screen event handler
			handler := &mockScreenEventHandlerOM{}
			monitor.SetScreenEventHandler(handler)

			// Process the input
			monitor.HandleData(tt.input)

			// Check screen clear count
			handler.mu.Lock()
			clearCount := handler.screenClearCount
			handler.mu.Unlock()

			if clearCount != tt.expectedClears {
				t.Errorf("expected %d screen clears, got %d", tt.expectedClears, clearCount)
			}
		})
	}
}

func TestOutputMonitorSetScreenEventHandler(t *testing.T) {
	cfg := &config.Config{}
	monitor := NewOutputMonitor(cfg, nil, nil, nil)

	// Initially set to self
	if monitor.screenEventHandler != monitor {
		t.Error("expected screen event handler to be set to self initially")
	}

	// Set handler
	handler := &mockScreenEventHandlerOM{}
	monitor.SetScreenEventHandler(handler)

	// Verify it was set
	monitor.mu.Lock()
	hasHandler := monitor.screenEventHandler != nil
	monitor.mu.Unlock()

	if !hasHandler {
		t.Error("expected screen event handler to be set")
	}
}

func TestOutputMonitor_HandleLine(t *testing.T) {
	mockMatcher := &MockPatternMatcher{
		matches: []MatchResult{
			{PatternName: "test", Text: "test", Position: 0},
		},
	}
	mockIdle := &MockIdleDetector{isIdle: true}
	mockNotifier := &MockNotifier{}

	config := &config.Config{IdleTimeout: 2 * time.Minute}
	monitor := NewOutputMonitor(config, mockMatcher, mockIdle, mockNotifier)

	// Test HandleLine
	monitor.HandleLine("test line")

	notifs := mockNotifier.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("expected 1 notification but got %d", len(notifs))
	}

	// Check last output time was updated
	if time.Since(monitor.GetLastOutputTime()) > time.Second {
		t.Error("last output time was not updated")
	}
}

func TestOutputMonitor_Flush(t *testing.T) {
	mockMatcher := &MockPatternMatcher{
		matchFunc: func(line string) []MatchResult {
			if line == "incomplete line without newline" {
				return []MatchResult{
					{PatternName: "test", Text: "test", Position: 0},
				}
			}
			return nil
		},
	}
	mockIdle := &MockIdleDetector{isIdle: true}
	mockNotifier := &MockNotifier{}

	config := &config.Config{IdleTimeout: 2 * time.Minute}
	monitor := NewOutputMonitor(config, mockMatcher, mockIdle, mockNotifier)

	// Add incomplete line
	monitor.HandleData([]byte("incomplete line without newline"))

	// Should have no notifications yet
	if len(mockNotifier.GetNotifications()) != 0 {
		t.Error("notification sent before flush")
	}

	// Flush should process the line
	monitor.Flush()

	notifs := mockNotifier.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("expected 1 notification after flush but got %d", len(notifs))
	}
}

func TestOutputMonitor_MultipleMatches(t *testing.T) {
	mockMatcher := &MockPatternMatcher{
		matches: []MatchResult{
			{PatternName: "error", Text: "ERROR", Position: 0},
			{PatternName: "warning", Text: "WARNING", Position: 10},
		},
	}
	mockIdle := &MockIdleDetector{isIdle: true}
	mockNotifier := &MockNotifier{}

	config := &config.Config{IdleTimeout: 2 * time.Minute}
	monitor := NewOutputMonitor(config, mockMatcher, mockIdle, mockNotifier)

	monitor.HandleData([]byte("ERROR and WARNING in same line\n"))

	notifs := mockNotifier.GetNotifications()
	if len(notifs) != 2 {
		t.Errorf("expected 2 notifications but got %d", len(notifs))
	}

	// Check notification content
	patterns := make(map[string]bool)
	for _, n := range notifs {
		patterns[n.Pattern] = true
		if n.Message != "ERROR and WARNING in same line" {
			t.Errorf("unexpected message: %s", n.Message)
		}
	}

	if !patterns["error"] || !patterns["warning"] {
		t.Error("not all patterns were notified")
	}
}

func TestOutputMonitor_NilNotifier(t *testing.T) {
	mockMatcher := &MockPatternMatcher{
		matches: []MatchResult{
			{PatternName: "test", Text: "test", Position: 0},
		},
	}
	mockIdle := &MockIdleDetector{isIdle: true}

	config := &config.Config{IdleTimeout: 2 * time.Minute}
	monitor := NewOutputMonitor(config, mockMatcher, mockIdle, nil)

	// Should not panic with nil notifier
	monitor.HandleData([]byte("test line\n"))
}

func TestOutputMonitor_NilIdleDetector(t *testing.T) {
	mockMatcher := &MockPatternMatcher{
		matches: []MatchResult{
			{PatternName: "test", Text: "test", Position: 0},
		},
	}
	mockNotifier := &MockNotifier{}

	config := &config.Config{IdleTimeout: 2 * time.Minute}
	monitor := NewOutputMonitor(config, mockMatcher, nil, mockNotifier)

	// Should still send notifications when idle detector is nil
	monitor.HandleData([]byte("test line\n"))

	notifs := mockNotifier.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("expected 1 notification but got %d", len(notifs))
	}
}

func TestOutputMonitor_LineBuffering(t *testing.T) {
	tests := []struct {
		name            string
		inputs          [][]byte
		wantBufferEmpty bool
	}{
		{
			name:            "single complete line",
			inputs:          [][]byte{[]byte("line1\n")},
			wantBufferEmpty: true,
		},
		{
			name:            "incomplete line",
			inputs:          [][]byte{[]byte("incomplete")},
			wantBufferEmpty: false,
		},
		{
			name:            "incomplete then complete",
			inputs:          [][]byte{[]byte("part1 "), []byte("part2\n")},
			wantBufferEmpty: true,
		},
		{
			name:            "multiple lines with remainder",
			inputs:          [][]byte{[]byte("line1\nline2\npart")},
			wantBufferEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMatcher := &MockPatternMatcher{}
			config := &config.Config{}
			monitor := NewOutputMonitor(config, mockMatcher, nil, nil)

			for _, input := range tt.inputs {
				monitor.HandleData(input)
			}

			// Check buffer state
			hasData := monitor.lineBuffer.Len() > 0
			if tt.wantBufferEmpty && hasData {
				t.Errorf("expected empty buffer but has %d bytes", monitor.lineBuffer.Len())
			}
			if !tt.wantBufferEmpty && !hasData {
				t.Error("expected data in buffer but it's empty")
			}
		})
	}
}

func TestOutputMonitorTerminalTitleInNotifications(t *testing.T) {
	cfg := &config.Config{
		ForceNotify: true, // Always notify for this test
	}

	matcher := &MockPatternMatcher{
		matches: []MatchResult{
			{PatternName: "test", Text: "matched", Position: 0},
		},
	}

	notifier := &MockNotifier{}
	monitor := NewOutputMonitor(cfg, matcher, nil, notifier)

	// Send terminal title change
	monitor.HandleData([]byte("\033]0;My Task Title\007"))

	// Send a line that matches
	monitor.HandleData([]byte("matched line\n"))

	// Check notification
	notifs := notifier.GetNotifications()
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}

	// Verify title is simple (context is added by ContextNotifier now)
	expectedTitle := "Claude Code: test"
	if notifs[0].Title != expectedTitle {
		t.Errorf("expected title %q, got %q", expectedTitle, notifs[0].Title)
	}
}

func TestOutputMonitorFocusBasedNotificationSuppression(t *testing.T) {
	tests := []struct {
		name           string
		focusReporting bool
		isFocused      bool
		forceNotify    bool
		expectNotify   bool
	}{
		{
			name:           "focused terminal suppresses notifications",
			focusReporting: true,
			isFocused:      true,
			forceNotify:    false,
			expectNotify:   false,
		},
		{
			name:           "unfocused terminal allows notifications",
			focusReporting: true,
			isFocused:      false,
			forceNotify:    false,
			expectNotify:   true,
		},
		{
			name:           "force notify overrides focus",
			focusReporting: true,
			isFocused:      true,
			forceNotify:    true,
			expectNotify:   true,
		},
		{
			name:           "focus reporting disabled ignores focus state",
			focusReporting: false,
			isFocused:      true,
			forceNotify:    false,
			expectNotify:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ForceNotify: tt.forceNotify,
			}

			matcher := &MockPatternMatcher{
				matches: []MatchResult{
					{PatternName: "test", Text: "matched", Position: 0},
				},
			}

			notifier := &MockNotifier{}
			monitor := NewOutputMonitor(cfg, matcher, nil, notifier)

			// Set focus state
			monitor.SetFocusReportingEnabled(tt.focusReporting)
			if tt.isFocused {
				monitor.HandleFocusIn()
			} else {
				monitor.HandleFocusOut()
			}

			// Send a matching line
			monitor.HandleData([]byte("matched line\n"))

			// Check notifications
			notifs := notifier.GetNotifications()
			if tt.expectNotify {
				if len(notifs) != 1 {
					t.Errorf("expected 1 notification, got %d", len(notifs))
				}
			} else {
				if len(notifs) != 0 {
					t.Errorf("expected no notifications, got %d", len(notifs))
				}
			}
		})
	}
}

// Note: Testing idle state updates in status indicator is not included here
// because it depends on a type assertion to *status.Indicator which is an
// implementation detail. The functionality is tested via integration tests.
