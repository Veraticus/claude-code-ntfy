package monitor

import (
	"bytes"
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// OutputMonitor monitors output for patterns and triggers notifications
type OutputMonitor struct {
	config         *config.Config
	patternMatcher interfaces.PatternMatcher
	idleDetector   interfaces.IdleDetector
	notifier       interfaces.Notifier

	mu             sync.Mutex
	lastOutputTime time.Time
	lineBuffer     bytes.Buffer
}

// NewOutputMonitor creates a new output monitor
func NewOutputMonitor(cfg *config.Config, pm interfaces.PatternMatcher, idle interfaces.IdleDetector, notifier interfaces.Notifier) *OutputMonitor {
	return &OutputMonitor{
		config:         cfg,
		patternMatcher: pm,
		idleDetector:   idle,
		notifier:       notifier,
		lastOutputTime: time.Now(),
	}
}

// HandleData processes raw output data
func (om *OutputMonitor) HandleData(data []byte) {
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

	// Find matches in the line
	matches := om.patternMatcher.Match(line)
	if len(matches) == 0 {
		return
	}

	// Check if we should notify
	if om.shouldNotify() {
		// Create notifications for each match
		for _, match := range matches {
			notification := interfaces.Notification{
				Title:   "Claude Code Match: " + match.PatternName,
				Message: line,
				Time:    time.Now(),
				Pattern: match.PatternName,
			}

			// Send notification (error handling done by notifier)
			if om.notifier != nil {
				om.notifier.Send(notification)
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

	// Check if user is idle
	if om.idleDetector != nil {
		idle, err := om.idleDetector.IsUserIdle(om.config.IdleTimeout)
		if err == nil && !idle {
			// User is active, don't notify
			return false
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
