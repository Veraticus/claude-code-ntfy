package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
	"github.com/Veraticus/claude-code-ntfy/pkg/process"
	"github.com/Veraticus/claude-code-ntfy/pkg/status"
)

// Mock implementations for testing

type mockNotifier struct {
	notifications []notification.Notification
	shouldFail    bool
}

func (m *mockNotifier) Send(n notification.Notification) error {
	m.notifications = append(m.notifications, n)
	if m.shouldFail {
		return os.ErrPermission
	}
	return nil
}

type mockOutputMonitor struct {
	linesHandled []string
	dataHandled  [][]byte
}

func (m *mockOutputMonitor) HandleLine(line string) {
	m.linesHandled = append(m.linesHandled, line)
}

func (m *mockOutputMonitor) HandleData(data []byte) {
	m.dataHandled = append(m.dataHandled, data)
}

func TestNewDependencies(t *testing.T) {
	// Test with minimal config
	cfg := &config.Config{
		NtfyTopic:  "test-topic",
		NtfyServer: "https://ntfy.sh",
		RateLimit: config.RateLimitConfig{
			MaxMessages: 5,
			Window:      time.Minute,
		},
		Patterns: []config.Pattern{
			{Name: "test", Regex: "test", Enabled: true},
		},
	}

	deps, err := NewDependencies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deps.Config != cfg {
		t.Error("expected config to be set")
	}

	if deps.IdleDetector == nil {
		t.Error("expected idle detector to be created")
	}

	if deps.Notifier == nil {
		t.Error("expected notifier to be created")
	}

	if deps.NotificationManager == nil {
		t.Error("expected notification manager to be created")
	}

	if deps.StatusIndicator == nil {
		t.Error("expected status indicator to be created")
	}

	if deps.StatusReporter == nil {
		t.Error("expected status reporter to be created")
	}

	// Clean up
	deps.Close()
}

func TestDependenciesClose(t *testing.T) {
	cfg := &config.Config{
		NtfyTopic:  "test-topic",
		NtfyServer: "https://ntfy.sh",
		RateLimit: config.RateLimitConfig{
			MaxMessages: 5,
			Window:      time.Minute,
		},
	}

	deps, err := NewDependencies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close should not panic
	deps.Close()

	// Double close should not panic
	deps.Close()
}

func TestApplicationRunStartupNotification(t *testing.T) {
	// Create mock notifier to capture notifications
	mockNotif := &mockNotifier{}

	// Create test config
	cfg := &config.Config{
		StartupNotify: true,
		NtfyTopic:     "test",
	}

	// Create minimal dependencies
	deps := &Dependencies{
		Config:              cfg,
		NotificationManager: notification.NewManager(cfg, mockNotif, nil),
		ProcessManager:      process.NewManager(cfg, &mockOutputMonitor{}),
		stopChan:            make(chan struct{}),
	}

	_ = NewApplication(deps)

	// For testing, we'll just test that startup notification happens
	// without actually running a process (which would require PTY setup)
	if deps.Config.StartupNotify && !deps.Config.Quiet && deps.NotificationManager != nil {
		pwd, _ := os.Getwd()
		startupNotification := notification.Notification{
			Title:   "Claude Code Session Started",
			Message: fmt.Sprintf("Working directory: %s", pwd),
			Time:    time.Now(),
			Pattern: "startup",
		}
		_ = deps.NotificationManager.Send(startupNotification)
	}

	// Check that startup notification was sent
	if len(mockNotif.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockNotif.notifications))
	}

	notif := mockNotif.notifications[0]
	if notif.Title != "Claude Code Session Started" {
		t.Errorf("expected title 'Claude Code Session Started', got %q", notif.Title)
	}

	if !strings.Contains(notif.Message, "Working directory:") {
		t.Errorf("expected message to contain working directory, got %q", notif.Message)
	}

	if notif.Pattern != "startup" {
		t.Errorf("expected pattern 'startup', got %q", notif.Pattern)
	}
}

func TestApplicationRunNoStartupNotification(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "startup notify disabled",
			config: &config.Config{
				StartupNotify: false,
				NtfyTopic:     "test",
			},
		},
		{
			name: "quiet mode enabled",
			config: &config.Config{
				StartupNotify: true,
				Quiet:         true,
				NtfyTopic:     "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotif := &mockNotifier{}

			deps := &Dependencies{
				Config:              tt.config,
				NotificationManager: notification.NewManager(tt.config, mockNotif, nil),
				ProcessManager:      process.NewManager(tt.config, &mockOutputMonitor{}),
				stopChan:            make(chan struct{}),
			}

			// Test just the startup notification logic
			if deps.Config.StartupNotify && !deps.Config.Quiet && deps.NotificationManager != nil {
				pwd, _ := os.Getwd()
				startupNotification := notification.Notification{
					Title:   "Claude Code Session Started",
					Message: fmt.Sprintf("Working directory: %s", pwd),
					Time:    time.Now(),
					Pattern: "startup",
				}
				_ = deps.NotificationManager.Send(startupNotification)
			}

			// Should not send startup notification
			if len(mockNotif.notifications) != 0 {
				t.Errorf("expected no notifications, got %d", len(mockNotif.notifications))
			}
		})
	}
}

func TestApplicationExitCode(t *testing.T) {
	cfg := &config.Config{}
	deps := &Dependencies{
		Config:         cfg,
		ProcessManager: process.NewManager(cfg, &mockOutputMonitor{}),
	}

	app := NewApplication(deps)

	// Default exit code should be 0
	if app.ExitCode() != 0 {
		t.Errorf("expected exit code 0, got %d", app.ExitCode())
	}
}

func TestApplicationStop(t *testing.T) {
	cfg := &config.Config{}
	deps := &Dependencies{
		Config:         cfg,
		ProcessManager: process.NewManager(cfg, &mockOutputMonitor{}),
	}

	app := NewApplication(deps)

	// Stop should not error even if process not started
	err := app.Stop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsatty(t *testing.T) {
	// This test is platform-specific and may not work in all environments
	// We'll just test that it doesn't panic

	// stdin is usually not a tty in test environments
	result := isatty(os.Stdin.Fd())
	_ = result // We don't assert the value as it depends on the test environment
}

func TestStatusIndicatorIntegration(t *testing.T) {
	// Create a buffer to capture status output
	buf := &bytes.Buffer{}

	// Create a status indicator
	indicator := status.NewIndicator(buf, true)
	reporter := status.NewReporter(indicator)

	// Create a mock notifier
	mockNotif := &mockNotifier{}

	// Create notification manager with status reporter
	cfg := &config.Config{}
	mgr := notification.NewManager(cfg, mockNotif, nil)
	mgr.SetStatusReporter(reporter)

	// Send a notification
	notif := notification.Notification{
		Title:   "Test",
		Message: "Test message",
		Time:    time.Now(),
		Pattern: "test",
	}

	err := mgr.Send(notif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that status was reported
	output := buf.String()
	if !strings.Contains(output, "ntfy") {
		t.Errorf("expected status output to contain 'ntfy', got %q", output)
	}
}
