//go:build !linux && !darwin
// +build !linux,!darwin

package idle

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// newPlatformDetector creates a fallback detector for unsupported platforms.
func newPlatformDetector() interfaces.IdleDetector {
	return NewOutputBasedDetector()
}
