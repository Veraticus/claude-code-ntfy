package monitor

import (
	"testing"
	"time"
)

func TestTerminalState(t *testing.T) {
	t.Run("title management", func(t *testing.T) {
		ts := NewTerminalState()
		
		// Initial state
		if ts.GetTitle() != "" {
			t.Error("expected empty title initially")
		}
		
		// Set and get title
		ts.SetTitle("Test Title")
		if ts.GetTitle() != "Test Title" {
			t.Errorf("expected 'Test Title', got %q", ts.GetTitle())
		}
		
		// Update title
		ts.SetTitle("New Title")
		if ts.GetTitle() != "New Title" {
			t.Errorf("expected 'New Title', got %q", ts.GetTitle())
		}
	})
	
	t.Run("focus management", func(t *testing.T) {
		ts := NewTerminalState()
		
		// Initial state (assumed focused)
		if !ts.IsFocused() {
			t.Error("expected focused initially")
		}
		
		// Lose focus
		beforeChange := time.Now()
		ts.SetFocused(false)
		afterChange := time.Now()
		
		if ts.IsFocused() {
			t.Error("expected not focused after SetFocused(false)")
		}
		
		// Check focus change time
		changeTime := ts.GetLastFocusChange()
		if changeTime.Before(beforeChange) || changeTime.After(afterChange) {
			t.Error("focus change time not within expected range")
		}
		
		// Gain focus
		ts.SetFocused(true)
		if !ts.IsFocused() {
			t.Error("expected focused after SetFocused(true)")
		}
		
		// Setting same state shouldn't update time
		oldTime := ts.GetLastFocusChange()
		time.Sleep(10 * time.Millisecond)
		ts.SetFocused(true)
		newTime := ts.GetLastFocusChange()
		if !oldTime.Equal(newTime) {
			t.Error("focus change time shouldn't update when state doesn't change")
		}
	})
	
	t.Run("focus reporting enabled", func(t *testing.T) {
		ts := NewTerminalState()
		
		// Initial state
		if ts.IsFocusReportingEnabled() {
			t.Error("expected focus reporting disabled initially")
		}
		
		// Enable
		ts.SetFocusReportingEnabled(true)
		if !ts.IsFocusReportingEnabled() {
			t.Error("expected focus reporting enabled")
		}
		
		// Disable
		ts.SetFocusReportingEnabled(false)
		if ts.IsFocusReportingEnabled() {
			t.Error("expected focus reporting disabled")
		}
	})
	
	t.Run("concurrent access", func(t *testing.T) {
		ts := NewTerminalState()
		done := make(chan bool)
		
		// Concurrent writes
		go func() {
			for i := 0; i < 100; i++ {
				ts.SetTitle("Title A")
				ts.SetFocused(true)
				ts.SetFocusReportingEnabled(true)
			}
			done <- true
		}()
		
		go func() {
			for i := 0; i < 100; i++ {
				ts.SetTitle("Title B")
				ts.SetFocused(false)
				ts.SetFocusReportingEnabled(false)
			}
			done <- true
		}()
		
		// Concurrent reads
		go func() {
			for i := 0; i < 100; i++ {
				_ = ts.GetTitle()
				_ = ts.IsFocused()
				_ = ts.IsFocusReportingEnabled()
				_ = ts.GetLastFocusChange()
			}
			done <- true
		}()
		
		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}
	})
}