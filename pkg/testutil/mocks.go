package testutil

import (
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
)

// MockNotifier is a thread-safe mock implementation of notification.Notifier for testing
type MockNotifier struct {
	mu            sync.Mutex
	notifications []notification.Notification
	attempts      []notification.Notification // Track all send attempts
	sendErr       error
	sendDelay     time.Duration
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		notifications: []notification.Notification{},
		attempts:      []notification.Notification{},
	}
}

// Send implements the Notifier interface
func (m *MockNotifier) Send(n notification.Notification) error {
	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Always track the attempt
	m.attempts = append(m.attempts, n)

	if m.sendErr != nil {
		return m.sendErr
	}

	m.notifications = append(m.notifications, n)
	return nil
}

// GetNotifications returns a copy of successfully sent notifications
func (m *MockNotifier) GetNotifications() []notification.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]notification.Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

// GetAttempts returns a copy of all attempted sends (including failures)
func (m *MockNotifier) GetAttempts() []notification.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]notification.Notification, len(m.attempts))
	copy(result, m.attempts)
	return result
}

// SetError sets the error to return on Send calls
func (m *MockNotifier) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

// SetDelay sets a delay before each Send call
func (m *MockNotifier) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendDelay = delay
}

// Clear resets the mock state
func (m *MockNotifier) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = []notification.Notification{}
	m.attempts = []notification.Notification{}
	m.sendErr = nil
	m.sendDelay = 0
}

// MockIdleDetector is a mock implementation of interfaces.IdleDetector for testing
type MockIdleDetector struct {
	mu               sync.Mutex
	isIdle           bool
	lastActivityTime time.Time
	isIdleCallCount  int
}

// NewMockIdleDetector creates a new mock idle detector
func NewMockIdleDetector(isIdle bool) *MockIdleDetector {
	return &MockIdleDetector{
		isIdle:           isIdle,
		lastActivityTime: time.Now(),
	}
}

// IsIdle implements the IdleDetector interface
func (m *MockIdleDetector) IsIdle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isIdleCallCount++
	return m.isIdle
}

// LastActivity implements the IdleDetector interface
func (m *MockIdleDetector) LastActivity() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastActivityTime
}

// SetIdle sets the idle state
func (m *MockIdleDetector) SetIdle(idle bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isIdle = idle
}

// SetLastActivity sets the last activity time
func (m *MockIdleDetector) SetLastActivity(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActivityTime = t
}

// GetIsIdleCallCount returns how many times IsIdle was called
func (m *MockIdleDetector) GetIsIdleCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isIdleCallCount
}

// MockRateLimiter is a mock implementation of interfaces.RateLimiter for testing
type MockRateLimiter struct {
	mu          sync.Mutex
	allowResult bool
	allowCount  int
	resetCount  int
}

// NewMockRateLimiter creates a new mock rate limiter
func NewMockRateLimiter(allowResult bool) *MockRateLimiter {
	return &MockRateLimiter{
		allowResult: allowResult,
	}
}

// Allow implements the RateLimiter interface
func (m *MockRateLimiter) Allow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowCount++
	return m.allowResult
}

// Reset implements the RateLimiter interface
func (m *MockRateLimiter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetCount++
}

// SetAllowResult sets the result that Allow() will return
func (m *MockRateLimiter) SetAllowResult(allow bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowResult = allow
}

// GetAllowCount returns how many times Allow was called
func (m *MockRateLimiter) GetAllowCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.allowCount
}

// GetResetCount returns how many times Reset was called
func (m *MockRateLimiter) GetResetCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resetCount
}

// CountingRateLimiter is a rate limiter that allows first N calls
type CountingRateLimiter struct {
	mu           sync.Mutex
	maxAllowed   int
	currentCount int
}

// NewCountingRateLimiter creates a new counting rate limiter
func NewCountingRateLimiter(maxAllowed int) *CountingRateLimiter {
	return &CountingRateLimiter{
		maxAllowed: maxAllowed,
	}
}

// Allow implements the RateLimiter interface
func (c *CountingRateLimiter) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount++
	return c.currentCount <= c.maxAllowed
}

// Reset implements the RateLimiter interface
func (c *CountingRateLimiter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount = 0
}
