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
