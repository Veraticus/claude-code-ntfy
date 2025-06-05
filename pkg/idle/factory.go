// Package idle provides idle detection functionality for determining user activity status.
package idle

import (
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// NewIdleDetector creates a platform-appropriate idle detector.
// It returns:
// - LinuxIdleDetector on Linux systems (with tmux support if available)
// - DarwinIdleDetector on macOS systems (using ioreg)
// - OutputBasedDetector on other platforms as a fallback.
func NewIdleDetector() interfaces.IdleDetector {
	return newPlatformDetector()
}

// NewIdleDetectorWithFallback creates an idle detector with a guaranteed fallback.
// This is useful when you need to ensure output-based detection is always available,
// regardless of platform-specific detection capabilities.
func NewIdleDetectorWithFallback() interfaces.IdleDetector {
	detector := NewIdleDetector()

	// If the detector is already an OutputBasedDetector, just return it
	if _, ok := detector.(*OutputBasedDetector); ok {
		return detector
	}

	// Otherwise, wrap it to ensure UpdateActivity methods are available
	return &idleDetectorWrapper{
		detector: detector,
		fallback: NewOutputBasedDetector(),
	}
}

// idleDetectorWrapper wraps a platform-specific detector with output-based fallback.
type idleDetectorWrapper struct {
	detector interfaces.IdleDetector
	fallback *OutputBasedDetector
}

// IsUserIdle delegates to the wrapped detector.
func (w *idleDetectorWrapper) IsUserIdle(threshold time.Duration) (bool, error) {
	return w.detector.IsUserIdle(threshold)
}

// LastActivity delegates to the wrapped detector.
func (w *idleDetectorWrapper) LastActivity() time.Time {
	return w.detector.LastActivity()
}

// UpdateActivity updates the fallback detector's activity time.
func (w *idleDetectorWrapper) UpdateActivity() {
	w.fallback.UpdateActivity()

	// Also update platform-specific detector if it supports it
	if obd, ok := w.detector.(*OutputBasedDetector); ok {
		obd.UpdateActivity()
	}
}

// UpdateActivityTime updates the fallback detector's activity time to a specific time.
func (w *idleDetectorWrapper) UpdateActivityTime(t time.Time) {
	w.fallback.UpdateActivityTime(t)

	// Also update platform-specific detector if it supports it
	if obd, ok := w.detector.(*OutputBasedDetector); ok {
		obd.UpdateActivityTime(t)
	}
}
