//go:build linux
// +build linux

package process

import "syscall"

const (
	ioctlReadTermios  = syscall.TCGETS
	ioctlWriteTermios = syscall.TCSETS
)
