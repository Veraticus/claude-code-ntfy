package monitor

import (
	"bytes"

	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// Common ANSI escape sequences for screen clearing
var screenClearSequences = [][]byte{
	[]byte("\033[2J"), // Clear entire screen
	[]byte("\033[3J"), // Clear entire screen and scrollback
	[]byte("\033[H"),  // Move cursor to home position (often follows clear)
	[]byte("\033[0J"), // Clear from cursor to end of screen
	[]byte("\033[1J"), // Clear from cursor to beginning of screen
	[]byte("\033c"),   // Reset terminal
}

// TerminalSequenceDetector detects terminal escape sequences in output
type TerminalSequenceDetector struct {
	// Buffer to handle sequences that might be split across data chunks
	buffer []byte
}

// NewTerminalSequenceDetector creates a new terminal sequence detector
func NewTerminalSequenceDetector() interfaces.TerminalSequenceDetector {
	return &TerminalSequenceDetector{
		buffer: make([]byte, 0, 256),
	}
}

// DetectSequences analyzes data for terminal sequences and calls appropriate handlers
func (t *TerminalSequenceDetector) DetectSequences(data []byte, handler interfaces.ScreenEventHandler) {
	if handler == nil {
		return
	}

	// Append new data to buffer
	t.buffer = append(t.buffer, data...)

	// Look for screen clear sequences
	// We only trigger once per detection batch to avoid redundant redraws
	foundClear := false
	for _, seq := range screenClearSequences {
		if bytes.Contains(t.buffer, seq) {
			foundClear = true
			break
		}
	}

	if foundClear {
		handler.HandleScreenClear()
	}

	// Keep only the last part of the buffer that might contain incomplete sequences
	// The longest sequence we're looking for is about 15 bytes
	if len(t.buffer) > 20 {
		t.buffer = t.buffer[len(t.buffer)-20:]
	}
}
