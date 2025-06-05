// Package idle provides idle detection functionality.
package idle

import (
	"time"
)

// SimpleDetector is a basic idle detector that always considers the user idle
type SimpleDetector struct {
	lastActivity time.Time
}

// NewSimpleDetector creates a new simple idle detector
func NewSimpleDetector() *SimpleDetector {
	return &SimpleDetector{
		lastActivity: time.Now(),
	}
}

// IsUserIdle always returns true for now (assumes user is idle)
func (d *SimpleDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	// For now, always consider user idle to enable notifications
	// This will be replaced with proper platform-specific detection
	return true, nil
}

// LastActivity returns the last activity time
func (d *SimpleDetector) LastActivity() time.Time {
	return d.lastActivity
}

