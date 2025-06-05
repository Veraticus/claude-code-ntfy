package testutil

import (
	"sync"
)

// MockPatternMatcher is a mock implementation of monitor.PatternMatcher for testing
type MockPatternMatcher struct {
	mu             sync.Mutex
	matchResult    string
	matchFound     bool
	matchCallCount int
}

// NewMockPatternMatcher creates a new mock pattern matcher
func NewMockPatternMatcher(matchResult string, matchFound bool) *MockPatternMatcher {
	return &MockPatternMatcher{
		matchResult: matchResult,
		matchFound:  matchFound,
	}
}

// Match implements the PatternMatcher interface
func (m *MockPatternMatcher) Match(text string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matchCallCount++
	return m.matchResult, m.matchFound
}

// SetMatchResult sets what Match will return
func (m *MockPatternMatcher) SetMatchResult(result string, found bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matchResult = result
	m.matchFound = found
}

// GetMatchCallCount returns how many times Match was called
func (m *MockPatternMatcher) GetMatchCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.matchCallCount
}
