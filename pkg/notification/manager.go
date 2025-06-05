package notification

import (
	"sync"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// Manager orchestrates notification sending with batching and rate limiting
type Manager struct {
	config      *config.Config
	notifier    Notifier
	rateLimiter interfaces.RateLimiter
	batcher     *Batcher

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

// Send sends or batches a notification based on configuration
func (m *Manager) Send(notification Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check rate limit
	if m.rateLimiter != nil && !m.rateLimiter.Allow() {
		// Silently drop notification due to rate limit
		return nil
	}

	// If batching is enabled, add to batch
	if m.batcher != nil {
		m.batcher.Add(notification)
		return nil
	}

	// Otherwise send immediately
	return m.notifier.Send(notification)
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
	if err := m.notifier.Send(combined); err != nil {
		// Log error but don't propagate - notifications are best effort
		// Error logging would be done at the notifier level
	}
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
