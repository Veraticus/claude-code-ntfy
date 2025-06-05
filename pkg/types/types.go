// Package types contains shared data structures used across the application.
package types

import (
	"regexp"
	"time"
)

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

