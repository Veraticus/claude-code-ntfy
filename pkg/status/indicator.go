package status

import (
	"fmt"
	"io"
	"sync"
	"time"
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
	
	// Save cursor position
	if _, err := fmt.Fprint(i.writer, "\033[s"); err != nil {
		return err
	}
	
	// Move to bottom left corner
	if _, err := fmt.Fprint(i.writer, "\033[999;1H"); err != nil {
		return err
	}
	
	// Clear the line
	if _, err := fmt.Fprint(i.writer, "\033[K"); err != nil {
		return err
	}
	
	// Draw status
	statusText := i.getStatusText()
	if statusText != "" {
		if _, err := fmt.Fprint(i.writer, statusText); err != nil {
			return err
		}
	}
	
	// Restore cursor position
	if _, err := fmt.Fprint(i.writer, "\033[u"); err != nil {
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
		timeSince := time.Since(i.lastSent)
		if timeSince < 5*time.Second {
			return "\033[32m✓ ntfy\033[0m" // Green checkmark
		} else if timeSince < 30*time.Second {
			return fmt.Sprintf("\033[90m✓ ntfy (%ds)\033[0m", int(timeSince.Seconds())) // Gray with time
		}
		// After 30 seconds, show nothing
		return ""
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
	
	// Save cursor position
	if _, err := fmt.Fprint(i.writer, "\033[s"); err != nil {
		return err
	}
	
	// Move to bottom left corner
	if _, err := fmt.Fprint(i.writer, "\033[999;1H"); err != nil {
		return err
	}
	
	// Clear the line
	if _, err := fmt.Fprint(i.writer, "\033[K"); err != nil {
		return err
	}
	
	// Restore cursor position
	if _, err := fmt.Fprint(i.writer, "\033[u"); err != nil {
		return err
	}
	
	return nil
}

// StartAutoRefresh starts a goroutine that refreshes the display periodically
func (i *Indicator) StartAutoRefresh(stopChan <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
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