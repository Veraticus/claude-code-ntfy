package notification

import (
	"sync"
	"time"
)

// Batcher groups notifications within a time window
type Batcher struct {
	window   time.Duration
	callback func([]Notification)

	mu      sync.Mutex
	pending []Notification
	timer   *time.Timer
}

// NewBatcher creates a new notification batcher
func NewBatcher(window time.Duration, callback func([]Notification)) *Batcher {
	return &Batcher{
		window:   window,
		callback: callback,
	}
}

// Add adds a notification to the batch
func (b *Batcher) Add(n Notification) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pending = append(b.pending, n)

	// Start timer if not already running
	if b.timer == nil {
		b.timer = time.AfterFunc(b.window, b.flush)
	}
}

// flush sends all pending notifications
func (b *Batcher) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pending) == 0 {
		b.timer = nil
		return
	}

	// Copy pending notifications
	toSend := make([]Notification, len(b.pending))
	copy(toSend, b.pending)

	// Clear pending
	b.pending = nil
	b.timer = nil

	// Call callback outside of lock
	b.mu.Unlock()
	b.callback(toSend)
	b.mu.Lock()
}

// Flush immediately sends any pending notifications
func (b *Batcher) Flush() {
	b.mu.Lock()
	if b.timer != nil {
		b.timer.Stop()
	}
	b.mu.Unlock()

	b.flush()
}
