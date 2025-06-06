package monitor

import (
	"testing"
)

// mockScreenEventHandler tracks screen clear events
type mockScreenEventHandler struct {
	screenClearCount int
}

func (m *mockScreenEventHandler) HandleScreenClear() {
	m.screenClearCount++
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
