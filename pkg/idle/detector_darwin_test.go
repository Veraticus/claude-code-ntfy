//go:build darwin
// +build darwin

package idle

import (
	"fmt"
	"testing"
	"time"
)

func TestNewDarwinIdleDetector(t *testing.T) {
	detector := NewDarwinIdleDetector()

	if detector == nil {
		t.Fatal("NewDarwinIdleDetector returned nil")
	}

	if detector.fallback == nil {
		t.Error("fallback detector should not be nil")
	}

	if detector.cmdExecutor == nil {
		t.Error("cmdExecutor should not be nil")
	}
}

func TestDarwinIdleDetector_parseHIDIdleTime(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedNanos int64
		expectError   bool
	}{
		{
			name: "Valid HIDIdleTime",
			input: `    | |   |   +-o IOHIDSystem  <class IOHIDSystem, id 0x1000002d0, registered, matched, active, busy 0 (0 ms), retain 22>
    | |   |     {
    | |   |       "HIDIdleTime" = 3456789012
    | |   |       "IOClass" = "IOHIDSystem"
    | |   |     }`,
			expectedNanos: 3456789012,
			expectError:   false,
		},
		{
			name: "HIDIdleTime with quotes",
			input: `    | |   |       "HIDIdleTime" = "1234567890"
    | |   |       "IOClass" = "IOHIDSystem"`,
			expectedNanos: 1234567890,
			expectError:   false,
		},
		{
			name: "Zero HIDIdleTime",
			input: `    | |   |       "HIDIdleTime" = 0
    | |   |       "IOClass" = "IOHIDSystem"`,
			expectedNanos: 0,
			expectError:   false,
		},
		{
			name:          "Missing HIDIdleTime",
			input:         `"IOClass" = "IOHIDSystem"`,
			expectedNanos: 0,
			expectError:   true,
		},
		{
			name: "Invalid HIDIdleTime format",
			input: `    | |   |       "HIDIdleTime" = "not-a-number"
    | |   |       "IOClass" = "IOHIDSystem"`,
			expectedNanos: 0,
			expectError:   true,
		},
		{
			name: "HIDIdleTime without equals",
			input: `    | |   |       "HIDIdleTime" 3456789012
    | |   |       "IOClass" = "IOHIDSystem"`,
			expectedNanos: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDarwinIdleDetector()
			result, err := detector.parseHIDIdleTime([]byte(tt.input))

			if (err != nil) != tt.expectError {
				t.Errorf("parseHIDIdleTime() error = %v, expectError %v", err, tt.expectError)
			}

			if result != tt.expectedNanos {
				t.Errorf("parseHIDIdleTime() = %v, want %v", result, tt.expectedNanos)
			}
		})
	}
}

func TestDarwinIdleDetector_getSystemIdleTime(t *testing.T) {
	tests := []struct {
		name             string
		mockOutput       []byte
		mockError        error
		expectedIdleTime time.Duration
		expectError      bool
	}{
		{
			name: "Valid idle time - 5 seconds",
			mockOutput: []byte(`    | |   |       "HIDIdleTime" = 5000000000
    | |   |       "IOClass" = "IOHIDSystem"`),
			mockError:        nil,
			expectedIdleTime: 5 * time.Second,
			expectError:      false,
		},
		{
			name: "Valid idle time - 2 minutes",
			mockOutput: []byte(`    | |   |       "HIDIdleTime" = 120000000000
    | |   |       "IOClass" = "IOHIDSystem"`),
			mockError:        nil,
			expectedIdleTime: 2 * time.Minute,
			expectError:      false,
		},
		{
			name:             "ioreg command error",
			mockOutput:       nil,
			mockError:        fmt.Errorf("ioreg not found"),
			expectedIdleTime: 0,
			expectError:      true,
		},
		{
			name:             "Invalid ioreg output",
			mockOutput:       []byte("invalid output"),
			mockError:        nil,
			expectedIdleTime: 0,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &DarwinIdleDetector{
				cmdExecutor: func(name string, args ...string) ([]byte, error) {
					if name != "ioreg" {
						t.Errorf("unexpected command: %s", name)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			idleTime, err := detector.getSystemIdleTime()

			if (err != nil) != tt.expectError {
				t.Errorf("getSystemIdleTime() error = %v, expectError %v", err, tt.expectError)
			}

			if idleTime != tt.expectedIdleTime {
				t.Errorf("getSystemIdleTime() = %v, want %v", idleTime, tt.expectedIdleTime)
			}
		})
	}
}

func TestDarwinIdleDetector_IsUserIdle(t *testing.T) {
	tests := []struct {
		name         string
		mockIdleTime time.Duration
		mockError    error
		fallbackIdle bool
		threshold    time.Duration
		expectedIdle bool
		useFallback  bool
	}{
		{
			name:         "System idle - exceeds threshold",
			mockIdleTime: 5 * time.Minute,
			mockError:    nil,
			fallbackIdle: false,
			threshold:    2 * time.Minute,
			expectedIdle: true,
			useFallback:  false,
		},
		{
			name:         "System not idle - below threshold",
			mockIdleTime: 30 * time.Second,
			mockError:    nil,
			fallbackIdle: false,
			threshold:    1 * time.Minute,
			expectedIdle: false,
			useFallback:  false,
		},
		{
			name:         "ioreg fails - use fallback",
			mockIdleTime: 0,
			mockError:    fmt.Errorf("ioreg error"),
			fallbackIdle: true,
			threshold:    1 * time.Minute,
			expectedIdle: true,
			useFallback:  true,
		},
		{
			name:         "Exact threshold",
			mockIdleTime: 1 * time.Minute,
			mockError:    nil,
			fallbackIdle: false,
			threshold:    1 * time.Minute,
			expectedIdle: true,
			useFallback:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &DarwinIdleDetector{
				cmdExecutor: func(name string, args ...string) ([]byte, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					output := fmt.Sprintf(`"HIDIdleTime" = %d`, tt.mockIdleTime.Nanoseconds())
					return []byte(output), nil
				},
				fallback: NewOutputBasedDetector(),
			}

			// Set up fallback state
			if tt.fallbackIdle {
				detector.fallback.UpdateActivityTime(time.Now().Add(-2 * time.Minute))
			} else {
				detector.fallback.UpdateActivityTime(time.Now())
			}

			idle, err := detector.IsUserIdle(tt.threshold)

			if err != nil {
				t.Errorf("IsUserIdle() unexpected error = %v", err)
			}

			if idle != tt.expectedIdle {
				t.Errorf("IsUserIdle() = %v, want %v", idle, tt.expectedIdle)
			}
		})
	}
}

func TestDarwinIdleDetector_LastActivity(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		mockIdleTime     time.Duration
		mockError        error
		fallbackActivity time.Time
		useFallback      bool
	}{
		{
			name:         "System idle time available",
			mockIdleTime: 3 * time.Minute,
			mockError:    nil,
			useFallback:  false,
		},
		{
			name:             "ioreg fails - use fallback",
			mockIdleTime:     0,
			mockError:        fmt.Errorf("ioreg error"),
			fallbackActivity: now.Add(-5 * time.Minute),
			useFallback:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &DarwinIdleDetector{
				cmdExecutor: func(name string, args ...string) ([]byte, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					output := fmt.Sprintf(`"HIDIdleTime" = %d`, tt.mockIdleTime.Nanoseconds())
					return []byte(output), nil
				},
				fallback: NewOutputBasedDetector(),
			}

			// Set up fallback state
			if tt.useFallback {
				detector.fallback.UpdateActivityTime(tt.fallbackActivity)
			}

			activity := detector.LastActivity()

			if tt.useFallback {
				// Check fallback was used
				if !activity.Equal(tt.fallbackActivity) {
					t.Errorf("LastActivity() = %v, want %v (fallback)", activity, tt.fallbackActivity)
				}
			} else {
				// Check system idle time was used
				expectedActivity := now.Add(-tt.mockIdleTime)
				diff := activity.Sub(expectedActivity)
				if diff < -1*time.Second || diff > 1*time.Second {
					t.Errorf("LastActivity() = %v, want approximately %v (diff: %v)",
						activity, expectedActivity, diff)
				}
			}
		})
	}
}

func TestDarwinIdleDetector_UpdateActivity(t *testing.T) {
	detector := NewDarwinIdleDetector()

	// Get initial fallback activity
	initialActivity := detector.fallback.LastActivity()

	// Sleep to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update activity
	detector.UpdateActivity()

	// Verify fallback was updated
	newActivity := detector.fallback.LastActivity()
	if !newActivity.After(initialActivity) {
		t.Error("UpdateActivity() should update fallback detector")
	}
}

func TestDarwinIdleDetector_UpdateActivityTime(t *testing.T) {
	detector := NewDarwinIdleDetector()

	testTime := time.Now().Add(-1 * time.Hour)
	detector.UpdateActivityTime(testTime)

	// Verify fallback was updated
	activity := detector.fallback.LastActivity()
	if !activity.Equal(testTime) {
		t.Errorf("UpdateActivityTime() did not update fallback: got %v, want %v",
			activity, testTime)
	}
}

func TestDarwinIdleDetector_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		whichErr error
		expected bool
	}{
		{
			name:     "ioreg available",
			whichErr: nil,
			expected: true,
		},
		{
			name:     "ioreg not available",
			whichErr: fmt.Errorf("command not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &DarwinIdleDetector{
				cmdExecutor: func(name string, args ...string) ([]byte, error) {
					if name == "which" && len(args) > 0 && args[0] == "ioreg" {
						if tt.whichErr != nil {
							return nil, tt.whichErr
						}
						return []byte("/usr/sbin/ioreg\n"), nil
					}
					return nil, fmt.Errorf("unexpected command")
				},
			}

			result := detector.IsAvailable()
			if result != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDarwinIdleDetector_Integration(t *testing.T) {
	// This test verifies the integration between ioreg and fallback detection
	detector := NewDarwinIdleDetector()

	ioregCalls := 0
	detector.cmdExecutor = func(name string, args ...string) ([]byte, error) {
		if name == "ioreg" {
			ioregCalls++
			// First call succeeds, subsequent calls fail
			if ioregCalls == 1 {
				return []byte(`"HIDIdleTime" = 180000000000`), nil // 3 minutes
			}
			return nil, fmt.Errorf("ioreg failed")
		}
		return nil, fmt.Errorf("unexpected command")
	}

	// Test 1: ioreg works, should report idle
	idle, err := detector.IsUserIdle(2 * time.Minute)
	if err != nil {
		t.Errorf("IsUserIdle() unexpected error: %v", err)
	}
	if !idle {
		t.Error("Should be idle when ioreg shows 3 minutes with 2 minute threshold")
	}

	// Test 2: Update fallback activity
	detector.UpdateActivity()

	// Test 3: ioreg fails, should use fallback (not idle)
	idle, err = detector.IsUserIdle(30 * time.Second)
	if err != nil {
		t.Errorf("IsUserIdle() unexpected error: %v", err)
	}
	if idle {
		t.Error("Should not be idle when fallback was just updated")
	}

	// Verify ioreg was called twice
	if ioregCalls != 2 {
		t.Errorf("Expected 2 ioreg calls, got %d", ioregCalls)
	}
}
