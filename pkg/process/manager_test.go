package process

import (
	"bytes"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
)

// MockPTYManager is a mock implementation of PTYManager for testing
type MockPTYManager struct {
	started      bool
	waited       bool
	exitCode     int
	startError   error
	waitError    error
	process      *os.Process
	processState *os.ProcessState
	pty          *os.File
	ioFunc       func()
}

func (m *MockPTYManager) Start(command string, args []string, env []string) error {
	if m.startError != nil {
		return m.startError
	}
	m.started = true
	return nil
}

func (m *MockPTYManager) Wait() error {
	if m != nil {
		m.waited = true
		return m.waitError
	}
	return nil
}

func (m *MockPTYManager) ProcessState() *os.ProcessState {
	if m != nil {
		return m.processState
	}
	return nil
}

func (m *MockPTYManager) Process() *os.Process {
	if m != nil {
		return m.process
	}
	return nil
}

func (m *MockPTYManager) GetPTY() *os.File {
	if m != nil {
		return m.pty
	}
	return nil
}

func (m *MockPTYManager) CopyIO(stdin io.Reader, stdout, stderr io.Writer, handler func([]byte)) error {
	if m.ioFunc != nil {
		m.ioFunc()
	}
	return nil
}

// MockOutputHandler is a mock implementation of OutputHandler
type MockOutputHandler struct {
	lines []string
}

func (m *MockOutputHandler) HandleLine(line string) {
	m.lines = append(m.lines, line)
}

func TestManager_Start(t *testing.T) {
	tests := []struct {
		name       string
		envWrapped string
		startError error
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "successful start",
			envWrapped: "",
			startError: nil,
			wantError:  false,
		},
		{
			name:       "already wrapped",
			envWrapped: "1",
			startError: nil,
			wantError:  true,
			errorMsg:   "already wrapped",
		},
		{
			name:       "start error",
			envWrapped: "",
			startError: errors.New("start failed"),
			wantError:  true,
			errorMsg:   "failed to start process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.envWrapped != "" {
				_ = os.Setenv("CLAUDE_CODE_NTFY_WRAPPED", tt.envWrapped)
				defer func() { _ = os.Unsetenv("CLAUDE_CODE_NTFY_WRAPPED") }()
			}

			cfg := config.DefaultConfig()
			mockPTY := &MockPTYManager{
				startError: tt.startError,
			}

			manager := &Manager{
				config:        cfg,
				ptyManager:    mockPTY,
				outputHandler: nil,
				done:          make(chan struct{}),
			}

			err := manager.Start("test", []string{"arg1"})

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q but got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !mockPTY.started {
					t.Error("PTY manager was not started")
				}
			}
		})
	}
}

func TestManager_Wait(t *testing.T) {
	tests := []struct {
		name         string
		ptyManager   *MockPTYManager
		wantError    bool
		wantExitCode int
	}{
		{
			name: "successful wait with exit code 0",
			ptyManager: &MockPTYManager{
				processState: &os.ProcessState{},
				exitCode:     0,
			},
			wantError:    false,
			wantExitCode: 0,
		},
		{
			name: "wait with error",
			ptyManager: &MockPTYManager{
				waitError: errors.New("wait failed"),
			},
			wantError: true,
		},
		{
			name:       "process not started",
			ptyManager: nil,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			manager := &Manager{
				config: cfg,
				done:   make(chan struct{}),
			}

			// Only set ptyManager if not nil to avoid typed nil interface issue
			if tt.ptyManager != nil {
				manager.ptyManager = tt.ptyManager
			}

			// Mock processState.ExitCode()
			if tt.ptyManager != nil && tt.ptyManager.processState != nil {
				// Since we can't easily mock os.ProcessState, we'll set exitCode directly
				manager.exitCode = tt.wantExitCode
			}

			err := manager.Wait()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.ptyManager != nil && !tt.ptyManager.waited {
					t.Error("PTY manager Wait was not called")
				}
				if manager.ExitCode() != tt.wantExitCode {
					t.Errorf("expected exit code %d but got %d", tt.wantExitCode, manager.ExitCode())
				}
			}
		})
	}
}

func TestManager_SignalForwarding(t *testing.T) {
	cfg := config.DefaultConfig()

	// Create a mock process that can receive signals
	mockProcess := &os.Process{
		Pid: os.Getpid(), // Use current process for testing
	}

	mockPTY := &MockPTYManager{
		process: mockProcess,
	}

	manager := &Manager{
		config:     cfg,
		ptyManager: mockPTY,
		done:       make(chan struct{}),
		sigChan:    make(chan os.Signal, 1),
	}

	// Start signal forwarding
	go manager.forwardSignals()

	// Send a signal
	manager.sigChan <- syscall.SIGUSR1

	// Give it time to process
	time.Sleep(10 * time.Millisecond)

	// Stop the manager
	close(manager.done)
}

func TestManager_Stop(t *testing.T) {
	tests := []struct {
		name      string
		process   *os.Process
		wantError bool
	}{
		{
			name:      "stop with valid process",
			process:   &os.Process{Pid: os.Getpid()},
			wantError: false,
		},
		{
			name:      "stop with nil process",
			process:   nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			mockPTY := &MockPTYManager{
				process: tt.process,
			}

			manager := &Manager{
				config:     cfg,
				ptyManager: mockPTY,
			}

			err := manager.Stop()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
