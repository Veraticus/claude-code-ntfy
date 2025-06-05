package idle

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TmuxIdleDetector detects idle time in tmux sessions.
type TmuxIdleDetector struct {
	sessionName string
	cmdExecutor func(name string, args ...string) ([]byte, error)
}

// NewTmuxIdleDetector creates a new tmux idle detector.
// If sessionName is empty, it will attempt to detect the current session.
func NewTmuxIdleDetector(sessionName string) *TmuxIdleDetector {
	return &TmuxIdleDetector{
		sessionName: sessionName,
		cmdExecutor: defaultCmdExecutor,
	}
}

// defaultCmdExecutor executes a command and returns its output.
func defaultCmdExecutor(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// IsUserIdle returns true if the user has been idle in tmux for longer than the threshold.
func (d *TmuxIdleDetector) IsUserIdle(threshold time.Duration) (bool, error) {
	idleTime, err := d.getIdleTime()
	if err != nil {
		return false, err
	}

	return idleTime >= threshold, nil
}

// LastActivity returns the last activity time based on tmux idle time.
func (d *TmuxIdleDetector) LastActivity() time.Time {
	idleTime, err := d.getIdleTime()
	if err != nil {
		// If we can't get tmux idle time, assume activity just happened
		return time.Now()
	}

	return time.Now().Add(-idleTime)
}

// getIdleTime retrieves the idle time from tmux.
func (d *TmuxIdleDetector) getIdleTime() (time.Duration, error) {
	// First check if we're in a tmux session
	if !d.isInTmux() {
		return 0, fmt.Errorf("not in a tmux session")
	}

	// Get the session name if not provided
	sessionName := d.sessionName
	if sessionName == "" {
		name, err := d.getCurrentSessionName()
		if err != nil {
			return 0, fmt.Errorf("failed to get current session name: %w", err)
		}
		sessionName = name
	}

	// Get idle time for all clients in the session
	idleTime, err := d.getSessionIdleTime(sessionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get session idle time: %w", err)
	}

	return idleTime, nil
}

// isInTmux checks if we're running inside a tmux session.
func (d *TmuxIdleDetector) isInTmux() bool {
	// Check TMUX environment variable
	tmuxVar := os.Getenv("TMUX")
	return tmuxVar != ""
}

// getCurrentSessionName gets the name of the current tmux session.
func (d *TmuxIdleDetector) getCurrentSessionName() (string, error) {
	output, err := d.cmdExecutor("tmux", "display-message", "-p", "#{session_name}")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getSessionIdleTime gets the minimum idle time across all clients in a session.
func (d *TmuxIdleDetector) getSessionIdleTime(sessionName string) (time.Duration, error) {
	// List all clients and their idle times for the session
	output, err := d.cmdExecutor("tmux", "list-clients", "-t", sessionName, "-F", "#{client_activity}")
	if err != nil {
		// If the command fails, it might be because there are no clients
		// or the session doesn't exist
		return 0, err
	}

	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	if len(lines) == 0 {
		return 0, fmt.Errorf("no clients found for session %s", sessionName)
	}

	// Find the most recent activity (minimum idle time)
	var mostRecentActivity time.Time
	now := time.Now()

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// client_activity is seconds since epoch
		activityStr := string(line)
		activitySecs, err := strconv.ParseInt(activityStr, 10, 64)
		if err != nil {
			continue
		}

		activityTime := time.Unix(activitySecs, 0)
		if mostRecentActivity.IsZero() || activityTime.After(mostRecentActivity) {
			mostRecentActivity = activityTime
		}
	}

	if mostRecentActivity.IsZero() {
		return 0, fmt.Errorf("could not parse any client activity times")
	}

	idleTime := now.Sub(mostRecentActivity)
	if idleTime < 0 {
		// Handle clock skew or other timing issues
		idleTime = 0
	}

	return idleTime, nil
}

// IsAvailable checks if tmux is available and we're in a tmux session.
func (d *TmuxIdleDetector) IsAvailable() bool {
	if !d.isInTmux() {
		return false
	}

	// Check if tmux command is available
	_, err := d.cmdExecutor("tmux", "-V")
	return err == nil
}
