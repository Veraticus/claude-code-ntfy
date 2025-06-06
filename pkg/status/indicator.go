package status

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// Status represents the current notification status
type Status int

const (
	StatusIdle Status = iota
	StatusSending
	StatusSuccess
	StatusFailed
)

// Indicator manages the status display in the terminal
type Indicator struct {
	mu       sync.Mutex
	status   Status
	lastSent time.Time
	enabled  bool
	writer   io.Writer

	// Idle/focus state tracking
	isIdle           bool
	isFocused        bool
	focusReportingOn bool

	// Activity tracking for dynamic refresh
	lastActivity time.Time
	refreshChan  chan struct{}
}

// NewIndicator creates a new status indicator
func NewIndicator(writer io.Writer, enabled bool) *Indicator {
	return &Indicator{
		status:      StatusIdle,
		writer:      writer,
		enabled:     enabled,
		isFocused:   true, // Assume focused by default
		refreshChan: make(chan struct{}, 1),
	}
}

// SetStatus updates the current status
func (i *Indicator) SetStatus(status Status) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.status = status
	if status == StatusSuccess {
		i.lastSent = time.Now()
	}

	// Best effort - don't fail if we can't update the display
	_ = i.draw()
}

// draw renders the status indicator
func (i *Indicator) draw() error {
	if !i.enabled || i.writer == nil {
		return nil
	}

	statusText := i.getStatusText()
	if statusText == "" {
		// If there's no status to show, don't do anything
		return nil
	}

	// Use DEC save/restore cursor (\0337/\0338) instead of standard (\033[s/\033[u)
	// because it's more widely supported and reliable across different terminals.
	// Also, we need to handle the case where going to line 999 might cause scrolling.
	//
	// Strategy:
	// 1. Save current cursor position and attributes with DECSC
	// 2. Save scroll region state
	// 3. Move to last line without scrolling
	// 4. Write our status
	// 5. Restore everything
	//
	// The sequence breaks down as:
	// \0337 - DECSC: Save cursor position and attributes
	// \033[r - Reset scroll region to full screen
	// \033[999;1H - Move to line 999, column 1 (will be clamped to actual last line)
	// \033[2K - Clear entire line
	// %s - Our status text
	// \0338 - DECRC: Restore cursor position and attributes
	sequence := fmt.Sprintf("\0337\033[r\033[999;1H\033[2K%s\0338", statusText)

	if _, err := fmt.Fprint(i.writer, sequence); err != nil {
		return err
	}

	return nil
}

// getStatusText returns the appropriate status text with color
func (i *Indicator) getStatusText() string {
	var parts []string

	// Add idle/focus state indicator
	if i.focusReportingOn {
		if i.isFocused {
			parts = append(parts, "\033[36m◉\033[0m") // Cyan filled circle for focused
		} else {
			parts = append(parts, "\033[90m○\033[0m") // Gray empty circle for unfocused
		}
	}

	if i.isIdle {
		parts = append(parts, "\033[33mⓏ\033[0m") // Yellow Z for idle
	} else {
		parts = append(parts, "\033[32m▶\033[0m") // Green play for active
	}

	// Add ntfy status
	ntfyStatus := ""
	switch i.status {
	case StatusSending:
		ntfyStatus = "\033[33m⟳ ntfy\033[0m" // Yellow spinning arrow
	case StatusSuccess:
		ntfyStatus = "\033[32m✓ ntfy\033[0m" // Green checkmark
	case StatusFailed:
		ntfyStatus = "\033[31m✗ ntfy\033[0m" // Red X
	}

	if ntfyStatus != "" {
		parts = append(parts, ntfyStatus)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// Clear removes the status indicator
func (i *Indicator) Clear() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.enabled || i.writer == nil {
		return nil
	}

	// Clear the status line using DEC save/restore
	sequence := "\0337\033[999;1H\033[2K\0338"
	if _, err := fmt.Fprint(i.writer, sequence); err != nil {
		return err
	}

	return nil
}

// StartAutoRefresh starts a goroutine that refreshes the display periodically
func (i *Indicator) StartAutoRefresh(stopChan <-chan struct{}) {
	go func() {
		// Use dynamic refresh intervals
		normalInterval := 2 * time.Second
		activeInterval := 100 * time.Millisecond

		ticker := time.NewTicker(normalInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				i.mu.Lock()
				// Check if we've had recent activity
				isActive := time.Since(i.lastActivity) < 500*time.Millisecond
				_ = i.draw() // Best effort
				i.mu.Unlock()

				// Adjust ticker interval based on activity
				if isActive {
					ticker.Reset(activeInterval)
				} else {
					ticker.Reset(normalInterval)
				}
			case <-i.refreshChan:
				// Immediate refresh requested
				i.mu.Lock()
				_ = i.draw()
				i.mu.Unlock()
			case <-stopChan:
				_ = i.Clear() // Best effort
				return
			}
		}
	}()
}

// HandleScreenClear implements interfaces.ScreenEventHandler
// It redraws the status indicator when the screen is cleared
func (i *Indicator) HandleScreenClear() {
	i.mu.Lock()
	i.lastActivity = time.Now()
	i.mu.Unlock()

	// Always redraw when screen is cleared or interfered with
	// Don't check status == StatusIdle, always redraw to maintain visibility
	if i.enabled {
		// Trigger immediate refresh
		select {
		case i.refreshChan <- struct{}{}:
		default:
			// Channel is full, refresh already pending
		}
	}
}

// HandleTitleChange implements ScreenEventHandler
func (i *Indicator) HandleTitleChange(title string) {
	// No-op for status indicator
}

// HandleFocusIn implements ScreenEventHandler
func (i *Indicator) HandleFocusIn() {
	i.mu.Lock()
	i.isFocused = true
	i.mu.Unlock()
	_ = i.draw()
}

// HandleFocusOut implements ScreenEventHandler
func (i *Indicator) HandleFocusOut() {
	i.mu.Lock()
	i.isFocused = false
	i.mu.Unlock()
	_ = i.draw()
}

// SetIdleState updates the idle state
func (i *Indicator) SetIdleState(isIdle bool) {
	i.mu.Lock()
	i.isIdle = isIdle
	i.mu.Unlock()
	_ = i.draw()
}

// SetFocusReportingEnabled updates whether focus reporting is enabled
func (i *Indicator) SetFocusReportingEnabled(enabled bool) {
	i.mu.Lock()
	i.focusReportingOn = enabled
	i.mu.Unlock()
	_ = i.draw()
}

// MarkActivity marks that there has been recent activity
func (i *Indicator) MarkActivity() {
	i.mu.Lock()
	i.lastActivity = time.Now()
	i.mu.Unlock()
}

// Ensure Indicator implements ScreenEventHandler
var _ interfaces.ScreenEventHandler = (*Indicator)(nil)
