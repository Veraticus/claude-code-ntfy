package process

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// Manager manages the wrapped Claude Code process
type Manager struct {
	config        *config.Config
	ptyManager    PTYInterface
	outputHandler interfaces.OutputHandler
	exitCode      int
	mu            sync.Mutex
	sigChan       chan os.Signal
	done          chan struct{}
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config, outputHandler interfaces.OutputHandler) *Manager {
	return &Manager{
		config:        cfg,
		ptyManager:    NewPTYManager(),
		outputHandler: outputHandler,
		done:          make(chan struct{}),
	}
}

// Start starts the Claude Code process
func (m *Manager) Start(command string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for self-wrap
	if os.Getenv("CLAUDE_CODE_NTFY_WRAPPED") == "1" {
		return fmt.Errorf("already wrapped by claude-code-ntfy")
	}

	// Set environment to prevent self-wrap
	env := append(os.Environ(), "CLAUDE_CODE_NTFY_WRAPPED=1")

	// Start the process with PTY
	if err := m.ptyManager.Start(command, args, env); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Start I/O copying with output handling
	go func() {
		var handler func([]byte)
		if m.outputHandler != nil {
			handler = func(data []byte) {
				// Check if output handler supports raw data handling
				type dataHandler interface {
					HandleData([]byte)
				}

				if dh, ok := m.outputHandler.(dataHandler); ok {
					// Use raw data handler
					dh.HandleData(data)
				} else {
					// Fall back to line-by-line processing
					lines := bytes.Split(data, []byte{'\n'})
					for _, line := range lines {
						if len(line) > 0 {
							m.outputHandler.HandleLine(string(line))
						}
					}
				}
			}
		}
		if err := m.ptyManager.CopyIO(os.Stdin, os.Stdout, os.Stderr, handler); err != nil {
			fmt.Fprintf(os.Stderr, "claude-code-ntfy: I/O error: %v\n", err)
		}
	}()

	// Setup signal forwarding
	m.setupSignalForwarding()

	return nil
}

// Wait waits for the process to exit
func (m *Manager) Wait() error {
	if m.ptyManager == nil {
		return fmt.Errorf("process not started")
	}

	err := m.ptyManager.Wait()

	m.mu.Lock()
	if m.ptyManager.ProcessState() != nil {
		m.exitCode = m.ptyManager.ProcessState().ExitCode()
	}
	m.mu.Unlock()

	// Signal that we're done
	close(m.done)

	// Cleanup signal handling
	m.cleanupSignals()

	return err
}

// ExitCode returns the exit code of the process
func (m *Manager) ExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

// setupSignalForwarding sets up signal forwarding to the child process
func (m *Manager) setupSignalForwarding() {
	m.sigChan = make(chan os.Signal, 1)
	signal.Notify(m.sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
	)

	go m.forwardSignals()
}

// forwardSignals forwards signals to the child process
func (m *Manager) forwardSignals() {
	for {
		select {
		case sig := <-m.sigChan:
			if m.ptyManager != nil && m.ptyManager.Process() != nil {
				// Forward the signal to the child process
				if err := m.ptyManager.Process().Signal(sig); err != nil {
					// Process might have already exited, but log it
					if !errors.Is(err, os.ErrProcessDone) {
						fmt.Fprintf(os.Stderr, "claude-code-ntfy: signal forward error: %v\n", err)
					}
				}
			}
		case <-m.done:
			return
		}
	}
}

// cleanupSignals stops signal forwarding
func (m *Manager) cleanupSignals() {
	if m.sigChan != nil {
		signal.Stop(m.sigChan)
		close(m.sigChan)
	}
}

// Stop gracefully stops the manager and cleans up resources
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptyManager != nil && m.ptyManager.Process() != nil {
		// Send SIGTERM first for graceful shutdown
		if err := m.ptyManager.Process().Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, force kill
			if !errors.Is(err, os.ErrProcessDone) {
				return m.ptyManager.Process().Kill()
			}
		}
	}

	return nil
}
