package notification

import (
	"sync"
	"testing"
	"time"
)

// testNotifier records notifications for testing
type testNotifier struct {
	mu            sync.Mutex
	notifications []Notification
	sendError     error
}

func (m *testNotifier) Send(n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}

	m.notifications = append(m.notifications, n)
	return nil
}

func (m *testNotifier) getNotifications() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func TestBackstopNotifier_SendResetsTimer(t *testing.T) {
	mock := &testNotifier{}
	backstop := NewBackstopNotifier(mock, 50*time.Millisecond)
	defer func() { _ = backstop.Close() }()

	// Send a notification
	n1 := Notification{Title: "Test 1", Pattern: "test"}
	err := backstop.Send(n1)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Wait less than timeout
	time.Sleep(30 * time.Millisecond)

	// Send another notification
	n2 := Notification{Title: "Test 2", Pattern: "test"}
	err = backstop.Send(n2)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Wait past original timeout
	time.Sleep(30 * time.Millisecond)

	// Should have only the two sent notifications, no backstop
	notifications := mock.getNotifications()
	if len(notifications) != 2 {
		t.Errorf("Expected 2 notifications, got %d", len(notifications))
	}
}

func TestBackstopNotifier_SendsAfterTimeout(t *testing.T) {
	mock := &testNotifier{}
	backstop := NewBackstopNotifier(mock, 50*time.Millisecond)
	defer func() { _ = backstop.Close() }()

	// Mark activity without sending
	backstop.MarkActivity()

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should have backstop notification
	notifications := mock.getNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	if notifications[0].Pattern != "backstop" {
		t.Errorf("Expected backstop pattern, got %s", notifications[0].Pattern)
	}
}

func TestBackstopNotifier_SendsOnlyOncePerSession(t *testing.T) {
	mock := &testNotifier{}
	backstop := NewBackstopNotifier(mock, 50*time.Millisecond)
	defer func() { _ = backstop.Close() }()

	// Mark activity to start timer
	backstop.MarkActivity()

	// Wait for backstop notification
	time.Sleep(100 * time.Millisecond)

	// Should have exactly one backstop notification
	notifications := mock.getNotifications()
	backstopCount := 0
	for _, n := range notifications {
		if n.Pattern == "backstop" {
			backstopCount++
		}
	}
	if backstopCount != 1 {
		t.Errorf("Expected exactly 1 backstop notification, got %d", backstopCount)
	}

	// Wait longer to ensure no more backstop notifications
	time.Sleep(150 * time.Millisecond)

	// Should still have only one backstop notification
	notifications = mock.getNotifications()
	backstopCount = 0
	for _, n := range notifications {
		if n.Pattern == "backstop" {
			backstopCount++
		}
	}
	if backstopCount != 1 {
		t.Errorf("Expected exactly 1 backstop notification after waiting, got %d", backstopCount)
	}
}

func TestBackstopNotifier_ResetSession(t *testing.T) {
	mock := &testNotifier{}
	backstop := NewBackstopNotifier(mock, 50*time.Millisecond)
	defer func() { _ = backstop.Close() }()

	// Mark activity to start timer
	backstop.MarkActivity()

	// Wait for first backstop notification
	time.Sleep(100 * time.Millisecond)

	// Should have exactly one backstop notification
	notifications := mock.getNotifications()
	backstopCount := 0
	for _, n := range notifications {
		if n.Pattern == "backstop" {
			backstopCount++
		}
	}
	if backstopCount != 1 {
		t.Errorf("Expected exactly 1 backstop notification, got %d", backstopCount)
	}

	// Reset session (simulating new prompt)
	backstop.ResetSession()

	// Wait for another backstop notification
	time.Sleep(100 * time.Millisecond)

	// Should now have two backstop notifications (one from each session)
	notifications = mock.getNotifications()
	backstopCount = 0
	for _, n := range notifications {
		if n.Pattern == "backstop" {
			backstopCount++
		}
	}
	if backstopCount != 2 {
		t.Errorf("Expected 2 backstop notifications after reset, got %d", backstopCount)
	}
}

func TestBackstopNotifier_NoTimeoutNoBackstop(t *testing.T) {
	mock := &testNotifier{}
	backstop := NewBackstopNotifier(mock, 0) // No timeout
	defer func() { _ = backstop.Close() }()

	// Send a regular notification
	n := Notification{Title: "Test", Pattern: "test"}
	err := backstop.Send(n)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Should have only the sent notification, no backstop
	notifications := mock.getNotifications()
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}
	if notifications[0].Pattern == "backstop" {
		t.Error("Unexpected backstop notification with zero timeout")
	}
}
