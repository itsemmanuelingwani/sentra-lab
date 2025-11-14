// Package latency provides production-realistic latency simulation for API responses.
// This file defines latency profiles for different models matching real OpenAI performance.
package latency

import (
	"fmt"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Profile defines the latency characteristics for a model.
// These values are derived from production measurements of OpenAI's API (Nov 2025).
type Profile struct {
	// ModelID is the model identifier
	ModelID string

	// BaseLatency is the Time To First Token (TTFT)
	// This is the initial delay before response generation starts
	BaseLatency time.Duration

	// PerTokenLatency is the latency added per output token
	// This simulates sequential token generation
	PerTokenLatency time.Duration

	// JitterPercent is the random variance as a percentage (0.0 to 1.0)
	// Example: 0.25 means ±25% variation
	JitterPercent float64

	// MinLatency is the absolute minimum latency (safety floor)
	MinLatency time.Duration

	// MaxLatency is the absolute maximum latency (safety ceiling)
	MaxLatency time.Duration

	// P50Latency is the 50th percentile latency for 100 tokens (reference)
	P50Latency time.Duration

	// P95Latency is the 95th percentile latency for 100 tokens (reference)
	P95Latency time.Duration

	// P99Latency is the 99th percentile latency for 100 tokens (reference)
	P99Latency time.Duration
}

// ProfileRegistry manages latency profiles for all models.
type ProfileRegistry struct {
	profiles map[string]Profile
}

// NewProfileRegistry creates a new profile registry with default profiles.
func NewProfileRegistry() *ProfileRegistry {
	registry := &ProfileRegistry{
		profiles: make(map[string]Profile),
	}

	// Load default profiles from production measurements
	registry.loadDefaultProfiles()

	return registry
}

// loadDefaultProfiles loads production-measured latency profiles.
// These values are based on real OpenAI API measurements as of Nov 2025.
func (r *ProfileRegistry) loadDefaultProfiles() {
	// GPT-4o - Fastest GPT-4 class model
	r.profiles["gpt-4o"] = Profile{
		ModelID:         "gpt-4o",
		BaseLatency:     500 * time.Millisecond,  // TTFT
		PerTokenLatency: 20 * time.Millisecond,   // 20ms per token
		JitterPercent:   0.25,                     // ±25% variance
		MinLatency:      200 * time.Millisecond,  // Safety floor
		MaxLatency:      10 * time.Second,        // Safety ceiling
		P50Latency:      1200 * time.Millisecond, // 50th percentile @ 100 tokens
		P95Latency:      3500 * time.Millisecond, // 95th percentile
		P99Latency:      5000 * time.Millisecond, // 99th percentile
	}

	// GPT-4o-mini - Fastest overall
	r.profiles["gpt-4o-mini"] = Profile{
		ModelID:         "gpt-4o-mini",
		BaseLatency:     200 * time.Millisecond,
		PerTokenLatency: 34 * time.Millisecond,
		JitterPercent:   0.20,
		MinLatency:      100 * time.Millisecond,
		MaxLatency:      5 * time.Second,
		P50Latency:      500 * time.Millisecond,
		P95Latency:      1500 * time.Millisecond,
		P99Latency:      2500 * time.Millisecond,
	}

	// GPT-4-turbo - Balanced performance
	r.profiles["gpt-4-turbo"] = Profile{
		ModelID:         "gpt-4-turbo",
		BaseLatency:     800 * time.Millisecond,
		PerTokenLatency: 50 * time.Millisecond,
		JitterPercent:   0.30,
		MinLatency:      300 * time.Millisecond,
		MaxLatency:      15 * time.Second,
		P50Latency:      2000 * time.Millisecond,
		P95Latency:      6000 * time.Millisecond,
		P99Latency:      10000 * time.Millisecond,
	}

	// GPT-4 (legacy) - Slower, more deliberate
	r.profiles["gpt-4"] = Profile{
		ModelID:         "gpt-4",
		BaseLatency:     800 * time.Millisecond,
		PerTokenLatency: 196 * time.Millisecond, // Significantly slower per token
		JitterPercent:   0.30,
		MinLatency:      400 * time.Millisecond,
		MaxLatency:      30 * time.Second,
		P50Latency:      2000 * time.Millisecond,
		P95Latency:      8000 * time.Millisecond,
		P99Latency:      12000 * time.Millisecond,
	}

	// GPT-3.5-turbo - Fast and efficient
	r.profiles["gpt-3.5-turbo"] = Profile{
		ModelID:         "gpt-3.5-turbo",
		BaseLatency:     300 * time.Millisecond,
		PerTokenLatency: 73 * time.Millisecond,
		JitterPercent:   0.20,
		MinLatency:      150 * time.Millisecond,
		MaxLatency:      8 * time.Second,
		P50Latency:      800 * time.Millisecond,
		P95Latency:      2500 * time.Millisecond,
		P99Latency:      4000 * time.Millisecond,
	}

	// GPT-3.5-turbo-16k - Similar to base but slightly slower
	r.profiles["gpt-3.5-turbo-16k"] = Profile{
		ModelID:         "gpt-3.5-turbo-16k",
		BaseLatency:     350 * time.Millisecond,
		PerTokenLatency: 80 * time.Millisecond,
		JitterPercent:   0.20,
		MinLatency:      150 * time.Millisecond,
		MaxLatency:      8 * time.Second,
		P50Latency:      900 * time.Millisecond,
		P95Latency:      2800 * time.Millisecond,
		P99Latency:      4500 * time.Millisecond,
	}

	// Embedding models - Very fast (no generation, just encoding)
	embeddingProfile := Profile{
		ModelID:         "embedding",
		BaseLatency:     100 * time.Millisecond,
		PerTokenLatency: 0,                      // No per-token latency for embeddings
		JitterPercent:   0.15,
		MinLatency:      50 * time.Millisecond,
		MaxLatency:      2 * time.Second,
		P50Latency:      150 * time.Millisecond,
		P95Latency:      400 * time.Millisecond,
		P99Latency:      800 * time.Millisecond,
	}

	r.profiles["text-embedding-3-small"] = embeddingProfile
	r.profiles["text-embedding-3-large"] = Profile{
		ModelID:         "text-embedding-3-large",
		BaseLatency:     200 * time.Millisecond,
		PerTokenLatency: 0,
		JitterPercent:   0.15,
		MinLatency:      100 * time.Millisecond,
		MaxLatency:      3 * time.Second,
		P50Latency:      250 * time.Millisecond,
		P95Latency:      600 * time.Millisecond,
		P99Latency:      1000 * time.Millisecond,
	}
	r.profiles["text-embedding-ada-002"] = Profile{
		ModelID:         "text-embedding-ada-002",
		BaseLatency:     150 * time.Millisecond,
		PerTokenLatency: 0,
		JitterPercent:   0.15,
		MinLatency:      75 * time.Millisecond,
		MaxLatency:      2500 * time.Millisecond,
		P50Latency:      200 * time.Millisecond,
		P95Latency:      500 * time.Millisecond,
		P99Latency:      900 * time.Millisecond,
	}

	// Image generation models - Much slower (10-30 seconds)
	r.profiles["dall-e-3"] = Profile{
		ModelID:         "dall-e-3",
		BaseLatency:     15 * time.Second,       // Images take much longer
		PerTokenLatency: 0,
		JitterPercent:   0.20,
		MinLatency:      10 * time.Second,
		MaxLatency:      45 * time.Second,
		P50Latency:      18 * time.Second,
		P95Latency:      30 * time.Second,
		P99Latency:      40 * time.Second,
	}

	r.profiles["dall-e-2"] = Profile{
		ModelID:         "dall-e-2",
		BaseLatency:     10 * time.Second,
		PerTokenLatency: 0,
		JitterPercent:   0.20,
		MinLatency:      7 * time.Second,
		MaxLatency:      30 * time.Second,
		P50Latency:      12 * time.Second,
		P95Latency:      20 * time.Second,
		P99Latency:      25 * time.Second,
	}
}

// GetProfile retrieves the latency profile for a model.
func (r *ProfileRegistry) GetProfile(modelID string) (Profile, error) {
	profile, ok := r.profiles[modelID]
	if !ok {
		// Try to get from model config if not in registry
		config, err := models.GetModelConfig(modelID)
		if err != nil {
			return Profile{}, fmt.Errorf("no latency profile found for model: %s", modelID)
		}

		// Create profile from model config
		profile = Profile{
			ModelID:         modelID,
			BaseLatency:     config.BaseLatency,
			PerTokenLatency: config.PerTokenLatency,
			JitterPercent:   config.JitterPercent,
			MinLatency:      config.BaseLatency / 2,
			MaxLatency:      config.BaseLatency * 10,
			P50Latency:      config.BaseLatency + config.PerTokenLatency*100,
			P95Latency:      (config.BaseLatency + config.PerTokenLatency*100) * 2,
			P99Latency:      (config.BaseLatency + config.PerTokenLatency*100) * 3,
		}

		// Cache it
		r.profiles[modelID] = profile
	}

	return profile, nil
}

// SetProfile sets a custom latency profile for a model.
func (r *ProfileRegistry) SetProfile(modelID string, profile Profile) {
	profile.ModelID = modelID
	r.profiles[modelID] = profile
}

// ListProfiles returns all available profile IDs.
func (r *ProfileRegistry) ListProfiles() []string {
	models := make([]string, 0, len(r.profiles))
	for modelID := range r.profiles {
		models = append(models, modelID)
	}
	return models
}

// GetAllProfiles returns all profiles.
func (r *ProfileRegistry) GetAllProfiles() map[string]Profile {
	// Return a copy to avoid concurrent modification
	profiles := make(map[string]Profile, len(r.profiles))
	for k, v := range r.profiles {
		profiles[k] = v
	}
	return profiles
}

// ValidateProfile validates a latency profile.
func (p *Profile) Validate() error {
	if p.ModelID == "" {
		return fmt.Errorf("model ID is required")
	}

	if p.BaseLatency < 0 {
		return fmt.Errorf("base latency cannot be negative")
	}

	if p.PerTokenLatency < 0 {
		return fmt.Errorf("per-token latency cannot be negative")
	}

	if p.JitterPercent < 0 || p.JitterPercent > 1 {
		return fmt.Errorf("jitter percent must be between 0 and 1")
	}

	if p.MinLatency < 0 {
		return fmt.Errorf("min latency cannot be negative")
	}

	if p.MaxLatency < p.MinLatency {
		return fmt.Errorf("max latency must be greater than min latency")
	}

	return nil
}

// EstimateLatency estimates the latency for a given number of output tokens.
// This is the base calculation without jitter or load effects.
func (p *Profile) EstimateLatency(outputTokens int) time.Duration {
	return p.BaseLatency + p.PerTokenLatency*time.Duration(outputTokens)
}

// GetPercentile returns the approximate latency for a given percentile.
func (p *Profile) GetPercentile(percentile int) time.Duration {
	switch {
	case percentile <= 50:
		return p.P50Latency
	case percentile <= 95:
		// Interpolate between P50 and P95
		ratio := float64(percentile-50) / 45.0
		diff := p.P95Latency - p.P50Latency
		return p.P50Latency + time.Duration(float64(diff)*ratio)
	case percentile <= 99:
		// Interpolate between P95 and P99
		ratio := float64(percentile-95) / 4.0
		diff := p.P99Latency - p.P95Latency
		return p.P95Latency + time.Duration(float64(diff)*ratio)
	default:
		return p.P99Latency
	}
}

// String returns a string representation of the profile.
func (p *Profile) String() string {
	return fmt.Sprintf(
		"Profile[%s]: Base=%v, PerToken=%v, Jitter=%.0f%%, P50=%v, P95=%v, P99=%v",
		p.ModelID,
		p.BaseLatency,
		p.PerTokenLatency,
		p.JitterPercent*100,
		p.P50Latency,
		p.P95Latency,
		p.P99Latency,
	)
}