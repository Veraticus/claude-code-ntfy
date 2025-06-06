package status

import (
	"fmt"
	"io"
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
}

// NewIndicator creates a new status indicator
func NewIndicator(writer io.Writer, enabled bool) *Indicator {
	return &Indicator{
		status:  StatusIdle,
		writer:  writer,
		enabled: enabled,
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

	// Use a simpler approach that's less likely to interfere:
	// Save cursor position, move to bottom left, write status, restore
	sequence := fmt.Sprintf("\033[s\033[999;1H\033[K%s\033[u", statusText)

	if _, err := fmt.Fprint(i.writer, sequence); err != nil {
		return err
	}

	return nil
}

// getStatusText returns the appropriate status text with color
func (i *Indicator) getStatusText() string {
	switch i.status {
	case StatusSending:
		return "\033[33m⟳ ntfy\033[0m" // Yellow spinning arrow
	case StatusSuccess:
		return "\033[32m✓ ntfy\033[0m" // Green checkmark
	case StatusFailed:
		return "\033[31m✗ ntfy\033[0m" // Red X
	default:
		return ""
	}
}

// Clear removes the status indicator
func (i *Indicator) Clear() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.enabled || i.writer == nil {
		return nil
	}

	// Clear the status line
	sequence := "\033[s\033[999;1H\033[K\033[u"
	if _, err := fmt.Fprint(i.writer, sequence); err != nil {
		return err
	}

	return nil
}

// StartAutoRefresh starts a goroutine that refreshes the display periodically
func (i *Indicator) StartAutoRefresh(stopChan <-chan struct{}) {
	go func() {
		// Use a longer refresh interval to reduce interference
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				i.mu.Lock()
				_ = i.draw() // Best effort
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
	defer i.mu.Unlock()

	// Only redraw if we have an active status
	if i.enabled && i.status != StatusIdle {
		_ = i.draw() // Best effort
	}
}

// Ensure Indicator implements ScreenEventHandler
var _ interfaces.ScreenEventHandler = (*Indicator)(nil)
