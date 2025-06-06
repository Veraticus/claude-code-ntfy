//go:build linux || darwin
// +build linux darwin

package process

import (
	"syscall"
	"unsafe"
)

// setRawMode sets the terminal to raw mode and returns a restore function
func setRawMode(fd int) (func(), error) {
	var oldState syscall.Termios

	// Get current terminal settings
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 { //nolint:gosec // Required for terminal operations
		return nil, err
	}

	// Create new state with raw mode
	newState := oldState
	// This is equivalent to cfmakeraw() in C
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8

	// Set raw mode
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 { //nolint:gosec // Required for terminal operations
		return nil, err
	}

	// Return restore function
	return func() {
		// Best effort restore - we can't return an error from this function
		_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0) //nolint:gosec // Required for terminal operations
	}, nil
}
