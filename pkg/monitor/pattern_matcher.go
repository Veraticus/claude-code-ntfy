package monitor

import (
	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
)

// PatternMatcher implements pattern matching for output
type PatternMatcher struct {
	patterns []interfaces.Pattern
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(patterns []interfaces.Pattern) *PatternMatcher {
	// Filter only enabled patterns with compiled regex
	enabledPatterns := make([]interfaces.Pattern, 0)
	for _, p := range patterns {
		if p.Enabled && p.CompiledRegex() != nil {
			enabledPatterns = append(enabledPatterns, p)
		}
	}

	return &PatternMatcher{
		patterns: enabledPatterns,
	}
}

// Match finds all pattern matches in the given text
func (pm *PatternMatcher) Match(text string) []interfaces.MatchResult {
	var results []interfaces.MatchResult

	for _, pattern := range pm.patterns {
		regex := pattern.CompiledRegex()
		if regex == nil {
			continue
		}

		// Find all matches
		matches := regex.FindAllStringIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				results = append(results, interfaces.MatchResult{
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
func (pm *PatternMatcher) GetPatterns() []interfaces.Pattern {
	return pm.patterns
}
