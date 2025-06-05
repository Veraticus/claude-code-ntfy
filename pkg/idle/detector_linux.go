//go:build linux
// +build linux

package idle

import (
	"time"
)

// LinuxIdleDetector implements idle detection for Linux systems.
// It attempts to use tmux idle detection if available, falling back
// to output-based detection if not.
type LinuxIdleDetector struct {
	tmuxDetector *TmuxIdleDetector
	fallback     *OutputBasedDetector
	useTmux      bool
}

// NewLinuxIdleDetector creates a new Linux idle detector.
func NewLinuxIdleDetector() *LinuxIdleDetector {
	tmuxDetector := NewTmuxIdleDetector("")
	fallback := NewOutputBasedDetector()

	// Check if tmux detection is available
	useTmux := tmuxDetector.IsAvailable()

	return &LinuxIdleDetector{
		tmuxDetector: tmuxDetector,
		fallback:     fallback,
		useTmux:      useTmux,
	}
}

// IsUserIdle returns true if the user has been idle for longer than the threshold.
func (d *LinuxIdleDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	if d.useTmux {
		idle, err := d.tmuxDetector.IsUserIdle(threshold)
		if err == nil {
			return idle, nil
		}
		// If tmux detection fails, fall back to output-based
	}

	return d.fallback.IsUserIdle(threshold)
}

// LastActivity returns the last time the user was active.
func (d *LinuxIdleDetector) LastActivity() time.Time {
	if d.useTmux {
		// Try to get activity from tmux
		activity := d.tmuxDetector.LastActivity()
		// If the activity time seems reasonable (not just "now" due to error),
		// and is different from the fallback time, use it
		fallbackActivity := d.fallback.LastActivity()
		if activity.Before(fallbackActivity) {
			return activity
		}
	}

	return d.fallback.LastActivity()
}

// UpdateActivity updates the activity time in the fallback detector.
// This should be called whenever output is received.
func (d *LinuxIdleDetector) UpdateActivity() {
	d.fallback.UpdateActivity()
}

// UpdateActivityTime updates the activity time to the specified time.
func (d *LinuxIdleDetector) UpdateActivityTime(t time.Time) {
	d.fallback.UpdateActivityTime(t)
}

// IsUsingTmux returns true if tmux detection is being used.
func (d *LinuxIdleDetector) IsUsingTmux() bool {
	return d.useTmux
}

// SetUseTmux allows enabling or disabling tmux detection.
// This is primarily useful for testing.
func (d *LinuxIdleDetector) SetUseTmux(use bool) {
	d.useTmux = use
}
