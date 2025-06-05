package notification

import (
	"testing"
	"time"
)

func TestNotification(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
		wantTitle    string
		wantMessage  string
		wantTime     time.Time
		wantPattern  string
	}{
		{
			name: "basic notification",
			notification: Notification{
				Title:   "Test Title",
				Message: "Test Message",
				Time:    time.Unix(1234567890, 0),
				Pattern: "test pattern",
			},
			wantTitle:   "Test Title",
			wantMessage: "Test Message",
			wantTime:    time.Unix(1234567890, 0),
			wantPattern: "test pattern",
		},
		{
			name: "empty notification",
			notification: Notification{
				Title:   "",
				Message: "",
				Time:    time.Time{},
				Pattern: "",
			},
			wantTitle:   "",
			wantMessage: "",
			wantTime:    time.Time{},
			wantPattern: "",
		},
		{
			name: "notification with special characters",
			notification: Notification{
				Title:   "Title with 特殊字符",
				Message: "Message with\nnewlines\tand\ttabs",
				Time:    time.Unix(1234567890, 0),
				Pattern: `regex\.pattern\d+`,
			},
			wantTitle:   "Title with 特殊字符",
			wantMessage: "Message with\nnewlines\tand\ttabs",
			wantTime:    time.Unix(1234567890, 0),
			wantPattern: `regex\.pattern\d+`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.notification.Title; got != tt.wantTitle {
				t.Errorf("Title = %v, want %v", got, tt.wantTitle)
			}
			if got := tt.notification.Message; got != tt.wantMessage {
				t.Errorf("Message = %v, want %v", got, tt.wantMessage)
			}
			if got := tt.notification.Time; !got.Equal(tt.wantTime) {
				t.Errorf("Time = %v, want %v", got, tt.wantTime)
			}
			if got := tt.notification.Pattern; got != tt.wantPattern {
				t.Errorf("Pattern = %v, want %v", got, tt.wantPattern)
			}
		})
	}
}
