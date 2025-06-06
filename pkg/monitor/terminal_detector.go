package monitor

import (
	"bytes"
	"regexp"

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

// Sequences that might interfere with status line display
var statusInterferingSequences = [][]byte{
	[]byte("\033[r"),      // Reset scrolling region (might affect bottom line)
	[]byte("\033[?47h"),   // Switch to alternate screen buffer
	[]byte("\033[?1047h"), // Save cursor and switch to alternate screen
	[]byte("\033[?1049h"), // Save cursor and switch to alternate screen (xterm)
	[]byte("\033[?47l"),   // Switch back from alternate screen
	[]byte("\033[?1047l"), // Restore cursor and switch from alternate screen
	[]byte("\033[?1049l"), // Restore cursor and switch from alternate screen (xterm)
	[]byte("\033D"),       // Index (scroll down)
	[]byte("\033M"),       // Reverse index (scroll up)
	[]byte("\033[S"),      // Scroll up (might affect bottom line)
	[]byte("\033[T"),      // Scroll down (might affect bottom line)
}

// Focus event sequences
var (
	focusInSequence  = []byte("\033[I")
	focusOutSequence = []byte("\033[O")
	// Enable focus reporting: \033[?1004h
	// Disable focus reporting: \033[?1004l
)

// OSC terminal title sequence pattern
// Matches: ESC]0;title BEL or ESC]0;title ESC\
// Also matches ESC]1; and ESC]2; variants
var titlePattern = regexp.MustCompile(`\033\](?:0|1|2);([^\007\033]*?)(?:\007|\033\\)`)

// TerminalSequenceDetector detects terminal escape sequences in output
type TerminalSequenceDetector struct {
	// Buffer to handle sequences that might be split across data chunks
	buffer []byte
	// Track if we've enabled focus reporting
	focusReportingEnabled bool
}

// NewTerminalSequenceDetector creates a new terminal sequence detector
func NewTerminalSequenceDetector() interfaces.TerminalSequenceDetector {
	return &TerminalSequenceDetector{
		buffer:                make([]byte, 0, 1024), // Larger buffer for OSC sequences
		focusReportingEnabled: false,
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
	foundClear := false
	for _, seq := range screenClearSequences {
		if bytes.Contains(t.buffer, seq) {
			foundClear = true
			break
		}
	}

	// Also check for sequences that interfere with status display
	if !foundClear {
		for _, seq := range statusInterferingSequences {
			if bytes.Contains(t.buffer, seq) {
				foundClear = true
				break
			}
		}
	}

	// Check for cursor positioning that might affect bottom line
	if !foundClear && t.detectBottomLineClear(t.buffer) {
		foundClear = true
	}

	if foundClear {
		handler.HandleScreenClear()
	}

	// Look for focus events
	if bytes.Contains(t.buffer, focusInSequence) {
		handler.HandleFocusIn()
	}
	if bytes.Contains(t.buffer, focusOutSequence) {
		handler.HandleFocusOut()
	}

	// Look for terminal title changes
	if matches := titlePattern.FindAllSubmatch(t.buffer, -1); matches != nil {
		// Get the last title change (most recent)
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) > 1 {
			title := string(lastMatch[1])
			handler.HandleTitleChange(title)
		}
	}

	// Keep buffer reasonable size - OSC sequences can be longer than regular escape sequences
	// Title sequences can be up to ~200 chars, so keep a larger buffer
	maxBufferSize := 512
	if len(t.buffer) > maxBufferSize {
		// Keep the last portion that might contain incomplete sequences
		t.buffer = t.buffer[len(t.buffer)-maxBufferSize:]
	}
}

// EnableFocusReporting returns the escape sequence to enable focus reporting
func EnableFocusReporting() []byte {
	return []byte("\033[?1004h")
}

// DisableFocusReporting returns the escape sequence to disable focus reporting
func DisableFocusReporting() []byte {
	return []byte("\033[?1004l")
}

// detectBottomLineClear checks for sequences that might clear the bottom line
func (t *TerminalSequenceDetector) detectBottomLineClear(data []byte) bool {
	// Check for cursor positioning to bottom line followed by clear
	// Pattern: ESC[<row>;<col>H followed by ESC[K or ESC[2K
	for i := 0; i < len(data)-5; i++ {
		if data[i] == '\033' && data[i+1] == '[' {
			// Look for cursor positioning
			j := i + 2
			for j < len(data) && data[j] != 'H' && data[j] != 'f' {
				j++
			}
			if j < len(data) && (data[j] == 'H' || data[j] == 'f') {
				// Found cursor positioning, check if it's followed by line clear
				for k := j + 1; k < len(data)-2 && k < j+20; k++ {
					if data[k] == '\033' && data[k+1] == '[' &&
						(data[k+2] == 'K' || (k+3 < len(data) && data[k+2] == '2' && data[k+3] == 'K')) {
						return true
					}
				}
			}
		}
	}

	// Check for ED (Erase Display) sequences that affect bottom
	// ESC[0J clears from cursor to end of screen
	if bytes.Contains(data, []byte("\033[0J")) ||
		bytes.Contains(data, []byte("\033[J")) { // Same as ESC[0J
		return true
	}

	return false
}
