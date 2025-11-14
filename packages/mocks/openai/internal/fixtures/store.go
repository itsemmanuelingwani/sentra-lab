// Package fixtures provides response fixture management for the OpenAI mock.
// This file implements the in-memory store for loaded fixtures.
package fixtures

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Fixture represents a single response fixture.
type Fixture struct {
	// ID is the unique identifier for this fixture
	ID string `yaml:"id"`

	// Pattern is the regex pattern to match prompts (optional)
	Pattern string `yaml:"pattern"`

	// Content is the response content
	Content string `yaml:"content"`

	// Role is the message role (default: "assistant")
	Role string `yaml:"role"`

	// FunctionCall is an optional function call in the response
	FunctionCall *models.FunctionCall `yaml:"function_call,omitempty"`

	// FinishReason is the finish reason (default: "stop")
	FinishReason string `yaml:"finish_reason"`

	// Metadata contains additional fixture metadata
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`

	// Weight for weighted random selection (default: 1.0)
	Weight float64 `yaml:"weight"`
}

// FixtureFile represents a YAML fixture file structure.
type FixtureFile struct {
	// Description describes the fixture set
	Description string `yaml:"description"`

	// Category categorizes the fixtures (e.g., "code", "creative")
	Category string `yaml:"category"`

	// Responses is the list of fixtures
	Responses []Fixture `yaml:"responses"`
}

// Store manages loaded fixtures in memory.
type Store struct {
	// fixtures maps fixture paths to fixture lists
	fixtures map[string][]Fixture

	// categories maps category names to fixture paths
	categories map[string][]string

	// mu protects concurrent access
	mu sync.RWMutex

	// stats tracks fixture usage
	fixtureHits  map[string]int64
	totalQueries int64
	statsMu      sync.RWMutex
}

// NewStore creates a new fixture store.
func NewStore() *Store {
	return &Store{
		fixtures:    make(map[string][]Fixture),
		categories:  make(map[string][]string),
		fixtureHits: make(map[string]int64),
	}
}

// Add adds fixtures to the store.
func (s *Store) Add(path string, fixtureFile FixtureFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store fixtures
	s.fixtures[path] = fixtureFile.Responses

	// Index by category
	if fixtureFile.Category != "" {
		s.categories[fixtureFile.Category] = append(s.categories[fixtureFile.Category], path)
	}

	return nil
}

// Get retrieves a random fixture from a path.
func (s *Store) Get(path string) (*Fixture, error) {
	s.mu.RLock()
	fixtures, ok := s.fixtures[path]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s", path)
	}

	if len(fixtures) == 0 {
		return nil, fmt.Errorf("no fixtures available in: %s", path)
	}

	// Record query
	s.recordQuery(path)

	// Select random fixture
	idx := rand.Intn(len(fixtures))
	fixture := fixtures[idx]

	// Set defaults
	if fixture.Role == "" {
		fixture.Role = "assistant"
	}
	if fixture.FinishReason == "" {
		fixture.FinishReason = "stop"
	}
	if fixture.Weight == 0 {
		fixture.Weight = 1.0
	}

	return &fixture, nil
}

// GetWeighted retrieves a fixture using weighted random selection.
func (s *Store) GetWeighted(path string) (*Fixture, error) {
	s.mu.RLock()
	fixtures, ok := s.fixtures[path]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s", path)
	}

	if len(fixtures) == 0 {
		return nil, fmt.Errorf("no fixtures available in: %s", path)
	}

	// Record query
	s.recordQuery(path)

	// Calculate total weight
	totalWeight := 0.0
	for _, f := range fixtures {
		weight := f.Weight
		if weight == 0 {
			weight = 1.0
		}
		totalWeight += weight
	}

	// Weighted random selection
	r := rand.Float64() * totalWeight
	cumulative := 0.0

	for i, f := range fixtures {
		weight := f.Weight
		if weight == 0 {
			weight = 1.0
		}
		cumulative += weight

		if r <= cumulative {
			fixture := fixtures[i]

			// Set defaults
			if fixture.Role == "" {
				fixture.Role = "assistant"
			}
			if fixture.FinishReason == "" {
				fixture.FinishReason = "stop"
			}
			if fixture.Weight == 0 {
				fixture.Weight = 1.0
			}

			return &fixture, nil
		}
	}

	// Fallback (shouldn't reach here)
	return s.Get(path)
}

// GetByID retrieves a specific fixture by ID.
func (s *Store) GetByID(path string, id string) (*Fixture, error) {
	s.mu.RLock()
	fixtures, ok := s.fixtures[path]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s", path)
	}

	// Find fixture by ID
	for _, f := range fixtures {
		if f.ID == id {
			fixture := f

			// Set defaults
			if fixture.Role == "" {
				fixture.Role = "assistant"
			}
			if fixture.FinishReason == "" {
				fixture.FinishReason = "stop"
			}

			s.recordQuery(path)
			return &fixture, nil
		}
	}

	return nil, fmt.Errorf("fixture not found: %s in %s", id, path)
}

// GetByCategory retrieves fixtures from all paths in a category.
func (s *Store) GetByCategory(category string) (*Fixture, error) {
	s.mu.RLock()
	paths, ok := s.categories[category]
	s.mu.RUnlock()

	if !ok || len(paths) == 0 {
		return nil, fmt.Errorf("category not found: %s", category)
	}

	// Pick random path
	path := paths[rand.Intn(len(paths))]

	// Get fixture from that path
	return s.GetWeighted(path)
}

// GetAll retrieves all fixtures from a path.
func (s *Store) GetAll(path string) ([]Fixture, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fixtures, ok := s.fixtures[path]
	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s", path)
	}

	// Return a copy
	result := make([]Fixture, len(fixtures))
	copy(result, fixtures)

	return result, nil
}

// List returns all loaded fixture paths.
func (s *Store) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paths := make([]string, 0, len(s.fixtures))
	for path := range s.fixtures {
		paths = append(paths, path)
	}

	return paths
}

// ListCategories returns all available categories.
func (s *Store) ListCategories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]string, 0, len(s.categories))
	for cat := range s.categories {
		categories = append(categories, cat)
	}

	return categories
}

// Count returns the number of fixtures in a path.
func (s *Store) Count(path string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fixtures, ok := s.fixtures[path]
	if !ok {
		return 0
	}

	return len(fixtures)
}

// TotalCount returns the total number of fixtures loaded.
func (s *Store) TotalCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, fixtures := range s.fixtures {
		total += len(fixtures)
	}

	return total
}

// recordQuery records a fixture query for statistics.
func (s *Store) recordQuery(path string) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.fixtureHits[path]++
	s.totalQueries++
}

// GetStats returns fixture usage statistics.
func (s *Store) GetStats() StoreStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Copy hits
	hits := make(map[string]int64)
	for k, v := range s.fixtureHits {
		hits[k] = v
	}

	return StoreStats{
		TotalFixtures:  s.TotalCount(),
		TotalQueries:   s.totalQueries,
		FixtureHits:    hits,
		LoadedPaths:    len(s.fixtures),
		LoadedCategories: len(s.categories),
	}
}

// ResetStats resets usage statistics.
func (s *Store) ResetStats() {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.fixtureHits = make(map[string]int64)
	s.totalQueries = 0
}

// Clear removes all fixtures from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fixtures = make(map[string][]Fixture)
	s.categories = make(map[string][]string)

	s.ResetStats()
}

// StoreStats contains statistics about fixture usage.
type StoreStats struct {
	// TotalFixtures is the total number of loaded fixtures
	TotalFixtures int

	// TotalQueries is the total number of queries
	TotalQueries int64

	// FixtureHits maps paths to hit counts
	FixtureHits map[string]int64

	// LoadedPaths is the number of loaded fixture paths
	LoadedPaths int

	// LoadedCategories is the number of loaded categories
	LoadedCategories int
}

// GetMostUsed returns the most frequently used fixture paths.
func (s *StoreStats) GetMostUsed(n int) []string {
	// Convert to slice for sorting
	type pathHit struct {
		path string
		hits int64
	}

	paths := make([]pathHit, 0, len(s.FixtureHits))
	for path, hits := range s.FixtureHits {
		paths = append(paths, pathHit{path, hits})
	}

	// Simple bubble sort for top N
	for i := 0; i < len(paths) && i < n; i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j].hits > paths[i].hits {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}

	if n > len(paths) {
		n = len(paths)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = paths[i].path
	}

	return result
}