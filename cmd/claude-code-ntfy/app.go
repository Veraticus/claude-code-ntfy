package main

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/idle"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
	"github.com/Veraticus/claude-code-ntfy/pkg/monitor"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
	"github.com/Veraticus/claude-code-ntfy/pkg/process"
)

// Dependencies holds all the dependencies for the application
type Dependencies struct {
	Config              *config.Config
	IdleDetector        interfaces.IdleDetector
	Notifier            notification.Notifier
	RateLimiter         interfaces.RateLimiter
	PatternMatcher      monitor.PatternMatcher
	NotificationManager *notification.Manager
	OutputMonitor       interfaces.DataHandler
	ProcessManager      *process.Manager
}

// NewDependencies creates all dependencies with the given configuration
func NewDependencies(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config: cfg,
	}

	// Create idle detector
	deps.IdleDetector = idle.NewIdleDetector()

	// Create notification components
	deps.Notifier = notification.NewNtfyClient(cfg.NtfyServer, cfg.NtfyTopic)
	deps.RateLimiter = notification.NewTokenBucketRateLimiter(cfg.RateLimit.MaxMessages, cfg.RateLimit.Window)
	deps.NotificationManager = notification.NewManager(cfg, deps.Notifier, deps.RateLimiter)

	// Create pattern matcher and output monitor
	deps.PatternMatcher = monitor.NewSimplePatternMatcher(cfg.Patterns)
	deps.OutputMonitor = monitor.NewOutputMonitor(cfg, deps.PatternMatcher, deps.IdleDetector, deps.NotificationManager)

	// Create process manager
	deps.ProcessManager = process.NewManager(cfg, deps.OutputMonitor)

	return deps, nil
}

// Close cleans up all dependencies
func (d *Dependencies) Close() {
	if d.NotificationManager != nil {
		_ = d.NotificationManager.Close()
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
