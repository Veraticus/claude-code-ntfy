package idle

import (
	"sync"
	"time"
)

// OutputBasedDetector tracks user activity based on output received.
// This serves as a fallback when platform-specific idle detection is not available.
type OutputBasedDetector struct {
	mu           sync.RWMutex
	lastActivity time.Time
}

// NewOutputBasedDetector creates a new output-based idle detector.
func NewOutputBasedDetector() *OutputBasedDetector {
	return &OutputBasedDetector{
		lastActivity: time.Now(),
	}
}

// IsUserIdle returns true if no output has been received within the threshold duration.
func (d *OutputBasedDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	elapsed := time.Since(d.lastActivity)
	return elapsed >= threshold, nil
}

// LastActivity returns the last time output was received.
func (d *OutputBasedDetector) LastActivity() time.Time {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.lastActivity
}

// UpdateActivity updates the last activity time to now.
// This should be called whenever output is received.
func (d *OutputBasedDetector) UpdateActivity() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastActivity = time.Now()
}

// UpdateActivityTime updates the last activity time to the specified time.
// This is useful for testing or when you need to set a specific activity time.
func (d *OutputBasedDetector) UpdateActivityTime(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastActivity = t
}
