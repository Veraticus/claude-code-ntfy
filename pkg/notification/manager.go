package notification

import (
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// Manager orchestrates notification sending with batching and rate limiting
type Manager struct {
	config         *config.Config
	notifier       Notifier
	rateLimiter    interfaces.RateLimiter
	batcher        *Batcher
	statusReporter interfaces.StatusReporter

	mu sync.Mutex
}

// NewManager creates a new notification manager
func NewManager(cfg *config.Config, notifier Notifier, rateLimiter interfaces.RateLimiter) *Manager {
	m := &Manager{
		config:      cfg,
		notifier:    notifier,
		rateLimiter: rateLimiter,
	}

	// Create batcher if batch window is configured
	if cfg.BatchWindow > 0 {
		m.batcher = NewBatcher(cfg.BatchWindow, m.sendBatch)
	}

	return m
}

// SetStatusReporter sets the status reporter for the manager
func (m *Manager) SetStatusReporter(reporter interfaces.StatusReporter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusReporter = reporter
}

// Send sends or batches a notification based on configuration
func (m *Manager) Send(notification Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check rate limit
	if m.rateLimiter != nil && !m.rateLimiter.Allow() {
		// Silently drop notification due to rate limit
		return nil
	}

	// Startup notifications should be sent immediately, not batched
	if notification.Pattern == "startup" {
		// Send immediately
		if m.statusReporter != nil {
			m.statusReporter.ReportSending()
		}

		err := m.notifier.Send(notification)

		if m.statusReporter != nil {
			if err != nil {
				m.statusReporter.ReportFailure()
			} else {
				m.statusReporter.ReportSuccess()
			}
		}

		return err
	}

	// If batching is enabled, add to batch
	if m.batcher != nil {
		m.batcher.Add(notification)
		return nil
	}

	// Otherwise send immediately
	if m.statusReporter != nil {
		m.statusReporter.ReportSending()
	}

	err := m.notifier.Send(notification)

	if m.statusReporter != nil {
		if err != nil {
			m.statusReporter.ReportFailure()
		} else {
			m.statusReporter.ReportSuccess()
		}
	}

	return err
}

// sendBatch sends a batch of notifications as a single notification
func (m *Manager) sendBatch(notifications []Notification) {
	if len(notifications) == 0 {
		return
	}

	// Create a combined notification
	combined := Notification{
		Title:   "Claude Code: Multiple Matches",
		Message: formatBatchMessage(notifications),
		Time:    time.Now(),
		Pattern: "batch",
	}

	// Send the combined notification
	// Report status if available
	if m.statusReporter != nil {
		m.statusReporter.ReportSending()
	}

	err := m.notifier.Send(combined)

	if m.statusReporter != nil {
		if err != nil {
			m.statusReporter.ReportFailure()
		} else {
			m.statusReporter.ReportSuccess()
		}
	}

	// Errors are logged at the notifier level - notifications are best effort
	_ = err
}

// Close gracefully shuts down the manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Flush any pending batches
	if m.batcher != nil {
		m.batcher.Flush()
	}

	return nil
}

// formatBatchMessage formats multiple notifications into a single message
func formatBatchMessage(notifications []Notification) string {
	// Simple format for now - can be enhanced later
	msg := ""
	for i, n := range notifications {
		if i > 0 {
			msg += "\n---\n"
		}
		msg += n.Pattern + ": " + n.Message
	}
	return msg
}
