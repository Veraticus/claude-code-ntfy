//go:build linux
// +build linux

package main

import (
	"syscall"
	"unsafe"
)

// isatty returns true if the given file descriptor is a terminal
func isatty(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&termios)), 0, 0, 0) // #nosec G103 -- Required for terminal detection
	return err == 0
}
