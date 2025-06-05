package notification

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// MockNotifier for testing
type MockNotifier struct {
	mu            sync.Mutex
	notifications []Notification
	attempts      []Notification // Track all send attempts
	sendErr       error
	sendDelay     time.Duration
}

func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		notifications: []Notification{},
		attempts:      []Notification{},
	}
}

func (m *MockNotifier) Send(n Notification) error {
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

func (m *MockNotifier) GetNotifications() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *MockNotifier) GetAttempts() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]Notification, len(m.attempts))
	copy(result, m.attempts)
	return result
}

func (m *MockNotifier) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

// MockRateLimiter for testing
type MockRateLimiter struct {
	mu          sync.Mutex
	allowResult bool
	callCount   int
	resetCount  int
}

func NewMockRateLimiter(allowResult bool) *MockRateLimiter {
	return &MockRateLimiter{
		allowResult: allowResult,
	}
}

func (m *MockRateLimiter) Allow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.allowResult
}

func (m *MockRateLimiter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetCount++
}

func (m *MockRateLimiter) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *MockRateLimiter) GetResetCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resetCount
}

func TestManager_Send(t *testing.T) {
	tests := []struct {
		name                  string
		config                *config.Config
		rateLimiterAllows     bool
		notifierError         error
		notification          Notification
		wantNotificationSent  bool
		wantRateLimiterCalled bool
	}{
		{
			name: "successful send without batching",
			config: &config.Config{
				BatchWindow: 0, // No batching
			},
			rateLimiterAllows: true,
			notification: Notification{
				Title:   "Test",
				Message: "Test message",
			},
			wantNotificationSent:  true,
			wantRateLimiterCalled: true,
		},
		{
			name: "rate limited",
			config: &config.Config{
				BatchWindow: 0,
			},
			rateLimiterAllows: false,
			notification: Notification{
				Title:   "Test",
				Message: "Test message",
			},
			wantNotificationSent:  false,
			wantRateLimiterCalled: true,
		},
		{
			name: "notifier error",
			config: &config.Config{
				BatchWindow: 0,
			},
			rateLimiterAllows: true,
			notifierError:     errors.New("send failed"),
			notification: Notification{
				Title:   "Test",
				Message: "Test message",
			},
			wantNotificationSent:  true, // Still attempted
			wantRateLimiterCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotifier := NewMockNotifier()
			mockNotifier.SetError(tt.notifierError)

			mockRateLimiter := NewMockRateLimiter(tt.rateLimiterAllows)

			manager := NewManager(tt.config, mockNotifier, mockRateLimiter)
			defer func() { _ = manager.Close() }()

			_ = manager.Send(tt.notification)

			// Allow time for async processing
			time.Sleep(50 * time.Millisecond)

			// Check if notification was sent
			// For error cases, check attempts rather than successful sends
			if tt.notifierError != nil {
				attempts := mockNotifier.GetAttempts()
				if tt.wantNotificationSent && len(attempts) == 0 {
					t.Error("Expected notification to be attempted, but none were attempted")
				}
			} else {
				notifications := mockNotifier.GetNotifications()
				if tt.wantNotificationSent && len(notifications) == 0 {
					t.Error("Expected notification to be sent, but none were sent")
				}
				if !tt.wantNotificationSent && len(notifications) > 0 {
					t.Errorf("Expected no notifications, but got %d", len(notifications))
				}
			}

			// Check if rate limiter was called
			if tt.wantRateLimiterCalled && mockRateLimiter.GetCallCount() == 0 {
				t.Error("Expected rate limiter to be called, but it wasn't")
			}
		})
	}
}

func TestManager_SendWithBatching(t *testing.T) {
	mockNotifier := NewMockNotifier()
	mockRateLimiter := NewMockRateLimiter(true) // Always allow

	cfg := &config.Config{
		BatchWindow: 100 * time.Millisecond,
	}

	manager := NewManager(cfg, mockNotifier, mockRateLimiter)
	defer func() { _ = manager.Close() }()

	// Send multiple notifications
	n1 := Notification{Title: "1"}
	n2 := Notification{Title: "2"}
	n3 := Notification{Title: "3"}

	_ = manager.Send(n1)
	_ = manager.Send(n2)
	_ = manager.Send(n3)

	// Should not be sent immediately
	time.Sleep(50 * time.Millisecond)
	if len(mockNotifier.GetNotifications()) != 0 {
		t.Error("Notifications sent before batch window")
	}

	// Wait for batch window to expire
	time.Sleep(100 * time.Millisecond)

	notifications := mockNotifier.GetNotifications()
	if len(notifications) != 1 {
		t.Errorf("Expected 1 batched notification, got %d", len(notifications))
	}

	// Verify it's a batch notification
	if len(notifications) > 0 {
		batch := notifications[0]
		if batch.Title != "Claude Code: Multiple Matches" {
			t.Errorf("Expected batch title, got %q", batch.Title)
		}
		if batch.Pattern != "batch" {
			t.Errorf("Expected batch pattern, got %q", batch.Pattern)
		}
	}
}

func TestManager_Close(t *testing.T) {
	tests := []struct {
		name         string
		withBatching bool
		pendingCount int
	}{
		{
			name:         "close without batching",
			withBatching: false,
			pendingCount: 0,
		},
		{
			name:         "close with batching and pending notifications",
			withBatching: true,
			pendingCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotifier := NewMockNotifier()
			mockRateLimiter := NewMockRateLimiter(true)

			batchWindow := time.Duration(0)
			if tt.withBatching {
				batchWindow = time.Hour // Long window so notifications stay pending
			}

			cfg := &config.Config{
				BatchWindow: batchWindow,
			}

			manager := NewManager(cfg, mockNotifier, mockRateLimiter)

			// Add pending notifications
			for i := 0; i < tt.pendingCount; i++ {
				_ = manager.Send(Notification{Title: string(rune('A' + i))})
			}

			// Close should flush pending notifications
			_ = manager.Close()

			// Allow time for processing
			time.Sleep(50 * time.Millisecond)

			if tt.withBatching {
				// Should have flushed all pending as a single batch
				notifications := mockNotifier.GetNotifications()
				if tt.pendingCount > 0 && len(notifications) != 1 {
					t.Errorf("Expected 1 batched notification, got %d", len(notifications))
				}
				if tt.pendingCount == 0 && len(notifications) != 0 {
					t.Errorf("Expected no notifications, got %d", len(notifications))
				}
			}

			// Closing again should be safe
			_ = manager.Close()
		})
	}
}

func TestManager_ConcurrentSend(t *testing.T) {
	mockNotifier := NewMockNotifier()
	mockRateLimiter := NewMockRateLimiter(true)

	cfg := &config.Config{
		BatchWindow: 0, // No batching for simpler test
	}

	manager := NewManager(cfg, mockNotifier, mockRateLimiter)
	defer func() { _ = manager.Close() }()

	// Send notifications concurrently
	numGoroutines := 10
	notificationsPerGoroutine := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < notificationsPerGoroutine; j++ {
				_ = manager.Send(Notification{
					Title: string(rune('A' + id)),
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow processing

	notifications := mockNotifier.GetNotifications()
	expectedTotal := numGoroutines * notificationsPerGoroutine
	if len(notifications) != expectedTotal {
		t.Errorf("Expected %d notifications, got %d", expectedTotal, len(notifications))
	}
}

// CountingRateLimiter that allows first N calls
type CountingRateLimiter struct {
	mu           sync.Mutex
	maxAllowed   int
	currentCount int
}

func (c *CountingRateLimiter) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount++
	return c.currentCount <= c.maxAllowed
}

func (c *CountingRateLimiter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount = 0
}

func TestManager_RateLimitingWithBatching(t *testing.T) {
	mockNotifier := NewMockNotifier()

	// Rate limiter that allows first 2 calls only
	rateLimiter := &CountingRateLimiter{
		maxAllowed: 2,
	}

	cfg := &config.Config{
		BatchWindow: 50 * time.Millisecond,
	}

	manager := NewManager(cfg, mockNotifier, rateLimiter)
	defer func() { _ = manager.Close() }()

	// Send 5 notifications in first batch
	for i := 0; i < 5; i++ {
		_ = manager.Send(Notification{Title: string(rune('A' + i))})
	}

	// Wait for batch
	time.Sleep(100 * time.Millisecond)

	// Send 3 more in second batch
	for i := 0; i < 3; i++ {
		_ = manager.Send(Notification{Title: string(rune('X' + i))})
	}

	// Wait for second batch
	time.Sleep(100 * time.Millisecond)

	// Only 1 batch should have been sent (rate limited to 2 notifications total)
	notifications := mockNotifier.GetNotifications()
	if len(notifications) != 1 {
		t.Errorf("Expected 1 batch notification due to rate limiting, got %d", len(notifications))
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		notifier    Notifier
		rateLimiter interfaces.RateLimiter
	}{
		{
			name: "with batching",
			config: &config.Config{
				BatchWindow: 100 * time.Millisecond,
			},
			notifier:    NewMockNotifier(),
			rateLimiter: NewMockRateLimiter(true),
		},
		{
			name: "without batching",
			config: &config.Config{
				BatchWindow: 0,
			},
			notifier:    NewMockNotifier(),
			rateLimiter: NewMockRateLimiter(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config, tt.notifier, tt.rateLimiter)
			if manager == nil {
				t.Error("NewManager() returned nil")
			}
			_ = manager.Close()
		})
	}
}
