//go:build linux
// +build linux

package idle

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// newPlatformDetector creates a Linux-specific idle detector.
func newPlatformDetector() interfaces.IdleDetector {
	return NewLinuxIdleDetector()
}
