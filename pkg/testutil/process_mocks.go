package testutil

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// MockPTYManager is a mock implementation of process.PTY for testing
type MockPTYManager struct {
	mu           sync.Mutex
	started      bool
	stopped      bool
	ptyName      string
	exitCode     int
	startErr     error
	waitErr      error
	outputBuffer *bytes.Buffer
	inputBuffer  *bytes.Buffer
}

// NewMockPTYManager creates a new mock PTY manager
func NewMockPTYManager() *MockPTYManager {
	return &MockPTYManager{
		ptyName:      "/dev/pts/mock",
		outputBuffer: &bytes.Buffer{},
		inputBuffer:  &bytes.Buffer{},
	}
}

// Start implements the PTY interface
func (m *MockPTYManager) Start(cmd string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.startErr != nil {
		return m.startErr
	}

	m.started = true
	return nil
}

// Stop implements the PTY interface
func (m *MockPTYManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopped = true
	return nil
}

// Wait implements the PTY interface
func (m *MockPTYManager) Wait() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.waitErr != nil {
		return m.waitErr
	}

	return nil
}

// GetOutput implements the PTY interface
func (m *MockPTYManager) GetOutput() io.Reader {
	m.mu.Lock()
	defer m.mu.Unlock()

	return bytes.NewReader(m.outputBuffer.Bytes())
}

// GetInput implements the PTY interface
func (m *MockPTYManager) GetInput() io.Writer {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.inputBuffer
}

// GetPTY implements the PTY interface
func (m *MockPTYManager) GetPTY() (*os.File, error) {
	return nil, nil
}

// Resize implements the PTY interface
func (m *MockPTYManager) Resize(rows, cols uint16) error {
	return nil
}

// GetPTYName implements the PTY interface
func (m *MockPTYManager) GetPTYName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ptyName
}

// ExitCode implements the PTY interface
func (m *MockPTYManager) ExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

// SetStartError sets the error to return from Start
func (m *MockPTYManager) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

// SetWaitError sets the error to return from Wait
func (m *MockPTYManager) SetWaitError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waitErr = err
}

// SetExitCode sets the exit code
func (m *MockPTYManager) SetExitCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exitCode = code
}

// WriteOutput writes data to the output buffer
func (m *MockPTYManager) WriteOutput(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputBuffer.Write(data)
}

// IsStarted returns whether Start was called
func (m *MockPTYManager) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

// IsStopped returns whether Stop was called
func (m *MockPTYManager) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// MockOutputHandler is a mock implementation of interfaces.OutputHandler for testing
type MockOutputHandler struct {
	mu              sync.Mutex
	handledOutput   []string
	handleCallCount int
	handleErr       error
}

// NewMockOutputHandler creates a new mock output handler
func NewMockOutputHandler() *MockOutputHandler {
	return &MockOutputHandler{
		handledOutput: []string{},
	}
}

// HandleOutput implements the OutputHandler interface
func (m *MockOutputHandler) HandleOutput(output string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handleCallCount++
	m.handledOutput = append(m.handledOutput, output)

	return m.handleErr
}

// GetHandledOutput returns all output that was handled
func (m *MockOutputHandler) GetHandledOutput() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, len(m.handledOutput))
	copy(result, m.handledOutput)
	return result
}

// GetHandleCallCount returns how many times HandleOutput was called
func (m *MockOutputHandler) GetHandleCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.handleCallCount
}

// SetHandleError sets the error to return from HandleOutput
func (m *MockOutputHandler) SetHandleError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handleErr = err
}
