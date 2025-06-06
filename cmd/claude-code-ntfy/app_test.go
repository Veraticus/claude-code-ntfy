package main

import (
	"os"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
)

func TestNewDependencies(t *testing.T) {
	// Test with minimal config
	cfg := &config.Config{
		NtfyTopic:       "test-topic",
		NtfyServer:      "https://ntfy.sh",
		BackstopTimeout: 30 * time.Second,
	}

	deps, err := NewDependencies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deps.Config != cfg {
		t.Error("expected config to be set")
	}

	if deps.Notifier == nil {
		t.Error("expected notifier to be created")
	}

	if deps.OutputMonitor == nil {
		t.Error("expected output monitor to be created")
	}

	if deps.ProcessManager == nil {
		t.Error("expected process manager to be created")
	}

	// Clean up
	deps.Close()
}

func TestDependenciesClose(t *testing.T) {
	cfg := &config.Config{
		NtfyTopic:       "test-topic",
		NtfyServer:      "https://ntfy.sh",
		BackstopTimeout: 30 * time.Second,
	}

	deps, err := NewDependencies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close should not panic
	deps.Close()

	// Multiple closes should be safe
	deps.Close()
}

func TestApplication_Run(t *testing.T) {
	// This test would need a mock process manager
	// For now, we'll just test that the application can be created
	cfg := &config.Config{
		NtfyTopic:       "test-topic",
		NtfyServer:      "https://ntfy.sh",
		BackstopTimeout: 30 * time.Second,
		Quiet:           true, // Quiet mode for testing
	}

	deps, err := NewDependencies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer deps.Close()

	app := NewApplication(deps)
	if app == nil {
		t.Error("expected application to be created")
	}

	// Test ExitCode default
	if code := app.ExitCode(); code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestIsatty(t *testing.T) {
	// Test isatty function
	// stderr is typically not a tty in test environment
	if isatty(os.Stderr.Fd()) {
		t.Skip("stderr is a tty in test environment")
	}
}
