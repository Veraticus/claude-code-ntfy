package process

import (
	"io"
	"os"
)

// PTY defines the interface for PTY operations
type PTY interface {
	Start(command string, args []string, env []string) error
	Wait() error
	Stop() error
	ProcessState() *os.ProcessState
	Process() *os.Process
	GetPTY() *os.File
	CopyIO(stdin io.Reader, stdout, stderr io.Writer, outputHandler func([]byte), inputHandler func()) error
}
