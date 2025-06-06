package process

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPTYManager_StartAndWait(t *testing.T) {
	// Skip on CI or non-unix platforms
	if runtime.GOOS == "windows" || os.Getenv("CI") == "true" {
		t.Skip("PTY tests require Unix environment")
	}

	ptyMgr := NewPTYManager()

	// Start a simple echo command
	err := ptyMgr.Start("echo", []string{"hello world"}, os.Environ())
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Get PTY
	pty := ptyMgr.GetPTY()
	if pty == nil {
		t.Fatal("PTY is nil")
	}

	// Wait for completion
	err = ptyMgr.Wait()
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	// Check process state
	if ptyMgr.ProcessState() == nil {
		t.Error("ProcessState is nil")
	}
}

func TestPTYManager_CopyIO(t *testing.T) {
	// Skip on CI or non-unix platforms
	if runtime.GOOS == "windows" || os.Getenv("CI") == "true" {
		t.Skip("PTY tests require Unix environment")
	}

	ptyMgr := NewPTYManager()

	// Start a cat command that will echo input
	err := ptyMgr.Start("cat", []string{}, os.Environ())
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Prepare input/output
	input := bytes.NewBufferString("test input\n")
	output := &bytes.Buffer{}

	// Track output with handler
	var handlerCalled bool
	handler := func(data []byte) {
		handlerCalled = true
	}

	// Start copying in background
	done := make(chan error, 1)
	go func() {
		done <- ptyMgr.CopyIO(input, output, nil, handler, nil)
	}()

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Terminate the process
	if ptyMgr.Process() != nil {
		_ = ptyMgr.Process().Signal(syscall.SIGTERM)
	}

	// Wait for copy to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("CopyIO did not complete in time")
	}

	// Wait for process
	_ = ptyMgr.Wait()

	// Check that handler was called
	if !handlerCalled {
		t.Error("output handler was not called")
	}
}

func TestPTYManager_TerminalResize(t *testing.T) {
	// Skip on CI or non-unix platforms
	if runtime.GOOS == "windows" || os.Getenv("CI") == "true" {
		t.Skip("PTY tests require Unix environment")
	}

	ptyMgr := NewPTYManager()

	// Start a sleep command
	err := ptyMgr.Start("sleep", []string{"1"}, os.Environ())
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// The resize monitoring goroutine should be running
	// We can't easily test SIGWINCH handling, but we can verify the goroutine starts

	// Wait for completion
	err = ptyMgr.Wait()
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}
}

func TestPTYManager_StartErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "invalid command",
			command: "/nonexistent/command",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "valid command",
			command: "true",
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip on CI or non-unix platforms
			if runtime.GOOS == "windows" || os.Getenv("CI") == "true" {
				t.Skip("PTY tests require Unix environment")
			}

			ptyMgr := NewPTYManager()
			err := ptyMgr.Start(tt.command, tt.args, os.Environ())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Clean up
				_ = ptyMgr.Wait()
			}
		})
	}
}

func TestPTYManager_DoubleStart(t *testing.T) {
	// Skip on CI or non-unix platforms
	if runtime.GOOS == "windows" || os.Getenv("CI") == "true" {
		t.Skip("PTY tests require Unix environment")
	}

	ptyMgr := NewPTYManager()

	// First start
	err := ptyMgr.Start("sleep", []string{"1"}, os.Environ())
	if err != nil {
		t.Fatalf("first start failed: %v", err)
	}

	// Second start should fail
	err = ptyMgr.Start("echo", []string{"test"}, os.Environ())
	if err == nil {
		t.Error("expected error on second start")
	} else if !strings.Contains(err.Error(), "already started") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Clean up
	_ = ptyMgr.Process().Signal(syscall.SIGTERM)
	_ = ptyMgr.Wait()
}

func TestPTYManager_WaitWithoutStart(t *testing.T) {
	ptyMgr := NewPTYManager()

	err := ptyMgr.Wait()
	if err == nil {
		t.Error("expected error when waiting without start")
	} else if !strings.Contains(err.Error(), "not started") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPTYManager_ProcessMethods(t *testing.T) {
	ptyMgr := NewPTYManager()

	// Before start
	if ptyMgr.Process() != nil {
		t.Error("Process should be nil before start")
	}
	if ptyMgr.ProcessState() != nil {
		t.Error("ProcessState should be nil before start")
	}
	if ptyMgr.GetPTY() != nil {
		t.Error("PTY should be nil before start")
	}
}

// TestOutputReader tests the outputReader functionality
func TestOutputReader(t *testing.T) {
	// Create a pipe for testing
	r, w := io.Pipe()
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	// Track handler calls
	var handlerData [][]byte
	handler := func(data []byte) {
		// Make a copy since the slice may be reused
		cpy := make([]byte, len(data))
		copy(cpy, data)
		handlerData = append(handlerData, cpy)
	}

	// Create output reader
	reader := &outputReader{
		reader:  r,
		handler: handler,
	}

	// Write test data
	testData := []byte("test data")
	go func() {
		_, _ = w.Write(testData)
		_ = w.Close()
	}()

	// Read through the output reader
	result := make([]byte, len(testData))
	n, err := reader.Read(result)

	if err != nil && err != io.EOF {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("expected %d bytes but got %d", len(testData), n)
	}
	if !bytes.Equal(result[:n], testData) {
		t.Errorf("expected %q but got %q", testData, result[:n])
	}

	// Check handler was called
	if len(handlerData) != 1 {
		t.Errorf("expected 1 handler call but got %d", len(handlerData))
	} else if !bytes.Equal(handlerData[0], testData) {
		t.Errorf("handler got %q but expected %q", handlerData[0], testData)
	}
}
