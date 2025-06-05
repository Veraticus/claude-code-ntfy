package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/notification"
)

func TestMockNotifier(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		mock := NewMockNotifier()
		n := notification.Notification{Title: "Test"}

		err := mock.Send(n)
		if err != nil {
			t.Errorf("Send() error = %v, want nil", err)
		}

		notifications := mock.GetNotifications()
		if len(notifications) != 1 {
			t.Errorf("GetNotifications() returned %d, want 1", len(notifications))
		}

		attempts := mock.GetAttempts()
		if len(attempts) != 1 {
			t.Errorf("GetAttempts() returned %d, want 1", len(attempts))
		}
	})

	t.Run("send with error", func(t *testing.T) {
		mock := NewMockNotifier()
		mockErr := errors.New("test error")
		mock.SetError(mockErr)

		n := notification.Notification{Title: "Test"}
		err := mock.Send(n)
		if err != mockErr {
			t.Errorf("Send() error = %v, want %v", err, mockErr)
		}

		// Should have no successful notifications
		notifications := mock.GetNotifications()
		if len(notifications) != 0 {
			t.Errorf("GetNotifications() returned %d, want 0", len(notifications))
		}

		// But should have an attempt
		attempts := mock.GetAttempts()
		if len(attempts) != 1 {
			t.Errorf("GetAttempts() returned %d, want 1", len(attempts))
		}
	})

	t.Run("clear state", func(t *testing.T) {
		mock := NewMockNotifier()
		_ = mock.Send(notification.Notification{Title: "Test"})
		mock.SetError(errors.New("error"))

		mock.Clear()

		if len(mock.GetNotifications()) != 0 {
			t.Error("Clear() did not reset notifications")
		}
		if len(mock.GetAttempts()) != 0 {
			t.Error("Clear() did not reset attempts")
		}

		// Error should be cleared
		err := mock.Send(notification.Notification{Title: "After clear"})
		if err != nil {
			t.Error("Clear() did not reset error")
		}
	})
}

func TestMockIdleDetector(t *testing.T) {
	t.Run("idle state", func(t *testing.T) {
		mock := NewMockIdleDetector(true)

		if !mock.IsIdle() {
			t.Error("IsIdle() = false, want true")
		}

		mock.SetIdle(false)
		if mock.IsIdle() {
			t.Error("IsIdle() = true, want false")
		}
	})

	t.Run("last activity", func(t *testing.T) {
		mock := NewMockIdleDetector(false)
		testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		mock.SetLastActivity(testTime)
		if !mock.LastActivity().Equal(testTime) {
			t.Errorf("LastActivity() = %v, want %v", mock.LastActivity(), testTime)
		}
	})

	t.Run("call counting", func(t *testing.T) {
		mock := NewMockIdleDetector(true)

		mock.IsIdle()
		mock.IsIdle()
		mock.IsIdle()

		if mock.GetIsIdleCallCount() != 3 {
			t.Errorf("GetIsIdleCallCount() = %d, want 3", mock.GetIsIdleCallCount())
		}
	})
}

func TestMockRateLimiter(t *testing.T) {
	t.Run("allow behavior", func(t *testing.T) {
		mock := NewMockRateLimiter(true)

		if !mock.Allow() {
			t.Error("Allow() = false, want true")
		}

		mock.SetAllowResult(false)
		if mock.Allow() {
			t.Error("Allow() = true, want false")
		}
	})

	t.Run("call counting", func(t *testing.T) {
		mock := NewMockRateLimiter(true)

		mock.Allow()
		mock.Allow()
		mock.Reset()
		mock.Allow()

		if mock.GetAllowCount() != 3 {
			t.Errorf("GetAllowCount() = %d, want 3", mock.GetAllowCount())
		}
		if mock.GetResetCount() != 1 {
			t.Errorf("GetResetCount() = %d, want 1", mock.GetResetCount())
		}
	})
}

func TestCountingRateLimiter(t *testing.T) {
	t.Run("allows up to limit", func(t *testing.T) {
		limiter := NewCountingRateLimiter(3)

		// First 3 should be allowed
		for i := 0; i < 3; i++ {
			if !limiter.Allow() {
				t.Errorf("Call %d: Allow() = false, want true", i+1)
			}
		}

		// 4th should be denied
		if limiter.Allow() {
			t.Error("Call 4: Allow() = true, want false")
		}
	})

	t.Run("reset restores capacity", func(t *testing.T) {
		limiter := NewCountingRateLimiter(2)

		limiter.Allow()
		limiter.Allow()

		// Should be denied
		if limiter.Allow() {
			t.Error("Before reset: Allow() = true, want false")
		}

		limiter.Reset()

		// Should be allowed again
		if !limiter.Allow() {
			t.Error("After reset: Allow() = false, want true")
		}
	})
}
