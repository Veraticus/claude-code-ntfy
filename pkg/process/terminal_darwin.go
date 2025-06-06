//go:build darwin
// +build darwin

package process

import "syscall"

const (
	ioctlReadTermios  = syscall.TIOCGETA
	ioctlWriteTermios = syscall.TIOCSETA
)
