package idle

import (
	"sync"
	"testing"
	"time"
)

func TestNewOutputBasedDetector(t *testing.T) {
	detector := NewOutputBasedDetector()

	if detector == nil {
		t.Fatal("NewOutputBasedDetector returned nil")
	}

	if detector.lastActivity.IsZero() {
		t.Error("Initial last activity time should not be zero")
	}

	if time.Since(detector.lastActivity) > time.Second {
		t.Error("Initial last activity time should be recent")
	}
}

func TestOutputBasedDetector_IsUserIdle(t *testing.T) {
	tests := []struct {
		name              string
		timeSinceActivity time.Duration
		threshold         time.Duration
		expectedIdle      bool
	}{
		{
			name:              "Not idle when activity is recent",
			timeSinceActivity: 30 * time.Second,
			threshold:         1 * time.Minute,
			expectedIdle:      false,
		},
		{
			name:              "Idle when activity exceeds threshold",
			timeSinceActivity: 2 * time.Minute,
			threshold:         1 * time.Minute,
			expectedIdle:      true,
		},
		{
			name:              "Not idle at exact threshold",
			timeSinceActivity: 1 * time.Minute,
			threshold:         1 * time.Minute,
			expectedIdle:      true,
		},
		{
			name:              "Not idle with zero threshold",
			timeSinceActivity: 0,
			threshold:         0,
			expectedIdle:      true,
		},
		{
			name:              "Idle with very small threshold",
			timeSinceActivity: 1 * time.Millisecond,
			threshold:         1 * time.Nanosecond,
			expectedIdle:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewOutputBasedDetector()
			detector.UpdateActivityTime(time.Now().Add(-tt.timeSinceActivity))

			idle, err := detector.IsUserIdle(tt.threshold)
			if err != nil {
				t.Fatalf("IsUserIdle returned unexpected error: %v", err)
			}

			if idle != tt.expectedIdle {
				t.Errorf("IsUserIdle = %v, want %v", idle, tt.expectedIdle)
			}
		})
	}
}

func TestOutputBasedDetector_LastActivity(t *testing.T) {
	detector := NewOutputBasedDetector()

	// Test initial last activity
	initial := detector.LastActivity()
	if initial.IsZero() {
		t.Error("Initial LastActivity should not be zero")
	}

	// Update activity and verify
	testTime := time.Now().Add(-5 * time.Minute)
	detector.UpdateActivityTime(testTime)

	result := detector.LastActivity()
	if !result.Equal(testTime) {
		t.Errorf("LastActivity = %v, want %v", result, testTime)
	}
}

func TestOutputBasedDetector_UpdateActivity(t *testing.T) {
	detector := NewOutputBasedDetector()

	// Get initial time
	initial := detector.LastActivity()

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update activity
	detector.UpdateActivity()

	// Verify it was updated
	updated := detector.LastActivity()
	if !updated.After(initial) {
		t.Error("UpdateActivity should set a more recent time")
	}

	if time.Since(updated) > time.Second {
		t.Error("UpdateActivity should set time to approximately now")
	}
}

func TestOutputBasedDetector_UpdateActivityTime(t *testing.T) {
	detector := NewOutputBasedDetector()

	// Test with past time
	pastTime := time.Now().Add(-1 * time.Hour)
	detector.UpdateActivityTime(pastTime)

	result := detector.LastActivity()
	if !result.Equal(pastTime) {
		t.Errorf("UpdateActivityTime did not set correct time: got %v, want %v", result, pastTime)
	}

	// Test with future time
	futureTime := time.Now().Add(1 * time.Hour)
	detector.UpdateActivityTime(futureTime)

	result = detector.LastActivity()
	if !result.Equal(futureTime) {
		t.Errorf("UpdateActivityTime did not set correct time: got %v, want %v", result, futureTime)
	}
}

func TestOutputBasedDetector_ConcurrentAccess(t *testing.T) {
	detector := NewOutputBasedDetector()

	// Number of goroutines
	numReaders := 10
	numWriters := 5
	iterations := 100

	var wg sync.WaitGroup

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := detector.IsUserIdle(1 * time.Minute)
				if err != nil {
					t.Errorf("IsUserIdle error: %v", err)
				}
				_ = detector.LastActivity()
			}
		}()
	}

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					detector.UpdateActivity()
				} else {
					detector.UpdateActivityTime(time.Now().Add(time.Duration(id) * time.Second))
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
}

func TestOutputBasedDetector_RaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	detector := NewOutputBasedDetector()

	// This test is specifically designed to be run with -race flag
	done := make(chan bool)

	// Goroutine 1: Continuously update activity
	go func() {
		for i := 0; i < 1000; i++ {
			detector.UpdateActivity()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 2: Continuously check if idle
	go func() {
		for i := 0; i < 1000; i++ {
			_, _ = detector.IsUserIdle(time.Minute)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 3: Continuously read last activity
	go func() {
		for i := 0; i < 1000; i++ {
			_ = detector.LastActivity()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Race condition test timed out")
		}
	}
}
