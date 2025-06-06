package process

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// PTYManager handles PTY-based process execution
type PTYManager struct {
	cmd         *exec.Cmd
	pty         *os.File
	mu          sync.Mutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
	restoreFunc func()
}

// Ensure PTYManager implements PTY
var _ PTY = (*PTYManager)(nil)

// NewPTYManager creates a new PTY manager
func NewPTYManager() *PTYManager {
	return &PTYManager{
		stopChan: make(chan struct{}),
	}
}

// Start starts a process with PTY
func (p *PTYManager) Start(command string, args []string, env []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil {
		return fmt.Errorf("process already started")
	}

	// Create the command
	p.cmd = exec.Command(command, args...)
	p.cmd.Env = env

	// Start the command with a PTY
	var err error
	p.pty, err = pty.Start(p.cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}

	// Copy terminal size
	if err := p.copyTerminalSize(); err != nil {
		// Log but don't fail - some environments don't have a terminal
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: failed to copy terminal size: %v\n", err)
	}

	// Start monitoring for terminal size changes
	p.wg.Add(1)
	go p.monitorTerminalSize()

	return nil
}

// GetPTY returns the PTY file descriptor
func (p *PTYManager) GetPTY() *os.File {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pty
}

// Wait waits for the process to complete
func (p *PTYManager) Wait() error {
	if p.cmd == nil {
		return fmt.Errorf("process not started")
	}

	err := p.cmd.Wait()

	// Signal stop to goroutines
	close(p.stopChan)

	// Wait for goroutines
	p.wg.Wait()

	// Close PTY
	p.mu.Lock()
	if p.pty != nil {
		_ = p.pty.Close()
	}
	p.mu.Unlock()

	return err
}

// ProcessState returns the process state
func (p *PTYManager) ProcessState() *os.ProcessState {
	if p.cmd == nil {
		return nil
	}
	return p.cmd.ProcessState
}

// Process returns the underlying process
func (p *PTYManager) Process() *os.Process {
	if p.cmd == nil {
		return nil
	}
	return p.cmd.Process
}

// Stop gracefully stops the PTY manager and restores terminal state
func (p *PTYManager) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Restore terminal if needed
	if p.restoreFunc != nil {
		p.restoreFunc()
		p.restoreFunc = nil
	}

	return nil
}

// copyTerminalSize copies the terminal size from stdin to the PTY
func (p *PTYManager) copyTerminalSize() error {
	size, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return err
	}

	return pty.Setsize(p.pty, size)
}

// monitorTerminalSize monitors for terminal size changes
func (p *PTYManager) monitorTerminalSize() {
	defer p.wg.Done()

	// Create a channel for SIGWINCH signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-sigChan:
			p.mu.Lock()
			if p.pty != nil {
				if err := p.copyTerminalSize(); err != nil {
					fmt.Fprintf(os.Stderr, "claude-code-ntfy: failed to resize PTY: %v\n", err)
				}
			}
			p.mu.Unlock()
		case <-p.stopChan:
			return
		}
	}
}

// CopyIO handles copying between PTY and standard streams
func (p *PTYManager) CopyIO(stdin io.Reader, stdout, stderr io.Writer, outputHandler func([]byte), enableFocus bool) error {
	p.mu.Lock()
	if p.pty == nil {
		p.mu.Unlock()
		return fmt.Errorf("PTY not initialized")
	}
	ptyFile := p.pty
	p.mu.Unlock()

	// Store the restore function so we can call it from Stop()
	rawModeSet := false
	if file, ok := stdin.(*os.File); ok {
		if restore, err := setRawMode(int(file.Fd())); err == nil {
			rawModeSet = true
			p.mu.Lock()
			p.restoreFunc = restore
			p.mu.Unlock()
			defer func() {
				p.mu.Lock()
				if p.restoreFunc != nil {
					p.restoreFunc()
					p.restoreFunc = nil
				}
				p.mu.Unlock()
			}()
		}
	}

	// If raw mode was successfully set and focus detection is enabled,
	// send the enable focus reporting sequence
	if rawModeSet && enableFocus && outputHandler != nil {
		enableSeq := []byte("\033[?1004h")
		if _, err := ptyFile.Write(enableSeq); err == nil {
			if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "claude-code-ntfy: sent focus enable sequence\n")
			}
		}
	}

	// Use a wait group to track copy operations
	var wg sync.WaitGroup

	// Error channel to capture any errors
	errChan := make(chan error, 2)

	// Copy from stdin to PTY
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(p.pty, stdin); err != nil {
			errChan <- fmt.Errorf("stdin copy error: %w", err)
		}
	}()

	// Copy from PTY to stdout with optional output handling
	wg.Add(1)
	go func() {
		defer wg.Done()

		if outputHandler != nil {
			// Use a TeeReader to handle output
			reader := &outputReader{
				reader:  p.pty,
				handler: outputHandler,
			}
			if _, err := io.Copy(stdout, reader); err != nil {
				errChan <- fmt.Errorf("stdout copy error: %w", err)
			}
		} else {
			// Direct copy without handling
			if _, err := io.Copy(stdout, p.pty); err != nil {
				errChan <- fmt.Errorf("stdout copy error: %w", err)
			}
		}
	}()

	// Wait for copies to complete
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// outputReader wraps a reader and calls a handler for each chunk of data
type outputReader struct {
	reader  io.Reader
	handler func([]byte)
	buffer  []byte // Buffer for incomplete sequences
}

func (r *outputReader) Read(p []byte) (n int, err error) {
	// Read into a temporary buffer so we can filter
	tmpBuf := make([]byte, len(p))
	n, err = r.reader.Read(tmpBuf)
	if n > 0 {
		// Add to our internal buffer
		r.buffer = append(r.buffer, tmpBuf[:n]...)

		// Filter out focus reporting sequences
		filtered := r.filterFocusSequences(r.buffer)

		// Copy filtered data to output
		copy(p, filtered)
		resultLen := len(filtered)

		// Keep any incomplete sequences in buffer
		if len(filtered) < len(r.buffer) {
			// We filtered something out, clear the buffer
			r.buffer = r.buffer[:0]
		} else if len(r.buffer) > 1024 {
			// Prevent buffer from growing too large
			r.buffer = r.buffer[len(r.buffer)-100:]
		}

		// Call handler with the original unfiltered data
		if r.handler != nil {
			r.handler(tmpBuf[:n])
		}

		return resultLen, err
	}
	return n, err
}

// filterFocusSequences removes focus reporting escape sequences from data
func (r *outputReader) filterFocusSequences(data []byte) []byte {
	// Sequences to filter out
	focusSequences := [][]byte{
		[]byte("\033[?1004h"), // Enable focus reporting
		[]byte("\033[?1004l"), // Disable focus reporting
	}

	result := data
	for _, seq := range focusSequences {
		result = bytes.ReplaceAll(result, seq, []byte{})
	}

	return result
}
