package process

import (
	"io"
	"os"
)

// PTYInterface defines the interface for PTY operations
type PTYInterface interface {
	Start(command string, args []string, env []string) error
	Wait() error
	ProcessState() *os.ProcessState
	Process() *os.Process
	GetPTY() *os.File
	CopyIO(stdin io.Reader, stdout, stderr io.Writer, handler func([]byte)) error
}
