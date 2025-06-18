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

	mu                                       sync.Mutex
	lastNotificationTime                     time.Time
	lastActivityTime                         time.Time
	lastUserInteraction                      time.Time
	timer                                    *time.Timer
	backstopSent                             bool // Track if backstop notification was sent for current session
	backstopDisabled                         bool // Track if backstop timer has been disabled by user input
	idleNotificationSentSinceLastInteraction bool // Track if we've sent an idle notification since last user interaction
}

// NewBackstopNotifier creates a new backstop notifier
func NewBackstopNotifier(underlying Notifier, timeout time.Duration) *BackstopNotifier {
	bn := &BackstopNotifier{
		underlying:          underlying,
		timeout:             timeout,
		lastActivityTime:    time.Now(),
		lastUserInteraction: time.Now(),
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

	// Reset backstop sent flag and disabled flag since we have new activity
	bn.backstopSent = false
	bn.backstopDisabled = false

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

	// Only send if we haven't already sent a backstop for this session and it's not disabled
	if bn.backstopSent || bn.backstopDisabled {
		return
	}

	// Check if we've already sent an idle notification since the last user interaction
	if bn.idleNotificationSentSinceLastInteraction {
		return
	}

	// Send backstop notification
	notification := Notification{
		Title:   "Claude needs attention",
		Message: "No activity detected",
		Time:    time.Now(),
		Pattern: "backstop",
	}

	bn.lastNotificationTime = time.Now()
	bn.backstopSent = true
	bn.idleNotificationSentSinceLastInteraction = true

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
	bn.backstopDisabled = false
	bn.lastActivityTime = time.Now()
	// Reset idle notification flag since this is a new session that warrants attention
	bn.idleNotificationSentSinceLastInteraction = false

	// Reset the timer
	if bn.timer != nil {
		bn.timer.Stop()
	}
	// Start a new timer for the new session
	if bn.timeout > 0 {
		bn.timer = time.AfterFunc(bn.timeout, bn.sendBackstopNotification)
	}
}

// DisableBackstopTimer disables the backstop timer (e.g., when user input is detected)
func (bn *BackstopNotifier) DisableBackstopTimer() {
	bn.mu.Lock()
	defer bn.mu.Unlock()

	bn.backstopDisabled = true
	bn.lastUserInteraction = time.Now()
	bn.idleNotificationSentSinceLastInteraction = false

	// Stop the timer
	if bn.timer != nil {
		bn.timer.Stop()
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
