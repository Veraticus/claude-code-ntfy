package monitor

import (
	"sync"
	"time"
)

// TerminalState tracks the current state of the terminal
type TerminalState struct {
	mu sync.RWMutex

	// Current terminal title
	title string
	// Whether terminal is currently focused
	focused bool
	// Time of last focus change
	lastFocusChange time.Time
	// Whether focus reporting is enabled
	focusReportingEnabled bool
}

// NewTerminalState creates a new terminal state tracker
func NewTerminalState() *TerminalState {
	return &TerminalState{
		focused:         true, // Assume focused by default
		lastFocusChange: time.Now(),
	}
}

// SetTitle updates the terminal title
func (ts *TerminalState) SetTitle(title string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.title = title
}

// GetTitle returns the current terminal title
func (ts *TerminalState) GetTitle() string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.title
}

// SetFocused updates the focus state
func (ts *TerminalState) SetFocused(focused bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.focused != focused {
		ts.focused = focused
		ts.lastFocusChange = time.Now()
	}
}

// IsFocused returns whether the terminal is currently focused
func (ts *TerminalState) IsFocused() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.focused
}

// GetLastFocusChange returns the time of the last focus change
func (ts *TerminalState) GetLastFocusChange() time.Time {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.lastFocusChange
}

// SetFocusReportingEnabled updates whether focus reporting is enabled
func (ts *TerminalState) SetFocusReportingEnabled(enabled bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.focusReportingEnabled = enabled
}

// IsFocusReportingEnabled returns whether focus reporting is enabled
func (ts *TerminalState) IsFocusReportingEnabled() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.focusReportingEnabled
}
