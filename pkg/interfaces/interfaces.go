// Package interfaces defines the core interfaces used throughout the application.
package interfaces

import "time"

// IdleDetector detects user activity/inactivity.
type IdleDetector interface {
	IsUserIdle(threshold time.Duration) (bool, error)
	LastActivity() time.Time
}

// ProcessWrapper wraps and monitors a process.
type ProcessWrapper interface {
	Start(command string, args []string) error
	Wait() error
	ExitCode() int
}

// OutputHandler processes output lines.
type OutputHandler interface {
	HandleLine(line string)
}

// DataHandler processes raw output data.
type DataHandler interface {
	OutputHandler
	HandleData(data []byte)
}

// RateLimiter limits notification frequency.
type RateLimiter interface {
	Allow() bool
	Reset()
}

// StatusReporter reports status updates for operations.
// This can be used for notifications, process status, or any other status updates.
type StatusReporter interface {
	ReportSending()
	ReportSuccess()
	ReportFailure()
}

// ScreenEventHandler handles terminal screen events.
type ScreenEventHandler interface {
	// HandleScreenClear is called when a screen clear sequence is detected
	HandleScreenClear()
}

// TerminalSequenceDetector detects terminal escape sequences in output.
type TerminalSequenceDetector interface {
	// DetectSequences analyzes data for terminal sequences and calls appropriate handlers
	DetectSequences(data []byte, handler ScreenEventHandler)
}
