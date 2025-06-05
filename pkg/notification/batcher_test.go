package notification

import (
	"sync"
	"testing"
	"time"
)

func TestBatcher_Add(t *testing.T) {
	tests := []struct {
		name          string
		window        time.Duration
		notifications []struct {
			notification Notification
			delay        time.Duration
		}
		wantBatches int
		wantSizes   []int
	}{
		{
			name:   "single batch within window",
			window: 100 * time.Millisecond,
			notifications: []struct {
				notification Notification
				delay        time.Duration
			}{
				{notification: Notification{Title: "1"}, delay: 0},
				{notification: Notification{Title: "2"}, delay: 10 * time.Millisecond},
				{notification: Notification{Title: "3"}, delay: 10 * time.Millisecond},
			},
			wantBatches: 1,
			wantSizes:   []int{3},
		},
		{
			name:   "multiple batches across windows",
			window: 50 * time.Millisecond,
			notifications: []struct {
				notification Notification
				delay        time.Duration
			}{
				{notification: Notification{Title: "1"}, delay: 0},
				{notification: Notification{Title: "2"}, delay: 100 * time.Millisecond}, // Wait for first batch to complete
				{notification: Notification{Title: "3"}, delay: 10 * time.Millisecond},  // Add to second batch
			},
			wantBatches: 2,
			wantSizes:   []int{1, 2},
		},
		{
			name:   "single notification triggers batch",
			window: 30 * time.Millisecond,
			notifications: []struct {
				notification Notification
				delay        time.Duration
			}{
				{notification: Notification{Title: "only"}, delay: 0},
			},
			wantBatches: 1,
			wantSizes:   []int{1},
		},
		{
			name:   "no notifications",
			window: 50 * time.Millisecond,
			notifications: []struct {
				notification Notification
				delay        time.Duration
			}{},
			wantBatches: 0,
			wantSizes:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			batches := [][]Notification{}

			callback := func(notifications []Notification) {
				mu.Lock()
				batch := make([]Notification, len(notifications))
				copy(batch, notifications)
				batches = append(batches, batch)
				mu.Unlock()
			}

			batcher := NewBatcher(tt.window, callback)
			defer batcher.Flush()

			// Add notifications
			for _, n := range tt.notifications {
				if n.delay > 0 {
					time.Sleep(n.delay)
				}
				batcher.Add(n.notification)
			}

			// Wait for final batch
			time.Sleep(tt.window + 50*time.Millisecond)

			// Check results
			mu.Lock()
			defer mu.Unlock()

			if len(batches) != tt.wantBatches {
				t.Errorf("got %d batches, want %d", len(batches), tt.wantBatches)
			}

			for i, batch := range batches {
				if i < len(tt.wantSizes) {
					if len(batch) != tt.wantSizes[i] {
						t.Errorf("batch[%d] size = %d, want %d", i, len(batch), tt.wantSizes[i])
					}
				}
			}
		})
	}
}

func TestBatcher_Flush(t *testing.T) {
	var mu sync.Mutex
	flushCalled := false
	var flushedNotifications []Notification

	callback := func(notifications []Notification) {
		mu.Lock()
		flushCalled = true
		flushedNotifications = make([]Notification, len(notifications))
		copy(flushedNotifications, notifications)
		mu.Unlock()
	}

	batcher := NewBatcher(time.Hour, callback) // Long window

	// Add notifications
	n1 := Notification{Title: "1"}
	n2 := Notification{Title: "2"}
	batcher.Add(n1)
	batcher.Add(n2)

	// Flush should send immediately
	batcher.Flush()

	// Give callback time to execute
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !flushCalled {
		t.Error("Flush() did not trigger flush callback")
	}

	if len(flushedNotifications) != 2 {
		t.Errorf("Flushed %d notifications, want 2", len(flushedNotifications))
	}

	// Verify notifications are correct
	if flushedNotifications[0].Title != "1" || flushedNotifications[1].Title != "2" {
		t.Error("Flushed notifications don't match added notifications")
	}
}

func TestBatcher_FlushMultipleTimes(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	callback := func(notifications []Notification) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	batcher := NewBatcher(50*time.Millisecond, callback)
	batcher.Add(Notification{Title: "test"})

	// Flush multiple times
	batcher.Flush()
	batcher.Flush() // Should be no-op if no new notifications
	batcher.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// First flush should send, others should be no-op
	if callCount != 1 {
		t.Errorf("Callback called %d times, want 1", callCount)
	}
}

func TestBatcher_AddAfterFlush(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	callback := func(notifications []Notification) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	batcher := NewBatcher(100*time.Millisecond, callback)
	batcher.Add(Notification{Title: "first"})
	batcher.Flush()

	// Adding after flush should work
	batcher.Add(Notification{Title: "second"})

	// Wait for second batch
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have two batches
	if callCount != 2 {
		t.Errorf("Expected 2 batches, got %d", callCount)
	}
}

func TestBatcher_ConcurrentAdd(t *testing.T) {
	var mu sync.Mutex
	totalReceived := 0

	callback := func(notifications []Notification) {
		mu.Lock()
		totalReceived += len(notifications)
		mu.Unlock()
	}

	batcher := NewBatcher(100*time.Millisecond, callback)
	defer batcher.Flush()

	// Add notifications concurrently
	numGoroutines := 10
	notificationsPerGoroutine := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < notificationsPerGoroutine; j++ {
				batcher.Add(Notification{
					Title: string(rune('A' + id)),
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(150 * time.Millisecond) // Wait for batch to flush

	mu.Lock()
	defer mu.Unlock()

	expectedTotal := numGoroutines * notificationsPerGoroutine
	if totalReceived != expectedTotal {
		t.Errorf("Received %d notifications, want %d", totalReceived, expectedTotal)
	}
}

func TestNewBatcher(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
	}{
		{
			name:   "normal window",
			window: 100 * time.Millisecond,
		},
		{
			name:   "very short window",
			window: 1 * time.Millisecond,
		},
		{
			name:   "long window",
			window: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callback := func([]Notification) {}
			batcher := NewBatcher(tt.window, callback)
			if batcher == nil {
				t.Error("NewBatcher() returned nil")
			}
			batcher.Flush()
		})
	}
}
