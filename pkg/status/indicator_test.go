package status

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewIndicator(t *testing.T) {
	buf := &bytes.Buffer{}
	indicator := NewIndicator(buf, true)
	
	if indicator.status != StatusIdle {
		t.Errorf("expected initial status to be StatusIdle, got %v", indicator.status)
	}
	
	if indicator.writer != buf {
		t.Errorf("expected writer to be set")
	}
	
	if !indicator.enabled {
		t.Errorf("expected indicator to be enabled")
	}
}

func TestIndicatorSetStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         Status
		expectedOutput string
		enabled        bool
	}{
		{
			name:           "sending status",
			status:         StatusSending,
			expectedOutput: "⟳ ntfy",
			enabled:        true,
		},
		{
			name:           "success status",
			status:         StatusSuccess,
			expectedOutput: "✓ ntfy",
			enabled:        true,
		},
		{
			name:           "failed status",
			status:         StatusFailed,
			expectedOutput: "✗ ntfy",
			enabled:        true,
		},
		{
			name:           "idle status shows nothing",
			status:         StatusIdle,
			expectedOutput: "",
			enabled:        true,
		},
		{
			name:           "disabled indicator shows nothing",
			status:         StatusSuccess,
			expectedOutput: "",
			enabled:        false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			indicator := NewIndicator(buf, tt.enabled)
			
			indicator.SetStatus(tt.status)
			
			output := buf.String()
			
			// Check if expected output is in the result (ignoring escape sequences for positioning)
			if tt.expectedOutput != "" {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("expected output to contain %q, got %q", tt.expectedOutput, output)
				}
			} else if tt.enabled && output != "" {
				// For idle status, we still emit escape sequences to clear any previous status
				// This is expected behavior - just check that we don't have any ntfy text
				if strings.Contains(output, "ntfy") {
					t.Errorf("expected no ntfy text for idle status, got %q", output)
				}
			} else if !tt.enabled && output != "" {
				// If disabled, should have no output at all
				t.Errorf("expected no output for disabled indicator, got %q", output)
			}
		})
	}
}

func TestIndicatorSuccessStatusWithTime(t *testing.T) {
	buf := &bytes.Buffer{}
	indicator := NewIndicator(buf, true)
	
	// Set success status
	indicator.SetStatus(StatusSuccess)
	
	// Should show green checkmark initially
	output := buf.String()
	if !strings.Contains(output, "✓ ntfy") {
		t.Errorf("expected output to contain green checkmark, got %q", output)
	}
	
	// Manually set lastSent to 10 seconds ago
	indicator.mu.Lock()
	indicator.lastSent = time.Now().Add(-10 * time.Second)
	indicator.mu.Unlock()
	
	// Clear buffer and redraw
	buf.Reset()
	indicator.mu.Lock()
	_ = indicator.draw()
	indicator.mu.Unlock()
	
	output = buf.String()
	if !strings.Contains(output, "✓ ntfy (10s)") {
		t.Errorf("expected output to contain time indicator, got %q", output)
	}
	
	// Set to 35 seconds ago
	indicator.mu.Lock()
	indicator.lastSent = time.Now().Add(-35 * time.Second)
	indicator.mu.Unlock()
	
	// Clear buffer and redraw
	buf.Reset()
	indicator.mu.Lock()
	_ = indicator.draw()
	indicator.mu.Unlock()
	
	output = buf.String()
	// Should not contain ntfy text after 30 seconds
	if strings.Contains(output, "ntfy") {
		t.Errorf("expected no ntfy text after 30 seconds, got %q", output)
	}
}

func TestIndicatorClear(t *testing.T) {
	buf := &bytes.Buffer{}
	indicator := NewIndicator(buf, true)
	
	// Set a status
	indicator.SetStatus(StatusSuccess)
	
	// Clear the indicator
	buf.Reset()
	err := indicator.Clear()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	output := buf.String()
	// Should contain escape sequences but no visible text
	if strings.Contains(output, "ntfy") {
		t.Errorf("expected cleared output to not contain ntfy text, got %q", output)
	}
	
	// Should contain cursor positioning sequences
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected escape sequences in output, got %q", output)
	}
}

func TestIndicatorAutoRefresh(t *testing.T) {
	// Use a thread-safe wrapper for the buffer
	type safeBuffer struct {
		mu  sync.Mutex
		buf bytes.Buffer
	}
	
	sb := &safeBuffer{}
	
	// Create a writer that locks before writing
	writer := writerFunc(func(p []byte) (n int, err error) {
		sb.mu.Lock()
		defer sb.mu.Unlock()
		return sb.buf.Write(p)
	})
	
	indicator := NewIndicator(writer, true)
	
	// Set success status
	indicator.SetStatus(StatusSuccess)
	
	// Start auto refresh
	stopChan := make(chan struct{})
	indicator.StartAutoRefresh(stopChan)
	
	// Wait a bit for refresh
	time.Sleep(1100 * time.Millisecond)
	
	// Stop refresh
	close(stopChan)
	
	// Give it time to clean up
	time.Sleep(100 * time.Millisecond)
	
	// Get output safely
	sb.mu.Lock()
	output := sb.buf.String()
	sb.mu.Unlock()
	
	// Count escape sequences for cursor save (at least 2 - initial draw and one refresh)
	saveCount := strings.Count(output, "\033[s")
	if saveCount < 2 {
		t.Errorf("expected at least 2 draws, got %d", saveCount)
	}
}

// writerFunc is an adapter to allow functions to implement io.Writer
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}