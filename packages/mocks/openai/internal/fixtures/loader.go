// Package fixtures provides response fixture management.
// This file implements loading fixtures from YAML files on disk.
package fixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sentra-lab/mocks/openai/internal/metrics"
)

// Loader loads fixtures from the file system.
type Loader struct {
	// store is the fixture store
	store *Store

	// baseDir is the base directory for fixtures
	baseDir string
}

// NewLoader creates a new fixture loader.
func NewLoader(store *Store, baseDir string) *Loader {
	return &Loader{
		store:   store,
		baseDir: baseDir,
	}
}

// LoadAll loads all fixtures from the base directory.
func (l *Loader) LoadAll() error {
	startTime := time.Now()
	totalLoaded := 0

	// Walk the directory tree
	err := filepath.Walk(l.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		// Load the fixture file
		if err := l.LoadFile(path); err != nil {
			metrics.LogFixtureError("fixture", path, err)
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		totalLoaded++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk fixtures directory: %w", err)
	}

	duration := time.Since(startTime)
	metrics.LogFixtureLoad("all", totalLoaded, duration)

	return nil
}

// LoadFile loads a single fixture file.
func (l *Loader) LoadFile(path string) error {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var fixtureFile FixtureFile
	if err := yaml.Unmarshal(data, &fixtureFile); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate fixtures
	if err := l.validateFixtureFile(&fixtureFile); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get relative path from base directory
	relPath, err := filepath.Rel(l.baseDir, path)
	if err != nil {
		relPath = path
	}

	// Add to store
	if err := l.store.Add(relPath, fixtureFile); err != nil {
		return fmt.Errorf("failed to add to store: %w", err)
	}

	return nil
}

// LoadDirectory loads all fixtures from a specific directory.
func (l *Loader) LoadDirectory(dir string) error {
	fullPath := filepath.Join(l.baseDir, dir)

	// Check if directory exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", fullPath)
	}

	startTime := time.Now()
	totalLoaded := 0

	// Read directory
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Load each YAML file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}

		filePath := filepath.Join(fullPath, name)
		if err := l.LoadFile(filePath); err != nil {
			metrics.LogFixtureError(dir, filePath, err)
			return err
		}

		totalLoaded++
	}

	duration := time.Since(startTime)
	metrics.LogFixtureLoad(dir, totalLoaded, duration)

	return nil
}

// LoadCategory loads fixtures for a specific category.
func (l *Loader) LoadCategory(category string) error {
	// Map categories to directories
	categoryDirs := map[string]string{
		"chat":       "responses/chat",
		"completion": "responses/completion",
		"embedding":  "responses/embedding",
		"image":      "responses/image",
		"error":      "errors",
		"pattern":    "patterns",
	}

	dir, ok := categoryDirs[category]
	if !ok {
		return fmt.Errorf("unknown category: %s", category)
	}

	return l.LoadDirectory(dir)
}

// Reload reloads all fixtures from disk.
func (l *Loader) Reload() error {
	// Clear existing fixtures
	l.store.Clear()

	// Load all fixtures
	return l.LoadAll()
}

// validateFixtureFile validates a fixture file structure.
func (l *Loader) validateFixtureFile(file *FixtureFile) error {
	if len(file.Responses) == 0 {
		return fmt.Errorf("fixture file has no responses")
	}

	for i, fixture := range file.Responses {
		if fixture.ID == "" {
			return fmt.Errorf("fixture %d: missing ID", i)
		}

		if fixture.Content == "" && fixture.FunctionCall == nil {
			return fmt.Errorf("fixture %s: missing content or function_call", fixture.ID)
		}

		// Validate weight
		if fixture.Weight < 0 {
			return fmt.Errorf("fixture %s: weight cannot be negative", fixture.ID)
		}

		// Validate finish_reason
		if fixture.FinishReason != "" {
			validReasons := map[string]bool{
				"stop":           true,
				"length":         true,
				"content_filter": true,
				"function_call":  true,
				"tool_calls":     true,
			}
			if !validReasons[fixture.FinishReason] {
				return fmt.Errorf("fixture %s: invalid finish_reason: %s", fixture.ID, fixture.FinishReason)
			}
		}
	}

	return nil
}

// Watch watches the fixtures directory for changes and reloads automatically.
// This is useful for development but should be disabled in production.
func (l *Loader) Watch(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastModTime := make(map[string]time.Time)

	// Initial scan
	l.scanForChanges(lastModTime)

	for {
		select {
		case <-ticker.C:
			if changed := l.scanForChanges(lastModTime); changed {
				if err := l.Reload(); err != nil {
					metrics.LogFixtureError("watch", l.baseDir, err)
				}
			}
		case <-stopCh:
			return
		}
	}
}

// scanForChanges scans for file changes.
func (l *Loader) scanForChanges(lastModTime map[string]time.Time) bool {
	changed := false

	filepath.Walk(l.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		modTime := info.ModTime()
		if lastMod, ok := lastModTime[path]; !ok || modTime.After(lastMod) {
			changed = true
			lastModTime[path] = modTime
		}

		return nil
	})

	return changed
}

// GetLoadedPaths returns all loaded fixture paths.
func (l *Loader) GetLoadedPaths() []string {
	return l.store.List()
}

// GetLoadedCategories returns all loaded categories.
func (l *Loader) GetLoadedCategories() []string {
	return l.store.ListCategories()
}

// GetStats returns loader statistics.
func (l *Loader) GetStats() LoaderStats {
	storeStats := l.store.GetStats()

	return LoaderStats{
		BaseDirectory:    l.baseDir,
		TotalFixtures:    storeStats.TotalFixtures,
		LoadedPaths:      storeStats.LoadedPaths,
		LoadedCategories: storeStats.LoadedCategories,
	}
}

// LoaderStats contains statistics about the loader.
type LoaderStats struct {
	BaseDirectory    string
	TotalFixtures    int
	LoadedPaths      int
	LoadedCategories int
}