package process

import (
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
	cmd      *exec.Cmd
	pty      *os.File
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// Ensure PTYManager implements PTYInterface
var _ PTYInterface = (*PTYManager)(nil)

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
		p.pty.Close()
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
func (p *PTYManager) CopyIO(stdin io.Reader, stdout, stderr io.Writer, outputHandler func([]byte)) error {
	p.mu.Lock()
	if p.pty == nil {
		p.mu.Unlock()
		return fmt.Errorf("PTY not initialized")
	}
	p.mu.Unlock()

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
}

func (r *outputReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 && r.handler != nil {
		r.handler(p[:n])
	}
	return n, err
}
