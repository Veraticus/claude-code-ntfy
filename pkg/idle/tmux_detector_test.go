package idle

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewTmuxIdleDetector(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
	}{
		{
			name:        "With session name",
			sessionName: "main",
		},
		{
			name:        "Without session name",
			sessionName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewTmuxIdleDetector(tt.sessionName)

			if detector == nil {
				t.Fatal("NewTmuxIdleDetector returned nil")
			}

			if detector.sessionName != tt.sessionName {
				t.Errorf("sessionName = %v, want %v", detector.sessionName, tt.sessionName)
			}

			if detector.cmdExecutor == nil {
				t.Error("cmdExecutor should not be nil")
			}
		})
	}
}

func TestTmuxIdleDetector_isInTmux(t *testing.T) {
	tests := []struct {
		name     string
		tmuxEnv  string
		expected bool
	}{
		{
			name:     "In tmux session",
			tmuxEnv:  "/tmp/tmux-1000/default,12345,0",
			expected: true,
		},
		{
			name:     "Not in tmux session",
			tmuxEnv:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original TMUX env var
			originalTmux := os.Getenv("TMUX")
			defer func() {
				_ = os.Setenv("TMUX", originalTmux)
			}()

			// Set test environment
			_ = os.Setenv("TMUX", tt.tmuxEnv)

			detector := NewTmuxIdleDetector("")
			result := detector.isInTmux()

			if result != tt.expected {
				t.Errorf("isInTmux() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTmuxIdleDetector_getCurrentSessionName(t *testing.T) {
	tests := []struct {
		name          string
		mockOutput    []byte
		mockError     error
		expectedName  string
		expectedError bool
	}{
		{
			name:          "Success",
			mockOutput:    []byte("main\n"),
			mockError:     nil,
			expectedName:  "main",
			expectedError: false,
		},
		{
			name:          "Success with trailing spaces",
			mockOutput:    []byte("  session-1  \n"),
			mockError:     nil,
			expectedName:  "session-1",
			expectedError: false,
		},
		{
			name:          "Command error",
			mockOutput:    nil,
			mockError:     fmt.Errorf("tmux not found"),
			expectedName:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &TmuxIdleDetector{
				cmdExecutor: func(name string, _ ...string) ([]byte, error) {
					if name != "tmux" {
						t.Errorf("unexpected command: %s", name)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			name, err := detector.getCurrentSessionName()

			if (err != nil) != tt.expectedError {
				t.Errorf("getCurrentSessionName() error = %v, expectedError %v", err, tt.expectedError)
			}

			if name != tt.expectedName {
				t.Errorf("getCurrentSessionName() = %v, want %v", name, tt.expectedName)
			}
		})
	}
}

func TestTmuxIdleDetector_getSessionIdleTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name            string
		sessionName     string
		mockOutput      []byte
		mockError       error
		expectedIdleMin time.Duration
		expectedIdleMax time.Duration
		expectedError   bool
	}{
		{
			name:            "Single client - 5 minutes idle",
			sessionName:     "main",
			mockOutput:      []byte(fmt.Sprintf("%d\n", now.Add(-5*time.Minute).Unix())),
			mockError:       nil,
			expectedIdleMin: 4 * time.Minute,
			expectedIdleMax: 6 * time.Minute,
			expectedError:   false,
		},
		{
			name:        "Multiple clients - most recent wins",
			sessionName: "main",
			mockOutput: []byte(fmt.Sprintf("%d\n%d\n%d\n",
				now.Add(-10*time.Minute).Unix(),
				now.Add(-2*time.Minute).Unix(),
				now.Add(-5*time.Minute).Unix())),
			mockError:       nil,
			expectedIdleMin: 1 * time.Minute,
			expectedIdleMax: 3 * time.Minute,
			expectedError:   false,
		},
		{
			name:            "Empty output",
			sessionName:     "main",
			mockOutput:      []byte(""),
			mockError:       nil,
			expectedIdleMin: 0,
			expectedIdleMax: 0,
			expectedError:   true,
		},
		{
			name:            "Invalid timestamp",
			sessionName:     "main",
			mockOutput:      []byte("invalid\n"),
			mockError:       nil,
			expectedIdleMin: 0,
			expectedIdleMax: 0,
			expectedError:   true,
		},
		{
			name:            "Command error",
			sessionName:     "main",
			mockOutput:      nil,
			mockError:       fmt.Errorf("session not found"),
			expectedIdleMin: 0,
			expectedIdleMax: 0,
			expectedError:   true,
		},
		{
			name:            "Future timestamp (clock skew)",
			sessionName:     "main",
			mockOutput:      []byte(fmt.Sprintf("%d\n", now.Add(1*time.Hour).Unix())),
			mockError:       nil,
			expectedIdleMin: 0,
			expectedIdleMax: 0,
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &TmuxIdleDetector{
				cmdExecutor: func(name string, _ ...string) ([]byte, error) {
					if name != "tmux" {
						t.Errorf("unexpected command: %s", name)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			idleTime, err := detector.getSessionIdleTime(tt.sessionName)

			if (err != nil) != tt.expectedError {
				t.Errorf("getSessionIdleTime() error = %v, expectedError %v", err, tt.expectedError)
			}

			if !tt.expectedError {
				if idleTime < tt.expectedIdleMin || idleTime > tt.expectedIdleMax {
					t.Errorf("getSessionIdleTime() = %v, want between %v and %v",
						idleTime, tt.expectedIdleMin, tt.expectedIdleMax)
				}
			}
		})
	}
}

func TestTmuxIdleDetector_IsUserIdle(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		_ = os.Setenv("TMUX", originalTmux)
	}()

	tests := []struct {
		name          string
		inTmux        bool
		sessionName   string
		mockSetup     func(*TmuxIdleDetector)
		threshold     time.Duration
		expectedIdle  bool
		expectedError bool
	}{
		{
			name:          "Not in tmux",
			inTmux:        false,
			sessionName:   "",
			mockSetup:     func(_ *TmuxIdleDetector) {},
			threshold:     1 * time.Minute,
			expectedIdle:  false,
			expectedError: true,
		},
		{
			name:        "In tmux - idle",
			inTmux:      true,
			sessionName: "main",
			mockSetup: func(d *TmuxIdleDetector) {
				d.cmdExecutor = func(_ string, args ...string) ([]byte, error) {
					if len(args) > 0 && args[0] == "display-message" {
						// getCurrentSessionName call
						return []byte("main\n"), nil
					}
					// getSessionIdleTime call - 5 minutes ago
					return []byte(fmt.Sprintf("%d\n", time.Now().Add(-5*time.Minute).Unix())), nil
				}
			},
			threshold:     2 * time.Minute,
			expectedIdle:  true,
			expectedError: false,
		},
		{
			name:        "In tmux - not idle",
			inTmux:      true,
			sessionName: "main",
			mockSetup: func(d *TmuxIdleDetector) {
				d.cmdExecutor = func(_ string, args ...string) ([]byte, error) {
					// getSessionIdleTime call - 30 seconds ago
					return []byte(fmt.Sprintf("%d\n", time.Now().Add(-30*time.Second).Unix())), nil
				}
			},
			threshold:     1 * time.Minute,
			expectedIdle:  false,
			expectedError: false,
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

			detector := NewTmuxIdleDetector(tt.sessionName)
			tt.mockSetup(detector)

			idle, err := detector.IsUserIdle(tt.threshold)

			if (err != nil) != tt.expectedError {
				t.Errorf("IsUserIdle() error = %v, expectedError %v", err, tt.expectedError)
			}

			if idle != tt.expectedIdle {
				t.Errorf("IsUserIdle() = %v, want %v", idle, tt.expectedIdle)
			}
		})
	}
}

func TestTmuxIdleDetector_LastActivity(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		_ = os.Setenv("TMUX", originalTmux)
	}()

	// Set in tmux
	_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")

	tests := []struct {
		name              string
		mockSetup         func(*TmuxIdleDetector)
		expectedRecent    bool
		expectedTimeRange time.Duration
	}{
		{
			name: "Success - returns activity time",
			mockSetup: func(d *TmuxIdleDetector) {
				d.cmdExecutor = func(_ string, args ...string) ([]byte, error) {
					// Return activity from 3 minutes ago
					return []byte(fmt.Sprintf("%d\n", time.Now().Add(-3*time.Minute).Unix())), nil
				}
			},
			expectedRecent:    false,
			expectedTimeRange: 3 * time.Minute,
		},
		{
			name: "Error - returns current time",
			mockSetup: func(d *TmuxIdleDetector) {
				d.cmdExecutor = func(_ string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("tmux error")
				}
			},
			expectedRecent:    true,
			expectedTimeRange: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewTmuxIdleDetector("main")
			tt.mockSetup(detector)

			activity := detector.LastActivity()
			elapsed := time.Since(activity)

			if tt.expectedRecent {
				if elapsed > 1*time.Second {
					t.Errorf("LastActivity() should return recent time, but got %v ago", elapsed)
				}
			} else {
				tolerance := 1 * time.Second
				if elapsed < tt.expectedTimeRange-tolerance || elapsed > tt.expectedTimeRange+tolerance {
					t.Errorf("LastActivity() = %v ago, want approximately %v ago",
						elapsed, tt.expectedTimeRange)
				}
			}
		})
	}
}

func TestTmuxIdleDetector_IsAvailable(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		_ = os.Setenv("TMUX", originalTmux)
	}()

	tests := []struct {
		name         string
		inTmux       bool
		tmuxCmdError error
		expected     bool
	}{
		{
			name:         "In tmux with tmux available",
			inTmux:       true,
			tmuxCmdError: nil,
			expected:     true,
		},
		{
			name:         "In tmux but tmux command fails",
			inTmux:       true,
			tmuxCmdError: fmt.Errorf("command not found"),
			expected:     false,
		},
		{
			name:         "Not in tmux",
			inTmux:       false,
			tmuxCmdError: nil,
			expected:     false,
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

			detector := &TmuxIdleDetector{
				cmdExecutor: func(name string, args ...string) ([]byte, error) {
					if name == "tmux" && len(args) > 0 && args[0] == "-V" {
						if tt.tmuxCmdError != nil {
							return nil, tt.tmuxCmdError
						}
						return []byte("tmux 3.2a\n"), nil
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
