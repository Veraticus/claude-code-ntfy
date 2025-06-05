//go:build darwin
// +build darwin

package idle

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DarwinIdleDetector implements idle detection for macOS systems.
// It uses ioreg to query the system idle time, falling back to
// output-based detection if that fails.
type DarwinIdleDetector struct {
	fallback    *OutputBasedDetector
	cmdExecutor func(name string, args ...string) ([]byte, error)
}

// NewDarwinIdleDetector creates a new Darwin (macOS) idle detector.
func NewDarwinIdleDetector() *DarwinIdleDetector {
	return &DarwinIdleDetector{
		fallback:    NewOutputBasedDetector(),
		cmdExecutor: defaultDarwinCmdExecutor,
	}
}

// defaultDarwinCmdExecutor executes a command and returns its output.
func defaultDarwinCmdExecutor(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// IsUserIdle returns true if the user has been idle for longer than the threshold.
func (d *DarwinIdleDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	idleTime, err := d.getSystemIdleTime()
	if err != nil {
		// Fall back to output-based detection
		return d.fallback.IsUserIdle(threshold)
	}

	return idleTime >= threshold, nil
}

// LastActivity returns the last time the user was active.
func (d *DarwinIdleDetector) LastActivity() time.Time {
	idleTime, err := d.getSystemIdleTime()
	if err != nil {
		// Fall back to output-based detection
		return d.fallback.LastActivity()
	}

	return time.Now().Add(-idleTime)
}

// getSystemIdleTime retrieves the system idle time using ioreg.
func (d *DarwinIdleDetector) getSystemIdleTime() (time.Duration, error) {
	// Use ioreg to get HIDIdleTime
	output, err := d.cmdExecutor("ioreg", "-c", "IOHIDSystem", "-d", "4")
	if err != nil {
		return 0, fmt.Errorf("failed to execute ioreg: %w", err)
	}

	// Parse the output to find HIDIdleTime
	idleNanos, err := d.parseHIDIdleTime(output)
	if err != nil {
		return 0, fmt.Errorf("failed to parse HIDIdleTime: %w", err)
	}

	return time.Duration(idleNanos), nil
}

// parseHIDIdleTime parses the HIDIdleTime from ioreg output.
func (d *DarwinIdleDetector) parseHIDIdleTime(output []byte) (int64, error) {
	// Look for the HIDIdleTime line in the output
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		lineStr := string(bytes.TrimSpace(line))
		if strings.Contains(lineStr, "HIDIdleTime") {
			// Extract the value
			// Format: "HIDIdleTime" = 123456789
			parts := strings.Split(lineStr, "=")
			if len(parts) != 2 {
				continue
			}

			// Clean up the value
			valueStr := strings.TrimSpace(parts[1])
			// Remove any quotes or other characters
			valueStr = strings.Trim(valueStr, "\"")
			valueStr = strings.TrimSpace(valueStr)

			// Parse as int64 (nanoseconds)
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse idle time value: %w", err)
			}

			return value, nil
		}
	}

	return 0, fmt.Errorf("HIDIdleTime not found in ioreg output")
}

// UpdateActivity updates the activity time in the fallback detector.
// This should be called whenever output is received.
func (d *DarwinIdleDetector) UpdateActivity() {
	d.fallback.UpdateActivity()
}

// UpdateActivityTime updates the activity time to the specified time.
func (d *DarwinIdleDetector) UpdateActivityTime(t time.Time) {
	d.fallback.UpdateActivityTime(t)
}

// IsAvailable checks if ioreg is available on the system.
func (d *DarwinIdleDetector) IsAvailable() bool {
	_, err := d.cmdExecutor("which", "ioreg")
	return err == nil
}
