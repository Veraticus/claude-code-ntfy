package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
	"github.com/Veraticus/claude-code-ntfy/pkg/monitor"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
	"github.com/Veraticus/claude-code-ntfy/pkg/process"
)

// Dependencies holds all the dependencies for the application
type Dependencies struct {
	Config         *config.Config
	Notifier       notification.Notifier
	OutputMonitor  interfaces.DataHandler
	ProcessManager *process.Manager
	stopChan       chan struct{}
}

// NewDependencies creates all dependencies with the given configuration
func NewDependencies(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config:   cfg,
		stopChan: make(chan struct{}),
	}

	// Create notification components
	baseNotifier := notification.NewNtfyClient(cfg.NtfyServer, cfg.NtfyTopic)

	// Wrap with backstop notifier if configured
	var finalNotifier notification.Notifier = baseNotifier
	if cfg.BackstopTimeout > 0 {
		finalNotifier = notification.NewBackstopNotifier(baseNotifier, cfg.BackstopTimeout)
	}
	deps.Notifier = finalNotifier

	// Create output monitor with the notifier
	outputMonitor := monitor.NewOutputMonitor(cfg, deps.Notifier)
	deps.OutputMonitor = outputMonitor

	// Create input handler that disables backstop timer
	inputHandler := func() {
		if backstopNotifier, ok := deps.Notifier.(*notification.BackstopNotifier); ok {
			backstopNotifier.DisableBackstopTimer()
			if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "claude-code-ntfy: user input detected, disabling backstop timer\n")
			}
		}
	}

	// Create process manager
	deps.ProcessManager = process.NewManager(cfg, deps.OutputMonitor, inputHandler)

	return deps, nil
}

// Close cleans up all dependencies
func (d *Dependencies) Close() {
	// Stop status indicator refresh
	if d.stopChan != nil {
		select {
		case <-d.stopChan:
			// Already closed
		default:
			close(d.stopChan)
		}
		d.stopChan = nil
	}

	// Close notifiers
	// First try to close as backstop notifier
	if backstopNotifier, ok := d.Notifier.(*notification.BackstopNotifier); ok {
		_ = backstopNotifier.Close()
	}
}

// Application represents the main application
type Application struct {
	deps *Dependencies
}

// NewApplication creates a new application with the given dependencies
func NewApplication(deps *Dependencies) *Application {
	return &Application{
		deps: deps,
	}
}

// Run starts the application with the given command and arguments
func (a *Application) Run(command string, args []string) error {
	// Send startup notification if configured
	if a.deps.Config.StartupNotify && !a.deps.Config.Quiet {
		pwd, _ := os.Getwd()
		startupNotification := notification.Notification{
			Title:   "Claude Code Session Started",
			Message: fmt.Sprintf("Working directory: %s", pwd),
			Time:    time.Now(),
			Pattern: "startup",
		}
		_ = a.deps.Notifier.Send(startupNotification)
	}

	if err := a.deps.ProcessManager.Start(command, args); err != nil {
		return err
	}

	return a.deps.ProcessManager.Wait()
}

// Stop gracefully stops the application
func (a *Application) Stop() error {
	return a.deps.ProcessManager.Stop()
}

// ExitCode returns the exit code of the wrapped process
func (a *Application) ExitCode() int {
	return a.deps.ProcessManager.ExitCode()
}
