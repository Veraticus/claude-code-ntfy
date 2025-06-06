package monitor

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
	"github.com/Veraticus/claude-code-ntfy/pkg/status"
)

// OutputMonitor monitors output for patterns and triggers notifications
type OutputMonitor struct {
	config         *config.Config
	patternMatcher PatternMatcher
	idleDetector   interfaces.IdleDetector
	notifier       notification.Notifier

	mu             sync.Mutex
	lastOutputTime time.Time
	lineBuffer     bytes.Buffer
	startTime      time.Time

	// Terminal sequence detection
	sequenceDetector   interfaces.TerminalSequenceDetector
	screenEventHandler interfaces.ScreenEventHandler
	terminalState      *TerminalState
}

// NewOutputMonitor creates a new output monitor
func NewOutputMonitor(cfg *config.Config, pm PatternMatcher, idle interfaces.IdleDetector, notifier notification.Notifier) *OutputMonitor {
	now := time.Now()
	om := &OutputMonitor{
		config:           cfg,
		patternMatcher:   pm,
		idleDetector:     idle,
		notifier:         notifier,
		lastOutputTime:   now,
		startTime:        now,
		sequenceDetector: NewTerminalSequenceDetector(),
		terminalState:    NewTerminalState(),
	}
	// Set self as the screen event handler
	om.screenEventHandler = om
	return om
}

// SetScreenEventHandler sets the handler for screen events
func (om *OutputMonitor) SetScreenEventHandler(handler interfaces.ScreenEventHandler) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.screenEventHandler = handler
}

// HandleData processes raw output data
func (om *OutputMonitor) HandleData(data []byte) {
	// Detect terminal sequences before locking (non-blocking operation)
	if om.sequenceDetector != nil && om.screenEventHandler != nil {
		om.sequenceDetector.DetectSequences(data, om.screenEventHandler)
	}

	om.mu.Lock()
	defer om.mu.Unlock()

	// Update last output time
	om.lastOutputTime = time.Now()

	// Add data to line buffer
	om.lineBuffer.Write(data)

	// Process complete lines
	buffer := om.lineBuffer.Bytes()
	om.lineBuffer.Reset()

	// Find complete lines
	start := 0
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == '\n' {
			line := string(buffer[start:i])
			om.processLine(line)
			start = i + 1
		}
	}

	// Keep any incomplete line in the buffer
	if start < len(buffer) {
		om.lineBuffer.Write(buffer[start:])
	}
}

// processLine processes a single line of output
func (om *OutputMonitor) processLine(line string) {
	// Skip if in quiet mode
	if om.config.Quiet {
		return
	}

	// Skip if we're still in the startup grace period
	if om.config.StartupGracePeriod > 0 && time.Since(om.startTime) < om.config.StartupGracePeriod {
		if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "claude-code-ntfy: skipping line during grace period: %q\n", line)
		}
		return
	}

	// Find matches in the line
	matches := om.patternMatcher.Match(line)
	if len(matches) == 0 {
		return
	}

	// Check if we should notify
	if om.shouldNotify() {
		// Create notifications for each match
		for _, match := range matches {
			// Include terminal title in notification if available
			title := "Claude Code Match: " + match.PatternName
			if termTitle := om.terminalState.GetTitle(); termTitle != "" {
				title = "Claude Code [" + termTitle + "]: " + match.PatternName
			}

			n := notification.Notification{
				Title:   title,
				Message: line,
				Time:    time.Now(),
				Pattern: match.PatternName,
			}

			// Send notification
			if om.notifier != nil {
				// Debug: log what's matching
				if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "claude-code-ntfy: pattern '%s' matched in line: %q\n", match.PatternName, line)
				}

				if err := om.notifier.Send(n); err != nil {
					// Log error but don't stop processing
					// This ensures we continue monitoring even if notifications fail
					fmt.Fprintf(os.Stderr, "claude-code-ntfy: notification error: %v\n", err)
				}
			}
		}
	}
}

// shouldNotify determines if a notification should be sent
func (om *OutputMonitor) shouldNotify() bool {
	// Force notify overrides all checks
	if om.config.ForceNotify {
		return true
	}

	// Check if terminal is focused (if focus reporting is enabled)
	if om.terminalState.IsFocusReportingEnabled() && om.terminalState.IsFocused() {
		// Terminal is focused, don't notify
		return false
	}

	// Check if user is idle
	if om.idleDetector != nil {
		idle, err := om.idleDetector.IsUserIdle(om.config.IdleTimeout)
		if err == nil {
			// Update idle state in status indicator if we're using it as screen event handler
			if indicator, ok := om.screenEventHandler.(*status.Indicator); ok {
				indicator.SetIdleState(idle)
			}

			if !idle {
				// User is active, don't notify
				return false
			}
		}
	}

	return true
}

// Flush processes any remaining data in the buffer
func (om *OutputMonitor) Flush() {
	om.mu.Lock()
	defer om.mu.Unlock()

	// Process any remaining line
	if om.lineBuffer.Len() > 0 {
		line := om.lineBuffer.String()
		om.processLine(line)
		om.lineBuffer.Reset()
	}
}

// HandleLine implements the OutputHandler interface
func (om *OutputMonitor) HandleLine(line string) {
	om.mu.Lock()
	om.lastOutputTime = time.Now()
	om.mu.Unlock()

	om.processLine(line)
}

// GetLastOutputTime returns the last time output was received
func (om *OutputMonitor) GetLastOutputTime() time.Time {
	om.mu.Lock()
	defer om.mu.Unlock()
	return om.lastOutputTime
}

// HandleScreenClear implements ScreenEventHandler
func (om *OutputMonitor) HandleScreenClear() {
	// Existing functionality - no changes needed
}

// HandleTitleChange implements ScreenEventHandler
func (om *OutputMonitor) HandleTitleChange(title string) {
	om.terminalState.SetTitle(title)
	if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: terminal title changed to: %q\n", title)
	}
}

// HandleFocusIn implements ScreenEventHandler
func (om *OutputMonitor) HandleFocusIn() {
	om.terminalState.SetFocused(true)
	if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: terminal gained focus\n")
	}
}

// HandleFocusOut implements ScreenEventHandler
func (om *OutputMonitor) HandleFocusOut() {
	om.terminalState.SetFocused(false)
	if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: terminal lost focus\n")
	}
}

// SetFocusReportingEnabled sets whether focus reporting is enabled
func (om *OutputMonitor) SetFocusReportingEnabled(enabled bool) {
	om.terminalState.SetFocusReportingEnabled(enabled)
}
