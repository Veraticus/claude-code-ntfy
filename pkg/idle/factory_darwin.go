//go:build darwin
// +build darwin

package idle

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// newPlatformDetector creates a Darwin-specific idle detector.
func newPlatformDetector() interfaces.IdleDetector {
	return NewDarwinIdleDetector()
}
