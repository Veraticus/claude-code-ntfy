package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/idle"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
	"github.com/Veraticus/claude-code-ntfy/pkg/monitor"
	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
	"github.com/Veraticus/claude-code-ntfy/pkg/process"
	"github.com/Veraticus/claude-code-ntfy/pkg/status"
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
	StatusIndicator     *status.Indicator
	StatusReporter      *status.Reporter
	stopChan            chan struct{}
}

// NewDependencies creates all dependencies with the given configuration
func NewDependencies(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config:   cfg,
		stopChan: make(chan struct{}),
	}

	// Create idle detector
	deps.IdleDetector = idle.NewIdleDetector()

	// Create status indicator (only enabled if we have a terminal and notifications are enabled)
	// The indicator will only flash briefly when notifications are sent
	isTerminal := isatty(os.Stderr.Fd())
	statusEnabled := isTerminal && !cfg.Quiet && cfg.NtfyTopic != ""
	deps.StatusIndicator = status.NewIndicator(os.Stderr, statusEnabled)
	deps.StatusReporter = status.NewReporter(deps.StatusIndicator)

	// Start auto-refresh to keep the indicator visible despite Claude's screen clears
	deps.StatusIndicator.StartAutoRefresh(deps.stopChan)

	// Create pattern matcher and output monitor first (needed for terminal title)
	deps.PatternMatcher = monitor.NewSimplePatternMatcher(cfg.Patterns)
	outputMonitor := monitor.NewOutputMonitor(cfg, deps.PatternMatcher, deps.IdleDetector, nil) // nil notifier for now
	deps.OutputMonitor = outputMonitor

	// Create notification components
	baseNotifier := notification.NewNtfyClient(cfg.NtfyServer, cfg.NtfyTopic)
	
	// Wrap with context notifier
	contextNotifier := notification.NewContextNotifier(baseNotifier, func() string {
		return outputMonitor.GetTerminalTitle()
	})

	// Wrap with backstop notifier if configured
	var finalNotifier notification.Notifier = contextNotifier
	if cfg.BackstopTimeout > 0 {
		finalNotifier = notification.NewBackstopNotifier(contextNotifier, cfg.BackstopTimeout)
	}
	deps.Notifier = finalNotifier

	deps.RateLimiter = notification.NewTokenBucketRateLimiter(cfg.RateLimit.MaxMessages, cfg.RateLimit.Window)
	deps.NotificationManager = notification.NewManager(cfg, deps.Notifier, deps.RateLimiter)

	// Connect status reporter to notification manager
	deps.NotificationManager.SetStatusReporter(deps.StatusReporter)
	
	// Now set the notification manager on the output monitor
	outputMonitor.SetNotifier(deps.NotificationManager)

	// Connect status indicator to output monitor for screen clear detection
	if statusEnabled {
		outputMonitor.SetScreenEventHandler(deps.StatusIndicator)
		// Enable focus reporting display if configured
		if cfg.EnableFocusDetection {
			deps.StatusIndicator.SetFocusReportingEnabled(true)
		}
	}

	// Create process manager
	deps.ProcessManager = process.NewManager(cfg, deps.OutputMonitor)

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

	// Clean up status indicator
	if d.StatusIndicator != nil {
		_ = d.StatusIndicator.Clear() // Best effort
	}

	// Close notifiers
	// First try to close as backstop notifier
	if backstopNotifier, ok := d.Notifier.(*notification.BackstopNotifier); ok {
		_ = backstopNotifier.Close()
	}

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
	// Send startup notification if configured
	if a.deps.Config.StartupNotify && !a.deps.Config.Quiet && a.deps.NotificationManager != nil {
		pwd, _ := os.Getwd()
		startupNotification := notification.Notification{
			Title:   "Claude Code Session Started",
			Message: fmt.Sprintf("Working directory: %s", pwd),
			Time:    time.Now(),
			Pattern: "startup",
		}
		_ = a.deps.NotificationManager.Send(startupNotification)

		// After sending startup notification, update the startup time in output monitor
		// to suppress subsequent notifications for StartupGracePeriod
		if om, ok := a.deps.OutputMonitor.(*monitor.OutputMonitor); ok {
			om.ResetStartTime()
		}
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
