package monitor

import (
	"testing"
)

// mockScreenEventHandler tracks screen clear events
type mockScreenEventHandler struct {
	screenClearCount int
	titleChanges     []string
	focusInCount     int
	focusOutCount    int
}

func (m *mockScreenEventHandler) HandleScreenClear() {
	m.screenClearCount++
}

func (m *mockScreenEventHandler) HandleTitleChange(title string) {
	m.titleChanges = append(m.titleChanges, title)
}

func (m *mockScreenEventHandler) HandleFocusIn() {
	m.focusInCount++
}

func (m *mockScreenEventHandler) HandleFocusOut() {
	m.focusOutCount++
}

func TestTerminalSequenceDetector(t *testing.T) {
	tests := []struct {
		name           string
		input          [][]byte // Multiple chunks to test buffering
		expectedClears int
	}{
		{
			name:           "single clear screen sequence",
			input:          [][]byte{[]byte("hello\033[2Jworld")},
			expectedClears: 1,
		},
		{
			name:           "multiple clear sequences",
			input:          [][]byte{[]byte("\033[2J\033[3J\033[H")},
			expectedClears: 1, // Only triggers once per batch
		},
		{
			name:           "clear sequence split across chunks",
			input:          [][]byte{[]byte("text\033[2"), []byte("Jmore text")},
			expectedClears: 1,
		},
		{
			name:           "reset terminal sequence",
			input:          [][]byte{[]byte("before\033cafter")},
			expectedClears: 1,
		},
		{
			name:           "no clear sequences",
			input:          [][]byte{[]byte("normal text output")},
			expectedClears: 0,
		},
		{
			name:           "clear with cursor positioning",
			input:          [][]byte{[]byte("\033[2J\033[H")},
			expectedClears: 1, // Only triggers once per batch
		},
		{
			name: "complex sequence split across multiple chunks",
			input: [][]byte{
				[]byte("start\033"),
				[]byte("[2J\033["),
				[]byte("3J\033[H"),
			},
			expectedClears: 2, // Second chunk completes \033[2J, third chunk has \033[3J and \033[H
		},
		{
			name:           "alternate screen buffer switch",
			input:          [][]byte{[]byte("\033[?1049h")},
			expectedClears: 1,
		},
		{
			name:           "scrolling region reset",
			input:          [][]byte{[]byte("\033[r")},
			expectedClears: 1,
		},
		{
			name:           "cursor position then line clear",
			input:          [][]byte{[]byte("\033[25;1H\033[K")},
			expectedClears: 1,
		},
		{
			name:           "clear from cursor to end of screen",
			input:          [][]byte{[]byte("\033[0J")},
			expectedClears: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewTerminalSequenceDetector()
			handler := &mockScreenEventHandler{}

			// Process all input chunks
			for _, chunk := range tt.input {
				detector.DetectSequences(chunk, handler)
			}

			if handler.screenClearCount != tt.expectedClears {
				t.Errorf("expected %d screen clears, got %d", tt.expectedClears, handler.screenClearCount)
			}
		})
	}
}

func TestTerminalSequenceDetectorStatusInterference(t *testing.T) {
	tests := []struct {
		name           string
		input          [][]byte
		expectedClears int
	}{
		{
			name:           "alternate screen buffer",
			input:          [][]byte{[]byte("\033[?47h")},
			expectedClears: 1,
		},
		{
			name:           "scrolling region reset",
			input:          [][]byte{[]byte("\033[r")},
			expectedClears: 1,
		},
		{
			name:           "cursor to bottom and clear",
			input:          [][]byte{[]byte("\033[999;1H\033[K")},
			expectedClears: 1,
		},
		{
			name:           "erase display from cursor",
			input:          [][]byte{[]byte("\033[0J")},
			expectedClears: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewTerminalSequenceDetector()
			handler := &mockScreenEventHandler{}

			for _, chunk := range tt.input {
				detector.DetectSequences(chunk, handler)
			}

			if handler.screenClearCount != tt.expectedClears {
				t.Errorf("expected %d screen clears, got %d", tt.expectedClears, handler.screenClearCount)
			}
		})
	}
}

func TestTerminalSequenceDetectorTitleAndFocus(t *testing.T) {
	tests := []struct {
		name             string
		input            [][]byte
		expectedTitles   []string
		expectedFocusIn  int
		expectedFocusOut int
	}{
		{
			name:           "terminal title change",
			input:          [][]byte{[]byte("\033]0;My Title\007")},
			expectedTitles: []string{"My Title"},
		},
		{
			name:           "terminal title with ST terminator",
			input:          [][]byte{[]byte("\033]2;Another Title\033\\")},
			expectedTitles: []string{"Another Title"},
		},
		{
			name:             "focus in event",
			input:            [][]byte{[]byte("\033[I")},
			expectedFocusIn:  1,
			expectedFocusOut: 0,
		},
		{
			name:             "focus out event",
			input:            [][]byte{[]byte("\033[O")},
			expectedFocusIn:  0,
			expectedFocusOut: 1,
		},
		{
			name:             "mixed events",
			input:            [][]byte{[]byte("\033]0;Test\007\033[I\033[O")},
			expectedTitles:   []string{"Test"},
			expectedFocusIn:  1,
			expectedFocusOut: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewTerminalSequenceDetector()
			handler := &mockScreenEventHandler{}

			for _, chunk := range tt.input {
				detector.DetectSequences(chunk, handler)
			}

			if len(handler.titleChanges) != len(tt.expectedTitles) {
				t.Errorf("expected %d title changes, got %d", len(tt.expectedTitles), len(handler.titleChanges))
			}

			for i, title := range tt.expectedTitles {
				if i < len(handler.titleChanges) && handler.titleChanges[i] != title {
					t.Errorf("expected title %q, got %q", title, handler.titleChanges[i])
				}
			}

			if handler.focusInCount != tt.expectedFocusIn {
				t.Errorf("expected %d focus in events, got %d", tt.expectedFocusIn, handler.focusInCount)
			}

			if handler.focusOutCount != tt.expectedFocusOut {
				t.Errorf("expected %d focus out events, got %d", tt.expectedFocusOut, handler.focusOutCount)
			}
		})
	}
}

func TestTerminalSequenceDetectorNilHandler(t *testing.T) {
	detector := NewTerminalSequenceDetector()

	// Should not panic with nil handler
	detector.DetectSequences([]byte("\033[2J"), nil)
}

func TestTerminalSequenceDetectorBufferManagement(t *testing.T) {
	detector := NewTerminalSequenceDetector()
	handler := &mockScreenEventHandler{}

	// Send a lot of data without clear sequences to test buffer trimming
	for i := 0; i < 100; i++ {
		detector.DetectSequences([]byte("normal text without sequences "), handler)
	}

	// Now send a clear sequence - it should still be detected
	detector.DetectSequences([]byte("\033[2J"), handler)

	if handler.screenClearCount != 1 {
		t.Errorf("expected 1 screen clear after buffer management, got %d", handler.screenClearCount)
	}
}
