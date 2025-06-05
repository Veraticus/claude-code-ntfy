package monitor

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/config"
)

// SimplePatternMatcher implements pattern matching for output
type SimplePatternMatcher struct {
	patterns []config.Pattern
}

// NewSimplePatternMatcher creates a new pattern matcher
func NewSimplePatternMatcher(patterns []config.Pattern) *SimplePatternMatcher {
	// Filter only enabled patterns with compiled regex
	enabledPatterns := make([]config.Pattern, 0)
	for _, p := range patterns {
		if p.Enabled && p.CompiledRegex() != nil {
			enabledPatterns = append(enabledPatterns, p)
		}
	}

	return &SimplePatternMatcher{
		patterns: enabledPatterns,
	}
}

// Match finds all pattern matches in the given text
func (pm *SimplePatternMatcher) Match(text string) []MatchResult {
	var results []MatchResult

	for _, pattern := range pm.patterns {
		regex := pattern.CompiledRegex()
		if regex == nil {
			continue
		}

		// Find all matches
		matches := regex.FindAllStringIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				results = append(results, MatchResult{
					PatternName: pattern.Name,
					Text:        text[match[0]:match[1]],
					Position:    match[0],
				})
			}
		}
	}

	return results
}

// GetPatterns returns the active patterns
func (pm *SimplePatternMatcher) GetPatterns() []config.Pattern {
	return pm.patterns
}
