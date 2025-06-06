package notification

import (
	"os"
	"path/filepath"
	"strings"
)

// ContextNotifier wraps another notifier and adds context to notifications
type ContextNotifier struct {
	underlying   Notifier
	cwdBasename  string
	terminalInfo func() string
}

// NewContextNotifier creates a new context notifier
func NewContextNotifier(underlying Notifier, terminalInfo func() string) *ContextNotifier {
	// Get CWD basename
	cwd, err := os.Getwd()
	cwdBasename := ""
	if err == nil {
		cwdBasename = filepath.Base(cwd)
	}
	
	return &ContextNotifier{
		underlying:   underlying,
		cwdBasename:  cwdBasename,
		terminalInfo: terminalInfo,
	}
}

// Send implements the Notifier interface
func (cn *ContextNotifier) Send(notification Notification) error {
	// Add context to title
	context := cn.cwdBasename
	
	// Get terminal title if available
	if cn.terminalInfo != nil {
		if title := cn.terminalInfo(); title != "" {
			// Parse out the Claude icon and clean up the title
			cleanTitle := cn.cleanTerminalTitle(title)
			if cleanTitle != "" && cleanTitle != "claude" {
				if context != "" {
					context = context + " - " + cleanTitle
				} else {
					context = cleanTitle
				}
			}
		}
	}
	
	// Replace notification title with context if available
	if context != "" {
		notification.Title = "Claude Code: " + context
	}
	
	// Forward to underlying notifier
	return cn.underlying.Send(notification)
}

// cleanTerminalTitle removes the Claude icon and cleans up the title
func (cn *ContextNotifier) cleanTerminalTitle(title string) string {
	// Common Claude icon patterns (various Unicode representations)
	claudeIcons := []string{
		"✅", // The icon you showed
		"🤖", // Robot emoji sometimes used
		"⚡", // Lightning bolt
		"✨", // Sparkles
		"🔮", // Crystal ball
		"💫", // Dizzy symbol
		"☁️", // Cloud
		"🌟", // Star
	}
	
	// Remove any of the Claude icons from the beginning
	cleaned := title
	for _, icon := range claudeIcons {
		cleaned = strings.TrimPrefix(cleaned, icon)
		cleaned = strings.TrimPrefix(cleaned, icon + " ")
	}
	
	// Also try to remove any non-ASCII character at the beginning followed by space
	// This catches other Unicode symbols we might not have listed
	if len(cleaned) > 0 {
		runes := []rune(cleaned)
		if len(runes) > 1 && runes[0] > 127 && runes[1] == ' ' {
			cleaned = string(runes[2:])
		}
	}
	
	return strings.TrimSpace(cleaned)
}