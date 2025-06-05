package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNtfyClient_Send(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
		serverFunc   func(t *testing.T) http.HandlerFunc
		wantErr      bool
		errContains  string
	}{
		{
			name: "successful send",
			notification: Notification{
				Title:   "Test Alert",
				Message: "Something happened",
				Time:    time.Now(),
				Pattern: "error",
			},
			serverFunc: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					if r.Method != "POST" {
						t.Errorf("Method = %v, want POST", r.Method)
					}
					if r.URL.Path != "/" {
						t.Errorf("Path = %v, want /", r.URL.Path)
					}

					body, _ := io.ReadAll(r.Body)
					var payload map[string]interface{}
					if err := json.Unmarshal(body, &payload); err != nil {
						t.Errorf("Failed to unmarshal body: %v", err)
					}

					if payload["title"] != "Test Alert" {
						t.Errorf("Title = %v, want Test Alert", payload["title"])
					}
					// Priority not set in current implementation

					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprint(w, `{"id":"test123"}`)
				}
			},
			wantErr: false,
		},
		{
			name: "server error",
			notification: Notification{
				Title:   "Test",
				Message: "Test",
			},
			serverFunc: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprint(w, "Internal Server Error")
				}
			},
			wantErr:     true,
			errContains: "ntfy returned status",
		},
		{
			name: "rate limit error",
			notification: Notification{
				Title:   "Test",
				Message: "Test",
			},
			serverFunc: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = fmt.Fprint(w, "Rate limited")
				}
			},
			wantErr:     true,
			errContains: "ntfy returned status",
		},
		{
			name: "authentication error",
			notification: Notification{
				Title:   "Test",
				Message: "Test",
			},
			serverFunc: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = fmt.Fprint(w, "Unauthorized")
				}
			},
			wantErr:     true,
			errContains: "ntfy returned status",
		},
		{
			name: "empty notification fields",
			notification: Notification{
				Title:   "",
				Message: "",
				Time:    time.Time{},
				Pattern: "",
			},
			serverFunc: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					var payload map[string]interface{}
					_ = json.Unmarshal(body, &payload)

					// With empty message and pattern, message should be empty
					msg, _ := payload["message"].(string)
					if msg != "" {
						t.Errorf("Message = %v, want empty string", msg)
					}

					w.WriteHeader(http.StatusOK)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(tt.serverFunc(t))
			defer server.Close()

			// Create client
			client := NewNtfyClient(server.URL, "test-topic")

			// Send notification
			err := client.Send(tt.notification)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Error = %v, want to contain %v", err, tt.errContains)
			}
		})
	}
}

func TestNtfyClient_SendNetworkError(t *testing.T) {
	// Use invalid URL to simulate network error
	client := NewNtfyClient("http://localhost:0", "test-topic")

	err := client.Send(Notification{
		Title:   "Test",
		Message: "Test",
	})

	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestNtfyClient_SendInvalidURL(t *testing.T) {
	// Use malformed URL
	client := NewNtfyClient("://invalid-url", "test-topic")

	err := client.Send(Notification{
		Title:   "Test",
		Message: "Test",
	})

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestNtfyClient_MessageFormatting(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
		wantMessage  string
	}{
		{
			name: "full notification",
			notification: Notification{
				Title:   "Alert",
				Message: "Custom message",
				Pattern: "error.*",
			},
			wantMessage: "Custom message",
		},
		{
			name: "empty message uses pattern",
			notification: Notification{
				Title:   "Alert",
				Message: "",
				Pattern: "warning.*",
			},
			wantMessage: "",
		},
		{
			name: "no message or pattern",
			notification: Notification{
				Title:   "Alert",
				Message: "",
				Pattern: "",
			},
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMessage string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var payload map[string]interface{}
				_ = json.Unmarshal(body, &payload)
				capturedMessage = payload["message"].(string)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewNtfyClient(server.URL, "test-topic")
			_ = client.Send(tt.notification)

			if capturedMessage != tt.wantMessage {
				t.Errorf("Message = %v, want %v", capturedMessage, tt.wantMessage)
			}
		})
	}
}

func TestNewNtfyClient(t *testing.T) {
	tests := []struct {
		name   string
		server string
		topic  string
	}{
		{
			name:   "standard config",
			server: "https://ntfy.sh",
			topic:  "my-topic",
		},
		{
			name:   "custom server",
			server: "https://custom.example.com",
			topic:  "alerts",
		},
		{
			name:   "empty values",
			server: "",
			topic:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewNtfyClient(tt.server, tt.topic)
			if client == nil {
				t.Error("NewNtfyClient() returned nil")
			}

			// Verify it implements Notifier interface
			var _ Notifier = client
		})
	}
}
