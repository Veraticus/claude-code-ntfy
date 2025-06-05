package notification

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStdoutNotifier_Send(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
		wantOutput   string
		wantContains []string
	}{
		{
			name: "basic notification",
			notification: Notification{
				Title:   "Test Alert",
				Message: "Something happened",
				Time:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Pattern: "error pattern",
			},
			wantContains: []string{
				"[NOTIFICATION] Test Alert: Something happened (Pattern: error pattern)",
			},
		},
		{
			name: "notification with empty fields",
			notification: Notification{
				Title:   "",
				Message: "",
				Time:    time.Time{},
				Pattern: "",
			},
			wantContains: []string{
				"[NOTIFICATION] :  (Pattern: )",
			},
		},
		{
			name: "notification with multiline message",
			notification: Notification{
				Title:   "Multi Alert",
				Message: "Line 1\nLine 2\nLine 3",
				Time:    time.Date(2024, 6, 15, 9, 30, 45, 0, time.UTC),
				Pattern: "multiline",
			},
			wantContains: []string{
				"[NOTIFICATION] Multi Alert: Line 1\nLine 2\nLine 3 (Pattern: multiline)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create notifier and send notification
			notifier := NewStdoutNotifier()
			err := notifier.Send(tt.notification)
			if err != nil {
				t.Errorf("Send() error = %v, want nil", err)
			}

			// Restore stdout and read output
			_ = w.Close()
			os.Stdout = old
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Check that output contains expected strings
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string: %q\nGot output:\n%s", want, output)
				}
			}
		})
	}
}

func TestNewStdoutNotifier(t *testing.T) {
	notifier := NewStdoutNotifier()
	if notifier == nil {
		t.Error("NewStdoutNotifier() returned nil")
	}

	// Verify it implements the Notifier interface
	var _ Notifier = notifier
}
