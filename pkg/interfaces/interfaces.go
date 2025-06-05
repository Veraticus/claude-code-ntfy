package interfaces

import (
	"regexp"
	"time"
)

// IdleDetector detects user activity/inactivity
type IdleDetector interface {
	IsUserIdle(threshold time.Duration) (bool, error)
	LastActivity() time.Time
}

// Notifier sends notifications
type Notifier interface {
	Send(notification Notification) error
}

// ProcessWrapper wraps and monitors a process
type ProcessWrapper interface {
	Start(command string, args []string) error
	Wait() error
	ExitCode() int
}

// OutputHandler processes output lines
type OutputHandler interface {
	HandleLine(line string)
}

// PatternMatcher matches patterns in text
type PatternMatcher interface {
	Match(text string) []MatchResult
}

// RateLimiter limits notification frequency
type RateLimiter interface {
	Allow() bool
	Reset()
}

// Notification represents a notification to be sent
type Notification struct {
	Title   string
	Message string
	Time    time.Time
	Pattern string
}

// MatchResult represents a pattern match result
type MatchResult struct {
	PatternName string
	Text        string
	Position    int
}

// Pattern represents a configurable pattern
type Pattern struct {
	Name        string         `yaml:"name"`
	Regex       string         `yaml:"regex"`
	Description string         `yaml:"description"`
	Enabled     bool           `yaml:"enabled"`
	compiled    *regexp.Regexp `yaml:"-"`
}

// CompiledRegex returns the compiled regular expression
func (p *Pattern) CompiledRegex() *regexp.Regexp {
	return p.compiled
}

// SetCompiledRegex sets the compiled regular expression
func (p *Pattern) SetCompiledRegex(re *regexp.Regexp) {
	p.compiled = re
}
