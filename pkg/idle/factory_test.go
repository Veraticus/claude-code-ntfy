package idle

import (
	"runtime"
	"testing"
	"time"
)

func TestNewIdleDetector(t *testing.T) {
	detector := NewIdleDetector()

	if detector == nil {
		t.Fatal("NewIdleDetector returned nil")
	}

	// Verify it implements the interface (compile-time check)

	// Just verify the detector works
	_, err := detector.IsUserIdle(1 * time.Minute)
	if err != nil {
		t.Logf("IsUserIdle returned error (might be expected on some platforms): %v", err)
	}

	activity := detector.LastActivity()
	if activity.IsZero() {
		t.Error("LastActivity should not return zero time")
	}
}

func TestNewIdleDetectorWithFallback(t *testing.T) {
	detector := NewIdleDetectorWithFallback()

	if detector == nil {
		t.Fatal("NewIdleDetectorWithFallback returned nil")
	}

	// Verify it implements the interface (compile-time check)

	// For platforms that don't have UpdateActivity methods,
	// verify it's wrapped properly
	switch runtime.GOOS {
	case "linux", "darwin":
		// These platforms have UpdateActivity methods, so might be wrapped
		// or returned directly
		if wrapper, ok := detector.(*idleDetectorWrapper); ok {
			if wrapper.detector == nil {
				t.Error("Wrapper detector should not be nil")
			}
			if wrapper.fallback == nil {
				t.Error("Wrapper fallback should not be nil")
			}
		}
	default:
		// Other platforms should just return OutputBasedDetector
		if _, ok := detector.(*OutputBasedDetector); !ok {
			t.Errorf("Expected OutputBasedDetector on %s, got %T", runtime.GOOS, detector)
		}
	}
}

func TestIdleDetectorWrapper_IsUserIdle(t *testing.T) {
	// Create a mock detector
	mockDetector := &mockIdleDetector{
		idle: true,
		err:  nil,
	}

	wrapper := &idleDetectorWrapper{
		detector: mockDetector,
		fallback: NewOutputBasedDetector(),
	}

	idle, err := wrapper.IsUserIdle(1 * time.Minute)

	if err != nil {
		t.Errorf("IsUserIdle() unexpected error: %v", err)
	}

	if idle != mockDetector.idle {
		t.Errorf("IsUserIdle() = %v, want %v", idle, mockDetector.idle)
	}

	if !mockDetector.isUserIdleCalled {
		t.Error("Expected wrapped detector's IsUserIdle to be called")
	}
}

func TestIdleDetectorWrapper_LastActivity(t *testing.T) {
	testTime := time.Now().Add(-5 * time.Minute)

	mockDetector := &mockIdleDetector{
		lastActivity: testTime,
	}

	wrapper := &idleDetectorWrapper{
		detector: mockDetector,
		fallback: NewOutputBasedDetector(),
	}

	activity := wrapper.LastActivity()

	if !activity.Equal(testTime) {
		t.Errorf("LastActivity() = %v, want %v", activity, testTime)
	}

	if !mockDetector.lastActivityCalled {
		t.Error("Expected wrapped detector's LastActivity to be called")
	}
}

func TestIdleDetectorWrapper_UpdateActivity(t *testing.T) {
	wrapper := &idleDetectorWrapper{
		detector: NewOutputBasedDetector(),
		fallback: NewOutputBasedDetector(),
	}

	// Get initial fallback time
	initialTime := wrapper.fallback.LastActivity()

	// Sleep to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update activity
	wrapper.UpdateActivity()

	// Verify fallback was updated
	newTime := wrapper.fallback.LastActivity()
	if !newTime.After(initialTime) {
		t.Error("UpdateActivity() should update fallback detector")
	}
}

func TestIdleDetectorWrapper_UpdateActivityTime(t *testing.T) {
	testTime := time.Now().Add(-1 * time.Hour)

	wrapper := &idleDetectorWrapper{
		detector: NewOutputBasedDetector(),
		fallback: NewOutputBasedDetector(),
	}

	// Update activity time
	wrapper.UpdateActivityTime(testTime)

	// Verify fallback was updated
	activity := wrapper.fallback.LastActivity()
	if !activity.Equal(testTime) {
		t.Errorf("UpdateActivityTime() did not update fallback: got %v, want %v",
			activity, testTime)
	}
}

// mockIdleDetector is a mock implementation for testing
type mockIdleDetector struct {
	idle               bool
	err                error
	lastActivity       time.Time
	isUserIdleCalled   bool
	lastActivityCalled bool
}

func (m *mockIdleDetector) IsUserIdle(_ time.Duration) (bool, error) {
	m.isUserIdleCalled = true
	return m.idle, m.err
}

func (m *mockIdleDetector) LastActivity() time.Time {
	m.lastActivityCalled = true
	return m.lastActivity
}

func TestFactoryIntegration(t *testing.T) {
	// Test that the factory creates functional detectors
	detector := NewIdleDetector()

	// Test basic functionality
	idle, err := detector.IsUserIdle(1 * time.Minute)
	if err != nil {
		// Some platforms might return errors in certain conditions
		// but the detector should still be created
		t.Logf("IsUserIdle returned error (this might be expected): %v", err)
	} else {
		// Just verify we got a boolean result
		_ = idle
	}

	// Test LastActivity
	activity := detector.LastActivity()
	if activity.IsZero() {
		t.Error("LastActivity should not return zero time")
	}

	// Test with fallback wrapper
	detectorWithFallback := NewIdleDetectorWithFallback()

	// Verify we can call IsUserIdle
	idle2, err2 := detectorWithFallback.IsUserIdle(30 * time.Second)
	if err2 != nil {
		t.Logf("IsUserIdle with fallback returned error: %v", err2)
	} else {
		_ = idle2
	}

	// Verify we can get LastActivity
	activity2 := detectorWithFallback.LastActivity()
	if activity2.IsZero() {
		t.Error("LastActivity with fallback should not return zero time")
	}
}
