//go:build linux
// +build linux

package idle

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewLinuxIdleDetector(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		_ = os.Setenv("TMUX", originalTmux)
	}()

	tests := []struct {
		name            string
		inTmux          bool
		tmuxAvailable   bool
		expectedUseTmux bool
	}{
		{
			name:            "In tmux with tmux available",
			inTmux:          true,
			tmuxAvailable:   true,
			expectedUseTmux: true,
		},
		{
			name:            "Not in tmux",
			inTmux:          false,
			tmuxAvailable:   true,
			expectedUseTmux: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set TMUX environment
			if tt.inTmux {
				_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
			} else {
				_ = os.Setenv("TMUX", "")
			}

			// Create detector with mocked tmux command
			detector := NewLinuxIdleDetector()

			// Override tmux detector's command executor for testing
			if detector.tmuxDetector != nil {
				detector.tmuxDetector.cmdExecutor = func(name string, args ...string) ([]byte, error) {
					if name == "tmux" && len(args) > 0 && args[0] == "-V" {
						if tt.tmuxAvailable {
							return []byte("tmux 3.2a\n"), nil
						}
						return nil, fmt.Errorf("command not found")
					}
					return nil, fmt.Errorf("unexpected command")
				}
			}

			// Re-check availability after setting up mock
			detector.useTmux = detector.tmuxDetector.IsAvailable()

			if detector.useTmux != tt.expectedUseTmux {
				t.Errorf("useTmux = %v, want %v", detector.useTmux, tt.expectedUseTmux)
			}

			if detector.tmuxDetector == nil {
				t.Error("tmuxDetector should not be nil")
			}

			if detector.fallback == nil {
				t.Error("fallback detector should not be nil")
			}
		})
	}
}

func TestLinuxIdleDetector_IsUserIdle(t *testing.T) {
	tests := []struct {
		name                string
		useTmux             bool
		tmuxIdle            bool
		tmuxError           error
		fallbackIdle        bool
		threshold           time.Duration
		expectedIdle        bool
		expectedUseFallback bool
	}{
		{
			name:                "Tmux detection success",
			useTmux:             true,
			tmuxIdle:            true,
			tmuxError:           nil,
			fallbackIdle:        false,
			threshold:           1 * time.Minute,
			expectedIdle:        true,
			expectedUseFallback: false,
		},
		{
			name:                "Tmux detection fails, use fallback",
			useTmux:             true,
			tmuxIdle:            false,
			tmuxError:           fmt.Errorf("tmux error"),
			fallbackIdle:        true,
			threshold:           1 * time.Minute,
			expectedIdle:        true,
			expectedUseFallback: true,
		},
		{
			name:                "Not using tmux, use fallback",
			useTmux:             false,
			tmuxIdle:            false,
			tmuxError:           nil,
			fallbackIdle:        true,
			threshold:           1 * time.Minute,
			expectedIdle:        true,
			expectedUseFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &LinuxIdleDetector{
				useTmux: tt.useTmux,
			}

			// Mock tmux detector
			mockTmuxCalled := false
			detector.tmuxDetector = &TmuxIdleDetector{
				cmdExecutor: func(_ string, _ ...string) ([]byte, error) {
					mockTmuxCalled = true
					if tt.tmuxError != nil {
						return nil, tt.tmuxError
					}
					// Return activity time based on whether idle
					if tt.tmuxIdle {
						return []byte(fmt.Sprintf("%d\n", time.Now().Add(-2*time.Minute).Unix())), nil
					}
					return []byte(fmt.Sprintf("%d\n", time.Now().Add(-30*time.Second).Unix())), nil
				},
			}

			// Mock fallback detector
			detector.fallback = NewOutputBasedDetector()
			if tt.fallbackIdle {
				detector.fallback.UpdateActivityTime(time.Now().Add(-2 * time.Minute))
			} else {
				detector.fallback.UpdateActivityTime(time.Now().Add(-30 * time.Second))
			}

			// Set up TMUX environment for tmux detector
			originalTmux := os.Getenv("TMUX")
			if tt.useTmux {
				_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
			}
			defer func() {
				_ = os.Setenv("TMUX", originalTmux)
			}()

			idle, err := detector.IsUserIdle(tt.threshold)

			if err != nil {
				t.Errorf("IsUserIdle() unexpected error = %v", err)
			}

			if idle != tt.expectedIdle {
				t.Errorf("IsUserIdle() = %v, want %v", idle, tt.expectedIdle)
			}

			// Verify which detector was used
			if tt.useTmux && !tt.expectedUseFallback && !mockTmuxCalled {
				t.Error("Expected tmux detector to be called but it wasn't")
			}
		})
	}
}

func TestLinuxIdleDetector_LastActivity(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name                 string
		useTmux              bool
		tmuxActivity         time.Time
		fallbackActivity     time.Time
		expectedActivity     time.Time
		expectedUsesFallback bool
	}{
		{
			name:                 "Tmux shows older activity",
			useTmux:              true,
			tmuxActivity:         now.Add(-5 * time.Minute),
			fallbackActivity:     now.Add(-1 * time.Minute),
			expectedActivity:     now.Add(-5 * time.Minute),
			expectedUsesFallback: false,
		},
		{
			name:                 "Fallback shows older activity",
			useTmux:              true,
			tmuxActivity:         now.Add(-1 * time.Minute),
			fallbackActivity:     now.Add(-5 * time.Minute),
			expectedActivity:     now.Add(-5 * time.Minute),
			expectedUsesFallback: true,
		},
		{
			name:                 "Not using tmux",
			useTmux:              false,
			tmuxActivity:         now.Add(-10 * time.Minute),
			fallbackActivity:     now.Add(-2 * time.Minute),
			expectedActivity:     now.Add(-2 * time.Minute),
			expectedUsesFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &LinuxIdleDetector{
				useTmux: tt.useTmux,
			}

			// Mock tmux detector
			detector.tmuxDetector = &TmuxIdleDetector{
				cmdExecutor: func(_ string, _ ...string) ([]byte, error) {
					// Return the tmux activity time
					return []byte(fmt.Sprintf("%d\n", tt.tmuxActivity.Unix())), nil
				},
			}

			// Set up fallback detector
			detector.fallback = NewOutputBasedDetector()
			detector.fallback.UpdateActivityTime(tt.fallbackActivity)

			// Set up TMUX environment
			originalTmux := os.Getenv("TMUX")
			if tt.useTmux {
				_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
			}
			defer func() {
				_ = os.Setenv("TMUX", originalTmux)
			}()

			activity := detector.LastActivity()

			// Allow for small time differences due to test execution
			diff := activity.Sub(tt.expectedActivity)
			if diff < -1*time.Second || diff > 1*time.Second {
				t.Errorf("LastActivity() = %v, want approximately %v (diff: %v)",
					activity, tt.expectedActivity, diff)
			}
		})
	}
}

func TestLinuxIdleDetector_UpdateActivity(t *testing.T) {
	detector := NewLinuxIdleDetector()

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

func TestLinuxIdleDetector_UpdateActivityTime(t *testing.T) {
	detector := NewLinuxIdleDetector()

	testTime := time.Now().Add(-1 * time.Hour)
	detector.UpdateActivityTime(testTime)

	// Verify fallback was updated
	activity := detector.fallback.LastActivity()
	if !activity.Equal(testTime) {
		t.Errorf("UpdateActivityTime() did not update fallback: got %v, want %v",
			activity, testTime)
	}
}

func TestLinuxIdleDetector_IsUsingTmux(t *testing.T) {
	tests := []struct {
		name     string
		useTmux  bool
		expected bool
	}{
		{
			name:     "Using tmux",
			useTmux:  true,
			expected: true,
		},
		{
			name:     "Not using tmux",
			useTmux:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &LinuxIdleDetector{
				useTmux: tt.useTmux,
			}

			result := detector.IsUsingTmux()
			if result != tt.expected {
				t.Errorf("IsUsingTmux() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLinuxIdleDetector_SetUseTmux(t *testing.T) {
	detector := NewLinuxIdleDetector()

	// Test disabling tmux
	detector.SetUseTmux(false)
	if detector.IsUsingTmux() {
		t.Error("SetUseTmux(false) should disable tmux detection")
	}

	// Test enabling tmux
	detector.SetUseTmux(true)
	if !detector.IsUsingTmux() {
		t.Error("SetUseTmux(true) should enable tmux detection")
	}
}

func TestLinuxIdleDetector_Integration(t *testing.T) {
	// This test verifies the integration between tmux and fallback detectors
	detector := NewLinuxIdleDetector()

	// Set up mocked tmux detector that tracks calls
	tmuxCalls := 0
	detector.tmuxDetector.cmdExecutor = func(name string, args ...string) ([]byte, error) {
		tmuxCalls++
		if len(args) > 0 && args[0] == "-V" {
			return []byte("tmux 3.2a\n"), nil
		}
		// Return activity from 3 minutes ago
		return []byte(fmt.Sprintf("%d\n", time.Now().Add(-3*time.Minute).Unix())), nil
	}

	// Set up environment
	originalTmux := os.Getenv("TMUX")
	_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		_ = os.Setenv("TMUX", originalTmux)
	}()

	// Enable tmux detection
	detector.SetUseTmux(true)

	// Test 1: When tmux shows idle, result should be idle
	idle, err := detector.IsUserIdle(2 * time.Minute)
	if err != nil {
		t.Errorf("IsUserIdle() unexpected error: %v", err)
	}
	if !idle {
		t.Error("Should be idle when tmux shows 3 minutes of inactivity with 2 minute threshold")
	}

	// Test 2: Update fallback activity
	detector.UpdateActivity()

	// Test 3: LastActivity should still return tmux time (older activity)
	// because tmux shows 3 minutes ago, which is older than the just-updated fallback
	activity := detector.LastActivity()
	expectedElapsed := 3 * time.Minute
	actualElapsed := time.Since(activity)
	tolerance := 1 * time.Second
	if actualElapsed < expectedElapsed-tolerance || actualElapsed > expectedElapsed+tolerance {
		t.Errorf("LastActivity() should return tmux time (~3 minutes ago), got %v ago", actualElapsed)
	}
}
