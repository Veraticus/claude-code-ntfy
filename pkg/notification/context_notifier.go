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
		"✅",  // Checkmark
		"🤖",  // Robot emoji sometimes used
		"⚡",  // Lightning bolt
		"✨",  // Sparkles
		"🔮",  // Crystal ball
		"💫",  // Dizzy symbol
		"☁️", // Cloud
		"🌟",  // Star
	}

	// Remove any of the Claude icons from the beginning
	cleaned := title
	for _, icon := range claudeIcons {
		cleaned = strings.TrimPrefix(cleaned, icon)
		cleaned = strings.TrimPrefix(cleaned, icon+" ")
	}

	// Remove garbage/control characters at the beginning
	// This handles cases like "ÓÇ∂‚ú≥ Test Coverage"
	runes := []rune(cleaned)
	startIdx := 0
	
	// Skip any non-printable or garbage characters at the start
	for startIdx < len(runes) {
		r := runes[startIdx]
		// Keep ASCII letters, numbers, and common punctuation
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || 
		   (r >= '0' && r <= '9') || r == ' ' || r == '-' || 
		   r == '_' || r == '.' || r == '/' || r == '[' || r == ']' {
			break
		}
		startIdx++
	}
	
	if startIdx < len(runes) {
		cleaned = string(runes[startIdx:])
	} else {
		cleaned = ""
	}

	return strings.TrimSpace(cleaned)
}