package monitor

import (
	"regexp"
	"testing"

	"github.com/Veraticus/claude-code-ntfy/pkg/types"
)

func TestPatternMatcher_Match(t *testing.T) {
	// Create test patterns
	patterns := []types.Pattern{
		{
			Name:    "numbers",
			Regex:   `\d+`,
			Enabled: true,
		},
		{
			Name:    "email",
			Regex:   `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			Enabled: true,
		},
		{
			Name:    "disabled",
			Regex:   `disabled`,
			Enabled: false, // This pattern should not be used
		},
		{
			Name:    "error",
			Regex:   `(?i)(error|failed)`,
			Enabled: true,
		},
	}

	// Compile regex patterns
	for i := range patterns {
		if patterns[i].Enabled && patterns[i].Regex != "" {
			re, err := regexp.Compile(patterns[i].Regex)
			if err != nil {
				t.Fatalf("failed to compile pattern %s: %v", patterns[i].Name, err)
			}
			patterns[i].SetCompiledRegex(re)
		}
	}

	pm := NewPatternMatcher(patterns)

	tests := []struct {
		name     string
		text     string
		expected []struct {
			pattern  string
			text     string
			position int
		}
	}{
		{
			name: "single number match",
			text: "Found 123 items",
			expected: []struct {
				pattern  string
				text     string
				position int
			}{
				{pattern: "numbers", text: "123", position: 6},
			},
		},
		{
			name: "multiple number matches",
			text: "Numbers: 123, 456, and 789",
			expected: []struct {
				pattern  string
				text     string
				position int
			}{
				{pattern: "numbers", text: "123", position: 9},
				{pattern: "numbers", text: "456", position: 14},
				{pattern: "numbers", text: "789", position: 23},
			},
		},
		{
			name: "email match",
			text: "Contact us at support@example.com for help",
			expected: []struct {
				pattern  string
				text     string
				position int
			}{
				{pattern: "email", text: "support@example.com", position: 14},
			},
		},
		{
			name: "error pattern match",
			text: "ERROR: Operation failed",
			expected: []struct {
				pattern  string
				text     string
				position int
			}{
				{pattern: "error", text: "ERROR", position: 0},
				{pattern: "error", text: "failed", position: 17},
			},
		},
		{
			name: "no matches",
			text: "This is just plain text",
			expected: []struct {
				pattern, text string
				position      int
			}{},
		},
		{
			name: "disabled pattern not matched",
			text: "This text contains disabled word",
			expected: []struct {
				pattern, text string
				position      int
			}{},
		},
		{
			name: "multiple patterns match",
			text: "ERROR 404: Page not found",
			expected: []struct {
				pattern  string
				text     string
				position int
			}{
				{pattern: "numbers", text: "404", position: 6},
				{pattern: "error", text: "ERROR", position: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := pm.Match(tt.text)

			if len(results) != len(tt.expected) {
				t.Errorf("expected %d matches but got %d", len(tt.expected), len(results))
				return
			}

			// Check each result
			for i, expected := range tt.expected {
				if i >= len(results) {
					t.Errorf("missing result %d", i)
					continue
				}

				result := results[i]
				if result.PatternName != expected.pattern {
					t.Errorf("result %d: expected pattern %s but got %s", i, expected.pattern, result.PatternName)
				}
				if result.Text != expected.text {
					t.Errorf("result %d: expected text %q but got %q", i, expected.text, result.Text)
				}
				if result.Position != expected.position {
					t.Errorf("result %d: expected position %d but got %d", i, expected.position, result.Position)
				}
			}
		})
	}
}

func TestPatternMatcher_NewPatternMatcher(t *testing.T) {
	patterns := []types.Pattern{
		{
			Name:    "enabled1",
			Regex:   `test`,
			Enabled: true,
		},
		{
			Name:    "disabled",
			Regex:   `test`,
			Enabled: false,
		},
		{
			Name:    "enabled2",
			Regex:   `test`,
			Enabled: true,
		},
		{
			Name:    "no_regex",
			Enabled: true,
		},
	}

	// Compile only enabled patterns
	for i := range patterns {
		if patterns[i].Enabled && patterns[i].Regex != "" {
			re, _ := regexp.Compile(patterns[i].Regex)
			patterns[i].SetCompiledRegex(re)
		}
	}

	pm := NewPatternMatcher(patterns)

	// Should only have 2 patterns (enabled1 and enabled2)
	activePatterns := pm.GetPatterns()
	if len(activePatterns) != 2 {
		t.Errorf("expected 2 active patterns but got %d", len(activePatterns))
	}

	// Check that only enabled patterns with compiled regex are included
	for _, p := range activePatterns {
		if !p.Enabled {
			t.Errorf("found disabled pattern %s", p.Name)
		}
		if p.CompiledRegex() == nil {
			t.Errorf("found pattern %s without compiled regex", p.Name)
		}
	}
}

func TestPatternMatcher_EmptyPatterns(t *testing.T) {
	pm := NewPatternMatcher([]types.Pattern{})

	results := pm.Match("test text with numbers 123")
	if len(results) != 0 {
		t.Errorf("expected no results but got %d", len(results))
	}
}

func TestPatternMatcher_ComplexRegex(t *testing.T) {
	patterns := []types.Pattern{
		{
			Name:    "ip_address",
			Regex:   `\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`,
			Enabled: true,
		},
		{
			Name:    "url",
			Regex:   `https?://[^\s]+`,
			Enabled: true,
		},
		{
			Name:    "timestamp",
			Regex:   `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,
			Enabled: true,
		},
	}

	// Compile patterns
	for i := range patterns {
		re, err := regexp.Compile(patterns[i].Regex)
		if err != nil {
			t.Fatalf("failed to compile pattern %s: %v", patterns[i].Name, err)
		}
		patterns[i].SetCompiledRegex(re)
	}

	pm := NewPatternMatcher(patterns)

	text := "Server 192.168.1.100 responded at 2024-01-15 14:30:45. Visit https://example.com for more info."
	results := pm.Match(text)

	expectedMatches := map[string]string{
		"ip_address": "192.168.1.100",
		"timestamp":  "2024-01-15 14:30:45",
		"url":        "https://example.com",
	}

	if len(results) != len(expectedMatches) {
		t.Errorf("expected %d matches but got %d", len(expectedMatches), len(results))
	}

	// Verify each expected match was found
	for _, result := range results {
		expected, ok := expectedMatches[result.PatternName]
		if !ok {
			t.Errorf("unexpected pattern match: %s", result.PatternName)
			continue
		}
		if result.Text != expected {
			t.Errorf("pattern %s: expected %q but got %q", result.PatternName, expected, result.Text)
		}
	}
}
