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
)

// OutputMonitor monitors output and tracks activity
type OutputMonitor struct {
	config   *config.Config
	notifier notification.Notifier

	mu             sync.Mutex
	lastOutputTime time.Time
	lineBuffer     bytes.Buffer

	// Terminal sequence detection
	sequenceDetector   interfaces.TerminalSequenceDetector
	screenEventHandler interfaces.ScreenEventHandler
	terminalState      *TerminalState
}

// NewOutputMonitor creates a new output monitor
func NewOutputMonitor(cfg *config.Config, notifier notification.Notifier) *OutputMonitor {
	now := time.Now()
	om := &OutputMonitor{
		config:           cfg,
		notifier:         notifier,
		lastOutputTime:   now,
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

// SetNotifier sets the notifier
func (om *OutputMonitor) SetNotifier(notifier notification.Notifier) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.notifier = notifier
}

// containsVisibleContent checks if the data contains any visible characters
// Visible characters include printable ASCII, newlines, tabs, and Unicode text
// Returns false for data containing only ANSI escape sequences or control characters
func containsVisibleContent(data []byte) bool {
	i := 0
	for i < len(data) {
		b := data[i]

		// Skip ANSI escape sequences
		if b == 0x1B { // ESC
			i++
			if i < len(data) {
				next := data[i]
				if next == '[' { // CSI sequence
					i++
					// Skip until we find the terminator (0x40-0x7E)
					for i < len(data) {
						c := data[i]
						i++
						if c >= 0x40 && c <= 0x7E {
							break
						}
					}
					continue
				} else if next == ']' { // OSC sequence
					i++
					// Skip until we find BEL or ST terminator
					for i < len(data) {
						c := data[i]
						i++
						if c == 0x07 { // BEL
							break
						}
						// Check for ST (ESC \)
						if c == 0x1B && i < len(data) && data[i] == '\\' {
							i++
							break
						}
					}
					continue
				} else if next == '(' || next == ')' { // Character set sequences
					i++
					if i < len(data) {
						i++ // Skip the character set designation
					}
					continue
				}
			}
		} else if b == 0x9B { // CSI (8-bit)
			i++
			// Skip until we find the terminator (0x40-0x7E)
			for i < len(data) {
				c := data[i]
				i++
				if c >= 0x40 && c <= 0x7E {
					break
				}
			}
			continue
		}

		// Check for visible characters
		if b == '\n' || b == '\r' || b == '\t' { // Newline, carriage return, tab
			return true
		}
		if b >= 32 && b <= 126 { // Printable ASCII
			return true
		}
		if b >= 128 { // Extended ASCII/Unicode (simplified check)
			return true
		}

		i++
	}

	return false
}

// HandleData processes raw output data
func (om *OutputMonitor) HandleData(data []byte) {
	// Detect terminal sequences before locking (non-blocking operation)
	if om.sequenceDetector != nil && om.screenEventHandler != nil {
		om.sequenceDetector.DetectSequences(data, om.screenEventHandler)
	}

	om.mu.Lock()
	defer om.mu.Unlock()

	// Always update last output time when we receive data
	om.lastOutputTime = time.Now()

	// Mark activity for backstop timer only if visible content is detected
	if containsVisibleContent(data) {
		if marker, ok := om.notifier.(notification.ActivityMarker); ok {
			marker.MarkActivity()
		}
	}

	// Add data to line buffer for processing
	om.lineBuffer.Write(data)

	// Process complete lines for bell detection
	buffer := om.lineBuffer.Bytes()
	om.lineBuffer.Reset()

	// Find complete lines
	start := 0
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == '\n' {
			line := buffer[start:i]
			om.processLine(line)
			start = i + 1
		}
	}

	// Keep any incomplete line in the buffer
	if start < len(buffer) {
		om.lineBuffer.Write(buffer[start:])
	}
}

// processLine checks for bell character
func (om *OutputMonitor) processLine(line []byte) {
	// Check for bell character
	if bytes.Contains(line, []byte{0x07}) {
		// Bell detected, disable backstop timer
		if backstopSetter, ok := om.notifier.(interface{ SetBackstopSent(bool) }); ok {
			backstopSetter.SetBackstopSent(true)
			if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "claude-code-ntfy: bell detected, disabling backstop timer\n")
			}
		}
	}
}

// Flush processes any remaining data in the buffer
func (om *OutputMonitor) Flush() {
	om.mu.Lock()
	defer om.mu.Unlock()

	// Process any remaining line
	if om.lineBuffer.Len() > 0 {
		line := om.lineBuffer.Bytes()
		om.processLine(line)
		om.lineBuffer.Reset()
	}
}

// HandleLine implements the OutputHandler interface
func (om *OutputMonitor) HandleLine(line string) {
	om.HandleData([]byte(line + "\n"))
}

// GetLastOutputTime returns the last time output was received
func (om *OutputMonitor) GetLastOutputTime() time.Time {
	om.mu.Lock()
	defer om.mu.Unlock()
	return om.lastOutputTime
}

// HandleScreenClear implements ScreenEventHandler
func (om *OutputMonitor) HandleScreenClear() {
	// Reset backstop notifier session on screen clear (indicates new prompt)
	if resetter, ok := om.notifier.(interface{ ResetSession() }); ok {
		resetter.ResetSession()
	}

	if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: screen cleared - resetting session\n")
	}
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

// LastOutputTime returns the time of the last output
func (om *OutputMonitor) LastOutputTime() time.Time {
	om.mu.Lock()
	defer om.mu.Unlock()
	return om.lastOutputTime
}

// GetTerminalTitle returns the current terminal title
func (om *OutputMonitor) GetTerminalTitle() string {
	if om.terminalState != nil {
		return om.terminalState.GetTitle()
	}
	return ""
}
