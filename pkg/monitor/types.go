package monitor

// MatchResult represents a pattern match result.
type MatchResult struct {
	PatternName string
	Text        string
	Position    int
}

// PatternMatcher matches patterns in text.
type PatternMatcher interface {
	Match(text string) []MatchResult
}
