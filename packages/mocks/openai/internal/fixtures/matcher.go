// Package fixtures provides response fixture management.
// This file implements pattern matching for selecting appropriate fixtures.
package fixtures

import (
	"regexp"
	"strings"
	"sync"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Matcher matches prompts to fixtures using patterns.
type Matcher struct {
	// store is the fixture store
	store *Store

	// patterns maps pattern names to compiled regexes
	patterns map[string]*regexp.Regexp

	// patternFixtures maps pattern names to fixture paths
	patternFixtures map[string]string

	// mu protects concurrent access
	mu sync.RWMutex

	// defaultPath is the fallback fixture path
	defaultPath string
}

// PatternConfig configures a pattern for matching.
type PatternConfig struct {
	// Name is the pattern name
	Name string

	// Regex is the regular expression pattern
	Regex string

	// Fixture is the fixture path to use when matched
	Fixture string

	// CaseInsensitive enables case-insensitive matching
	CaseInsensitive bool

	// Priority affects matching order (higher = checked first)
	Priority int
}

// NewMatcher creates a new pattern matcher.
func NewMatcher(store *Store, defaultPath string) *Matcher {
	return &Matcher{
		store:           store,
		patterns:        make(map[string]*regexp.Regexp),
		patternFixtures: make(map[string]string),
		defaultPath:     defaultPath,
	}
}

// AddPattern adds a pattern for matching.
func (m *Matcher) AddPattern(config PatternConfig) error {
	// Compile regex
	pattern := config.Regex
	if config.CaseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.patterns[config.Name] = re
	m.patternFixtures[config.Name] = config.Fixture

	return nil
}

// AddPatterns adds multiple patterns.
func (m *Matcher) AddPatterns(configs []PatternConfig) error {
	for _, config := range configs {
		if err := m.AddPattern(config); err != nil {
			return err
		}
	}
	return nil
}

// Match matches a prompt to a fixture using patterns.
func (m *Matcher) Match(messages []models.Message) (*Fixture, error) {
	// Extract text from messages
	text := m.extractText(messages)

	// Try to match patterns
	if fixturePath := m.matchPatterns(text); fixturePath != "" {
		return m.store.GetWeighted(fixturePath)
	}

	// Fall back to default path
	if m.defaultPath != "" {
		return m.store.GetWeighted(m.defaultPath)
	}

	// If no default, return generic response
	return m.store.GetWeighted("responses/chat/generic.yaml")
}

// MatchText matches plain text to a fixture.
func (m *Matcher) MatchText(text string) (*Fixture, error) {
	// Try to match patterns
	if fixturePath := m.matchPatterns(text); fixturePath != "" {
		return m.store.GetWeighted(fixturePath)
	}

	// Fall back to default
	if m.defaultPath != "" {
		return m.store.GetWeighted(m.defaultPath)
	}

	return m.store.GetWeighted("responses/chat/generic.yaml")
}

// MatchWithCategory matches and prefers a specific category.
func (m *Matcher) MatchWithCategory(messages []models.Message, preferredCategory string) (*Fixture, error) {
	text := m.extractText(messages)

	// Try to match patterns
	if fixturePath := m.matchPatterns(text); fixturePath != "" {
		return m.store.GetWeighted(fixturePath)
	}

	// Try preferred category
	if preferredCategory != "" {
		if fixture, err := m.store.GetByCategory(preferredCategory); err == nil {
			return fixture, nil
		}
	}

	// Fall back to default
	return m.Match(messages)
}

// matchPatterns matches text against all patterns.
func (m *Matcher) matchPatterns(text string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try each pattern (order matters if priorities are equal)
	for name, re := range m.patterns {
		if re.MatchString(text) {
			return m.patternFixtures[name]
		}
	}

	return ""
}

// extractText extracts text from messages for matching.
func (m *Matcher) extractText(messages []models.Message) string {
	var builder strings.Builder

	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "system" {
			builder.WriteString(msg.Content)
			builder.WriteString(" ")
		}
	}

	return strings.TrimSpace(builder.String())
}

// GetPatternName returns the pattern name that matches the text.
func (m *Matcher) GetPatternName(text string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, re := range m.patterns {
		if re.MatchString(text) {
			return name
		}
	}

	return "default"
}

// ListPatterns returns all registered pattern names.
func (m *Matcher) ListPatterns() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.patterns))
	for name := range m.patterns {
		names = append(names, name)
	}

	return names
}

// RemovePattern removes a pattern.
func (m *Matcher) RemovePattern(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.patterns, name)
	delete(m.patternFixtures, name)
}

// ClearPatterns removes all patterns.
func (m *Matcher) ClearPatterns() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.patterns = make(map[string]*regexp.Regexp)
	m.patternFixtures = make(map[string]string)
}

// SetDefaultPath sets the default fixture path.
func (m *Matcher) SetDefaultPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultPath = path
}

// GetDefaultPath returns the default fixture path.
func (m *Matcher) GetDefaultPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultPath
}

// TestPattern tests if a pattern matches text.
func (m *Matcher) TestPattern(patternName string, text string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	re, ok := m.patterns[patternName]
	if !ok {
		return false
	}

	return re.MatchString(text)
}

// GetMatchingPatterns returns all patterns that match the text.
func (m *Matcher) GetMatchingPatterns(text string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []string
	for name, re := range m.patterns {
		if re.MatchString(text) {
			matches = append(matches, name)
		}
	}

	return matches
}

// MatcherStats contains statistics about pattern matching.
type MatcherStats struct {
	TotalPatterns   int
	DefaultPath     string
	PatternFixtures map[string]string
}

// GetStats returns matcher statistics.
func (m *Matcher) GetStats() MatcherStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Copy pattern fixtures
	fixtures := make(map[string]string)
	for k, v := range m.patternFixtures {
		fixtures[k] = v
	}

	return MatcherStats{
		TotalPatterns:   len(m.patterns),
		DefaultPath:     m.defaultPath,
		PatternFixtures: fixtures,
	}
}