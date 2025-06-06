package notification

import (
	"sync"
	"time"
)

// ActivityMarker is an interface for marking activity
type ActivityMarker interface {
	MarkActivity()
}

// BackstopNotifier wraps another notifier and sends a notification after inactivity
type BackstopNotifier struct {
	underlying Notifier
	timeout    time.Duration

	mu                   sync.Mutex
	lastNotificationTime time.Time
	lastActivityTime     time.Time
	timer                *time.Timer
	backstopSent         bool // Track if backstop notification was sent for current session
}

// NewBackstopNotifier creates a new backstop notifier
func NewBackstopNotifier(underlying Notifier, timeout time.Duration) *BackstopNotifier {
	bn := &BackstopNotifier{
		underlying:       underlying,
		timeout:          timeout,
		lastActivityTime: time.Now(),
	}

	if timeout > 0 {
		bn.startTimer()
	}

	return bn
}

// Send implements the Notifier interface
func (bn *BackstopNotifier) Send(notification Notification) error {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	// Reset activity time
	bn.lastActivityTime = time.Now()
	bn.lastNotificationTime = time.Now()

	// Reset backstop sent flag since we have new activity
	bn.backstopSent = false

	// Reset the timer
	if bn.timer != nil {
		bn.timer.Stop()
	}
	// Always restart timer after a notification
	if bn.timeout > 0 {
		bn.timer = time.AfterFunc(bn.timeout, bn.sendBackstopNotification)
	}

	// Forward to underlying notifier
	return bn.underlying.Send(notification)
}

// MarkActivity marks that there was activity (output) without sending a notification
func (bn *BackstopNotifier) MarkActivity() {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	bn.lastActivityTime = time.Now()

	// Reset backstop sent flag since we have new activity
	bn.backstopSent = false

	// Reset the timer
	if bn.timer != nil {
		bn.timer.Stop()
	}
	// Always restart timer after activity
	if bn.timeout > 0 {
		bn.timer = time.AfterFunc(bn.timeout, bn.sendBackstopNotification)
	}
}

// sendBackstopNotification sends a notification after inactivity
func (bn *BackstopNotifier) sendBackstopNotification() {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	// Only send if we haven't already sent a backstop for this session
	if bn.backstopSent {
		return
	}

	// Send backstop notification
	notification := Notification{
		Title:   "Claude Code: Inactive",
		Message: "No activity detected - task may be complete",
		Time:    time.Now(),
		Pattern: "backstop",
	}

	bn.lastNotificationTime = time.Now()
	bn.backstopSent = true

	// Send via underlying notifier
	_ = bn.underlying.Send(notification)

	// Do NOT restart timer - we only send one backstop per session
}

// startTimer starts the initial timer
func (bn *BackstopNotifier) startTimer() {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	if bn.timeout > 0 {
		bn.timer = time.AfterFunc(bn.timeout, bn.sendBackstopNotification)
	}
}

// SetBackstopSent sets the backstop sent flag
func (bn *BackstopNotifier) SetBackstopSent(sent bool) {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	bn.backstopSent = sent

	// If we're marking it as sent, stop the timer
	if sent && bn.timer != nil {
		bn.timer.Stop()
	}
}

// ResetSession resets the backstop state for a new prompt/session
func (bn *BackstopNotifier) ResetSession() {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	bn.backstopSent = false
	bn.lastActivityTime = time.Now()

	// Reset the timer
	if bn.timer != nil {
		bn.timer.Stop()
	}
	// Start a new timer for the new session
	if bn.timeout > 0 {
		bn.timer = time.AfterFunc(bn.timeout, bn.sendBackstopNotification)
	}
}

// Close stops the timer
func (bn *BackstopNotifier) Close() error {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	if bn.timer != nil {
		bn.timer.Stop()
	}

	return nil
}
