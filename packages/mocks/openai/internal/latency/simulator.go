// Package latency provides latency simulation.
// This file implements the main latency simulator with production-realistic delays.
package latency

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/metrics"
)

// Simulator simulates production-realistic API latencies.
type Simulator struct {
	// registry contains latency profiles for all models
	registry *ProfileRegistry

	// jitter calculates random variance
	jitter *JitterCalculator

	// enabled controls whether simulation is active
	enabled atomic.Bool

	// loadMultiplier simulates increased latency during high load
	loadMultiplier atomic.Value // float64

	// peakHours defines hours (UTC) when load is simulated as high
	peakHours map[int]bool

	// stats tracks simulation statistics
	totalSimulations atomic.Int64
	totalDelay       atomic.Int64 // in milliseconds
}

// SimulatorConfig configures the latency simulator.
type SimulatorConfig struct {
	// Enabled controls whether latency simulation is active
	Enabled bool

	// EnableJitter enables random variance in latencies
	EnableJitter bool

	// JitterDistribution is the type of jitter distribution
	JitterDistribution JitterDistribution

	// EnableLoadSimulation simulates increased latency during peak hours
	EnableLoadSimulation bool

	// LoadMultiplier is the multiplier applied during peak hours (e.g., 1.3 = +30%)
	LoadMultiplier float64

	// PeakHours are the hours (UTC, 0-23) considered peak load
	PeakHours []int
}

// DefaultSimulatorConfig returns default configuration.
func DefaultSimulatorConfig() SimulatorConfig {
	return SimulatorConfig{
		Enabled:              true,
		EnableJitter:         true,
		JitterDistribution:   UniformJitter,
		EnableLoadSimulation: true,
		LoadMultiplier:       1.3, // +30% during peak
		PeakHours:            []int{9, 10, 11, 12, 13, 14, 15, 16, 17}, // 9 AM - 5 PM UTC
	}
}

// NewSimulator creates a new latency simulator.
func NewSimulator(config SimulatorConfig) *Simulator {
	s := &Simulator{
		registry:  NewProfileRegistry(),
		jitter:    NewJitterCalculator(config.EnableJitter, config.JitterDistribution),
		peakHours: make(map[int]bool),
	}

	s.enabled.Store(config.Enabled)
	s.loadMultiplier.Store(config.LoadMultiplier)

	// Index peak hours for O(1) lookup
	for _, hour := range config.PeakHours {
		s.peakHours[hour] = true
	}

	return s
}

// Simulate calculates and applies latency simulation for a request.
func (s *Simulator) Simulate(ctx context.Context, modelID string, outputTokens int) (time.Duration, error) {
	if !s.enabled.Load() {
		return 0, nil // No simulation
	}

	// Get latency profile
	profile, err := s.registry.GetProfile(modelID)
	if err != nil {
		return 0, fmt.Errorf("failed to get latency profile: %w", err)
	}

	// Calculate base latency (TTFT + per-token)
	baseLatency := profile.BaseLatency + profile.PerTokenLatency*time.Duration(outputTokens)

	// Apply jitter
	jitteredLatency := s.jitter.ApplyJitter(baseLatency, profile.JitterPercent)

	// Apply load multiplier if in peak hours
	finalLatency := jitteredLatency
	if s.isPeakHour() {
		multiplier := s.loadMultiplier.Load().(float64)
		finalLatency = time.Duration(float64(jitteredLatency) * multiplier)
	}

	// Enforce min/max bounds
	if finalLatency < profile.MinLatency {
		finalLatency = profile.MinLatency
	} else if finalLatency > profile.MaxLatency {
		finalLatency = profile.MaxLatency
	}

	// Record statistics
	s.totalSimulations.Add(1)
	s.totalDelay.Add(finalLatency.Milliseconds())

	// Record metrics
	metrics.RecordSimulatedLatency(modelID, finalLatency.Seconds())

	return finalLatency, nil
}

// SimulateAndSleep calculates latency and sleeps for that duration.
func (s *Simulator) SimulateAndSleep(ctx context.Context, modelID string, outputTokens int) error {
	latency, err := s.Simulate(ctx, modelID, outputTokens)
	if err != nil {
		return err
	}

	if latency == 0 {
		return nil // No sleep needed
	}

	// Sleep with context cancellation support
	select {
	case <-time.After(latency):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SimulateStreaming calculates per-chunk delays for streaming responses.
func (s *Simulator) SimulateStreaming(ctx context.Context, modelID string, numChunks int) ([]time.Duration, error) {
	if !s.enabled.Load() {
		return make([]time.Duration, numChunks), nil
	}

	// Get latency profile
	profile, err := s.registry.GetProfile(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latency profile: %w", err)
	}

	delays := make([]time.Duration, numChunks)

	// First chunk: base latency (TTFT)
	baseFirstChunk := profile.BaseLatency
	delays[0] = s.jitter.ApplyJitterRange(
		baseFirstChunk,
		profile.JitterPercent,
		profile.MinLatency,
		profile.MaxLatency,
	)

	// Apply load multiplier to first chunk if in peak hours
	if s.isPeakHour() {
		multiplier := s.loadMultiplier.Load().(float64)
		delays[0] = time.Duration(float64(delays[0]) * multiplier)
	}

	// Subsequent chunks: per-token latency with small jitter
	for i := 1; i < numChunks; i++ {
		baseChunkDelay := profile.PerTokenLatency

		// Add small random jitter to each chunk (±10%)
		delays[i] = s.jitter.ApplyJitterRange(
			baseChunkDelay,
			0.10, // Small jitter for chunks
			baseChunkDelay/2,
			baseChunkDelay*2,
		)
	}

	// Record statistics
	totalDelay := time.Duration(0)
	for _, delay := range delays {
		totalDelay += delay
	}
	s.totalSimulations.Add(1)
	s.totalDelay.Add(totalDelay.Milliseconds())

	// Record metrics
	metrics.RecordSimulatedLatency(modelID, totalDelay.Seconds())

	return delays, nil
}

// EstimateLatency estimates latency without applying it.
func (s *Simulator) EstimateLatency(ctx context.Context, modelID string, outputTokens int) (time.Duration, error) {
	if !s.enabled.Load() {
		return 0, nil
	}

	profile, err := s.registry.GetProfile(modelID)
	if err != nil {
		return 0, err
	}

	// Calculate without jitter or load effects (base estimate)
	return profile.EstimateLatency(outputTokens), nil
}

// isPeakHour checks if the current time is during peak hours.
func (s *Simulator) isPeakHour() bool {
	currentHour := time.Now().UTC().Hour()
	return s.peakHours[currentHour]
}

// Enable enables latency simulation.
func (s *Simulator) Enable() {
	s.enabled.Store(true)
}

// Disable disables latency simulation.
func (s *Simulator) Disable() {
	s.enabled.Store(false)
}

// IsEnabled returns whether simulation is enabled.
func (s *Simulator) IsEnabled() bool {
	return s.enabled.Load()
}

// SetLoadMultiplier sets the load multiplier for peak hours.
func (s *Simulator) SetLoadMultiplier(multiplier float64) {
	if multiplier < 1.0 {
		multiplier = 1.0 // Minimum is no effect
	}
	s.loadMultiplier.Store(multiplier)
}

// GetLoadMultiplier returns the current load multiplier.
func (s *Simulator) GetLoadMultiplier() float64 {
	return s.loadMultiplier.Load().(float64)
}

// SetPeakHours sets the peak hours for load simulation.
func (s *Simulator) SetPeakHours(hours []int) {
	s.peakHours = make(map[int]bool)
	for _, hour := range hours {
		if hour >= 0 && hour < 24 {
			s.peakHours[hour] = true
		}
	}
}

// GetPeakHours returns the current peak hours.
func (s *Simulator) GetPeakHours() []int {
	hours := make([]int, 0, len(s.peakHours))
	for hour := range s.peakHours {
		hours = append(hours, hour)
	}
	return hours
}

// GetProfile returns the latency profile for a model.
func (s *Simulator) GetProfile(modelID string) (Profile, error) {
	return s.registry.GetProfile(modelID)
}

// SetProfile sets a custom latency profile for a model.
func (s *Simulator) SetProfile(modelID string, profile Profile) {
	s.registry.SetProfile(modelID, profile)
}

// GetStats returns simulation statistics.
func (s *Simulator) GetStats() SimulatorStats {
	totalSims := s.totalSimulations.Load()
	totalDelayMs := s.totalDelay.Load()

	var avgDelay time.Duration
	if totalSims > 0 {
		avgDelay = time.Duration(totalDelayMs/totalSims) * time.Millisecond
	}

	return SimulatorStats{
		TotalSimulations: totalSims,
		TotalDelay:       time.Duration(totalDelayMs) * time.Millisecond,
		AverageDelay:     avgDelay,
		Enabled:          s.enabled.Load(),
		LoadMultiplier:   s.loadMultiplier.Load().(float64),
		IsPeakHour:       s.isPeakHour(),
	}
}

// ResetStats resets all statistics.
func (s *Simulator) ResetStats() {
	s.totalSimulations.Store(0)
	s.totalDelay.Store(0)
}

// SimulatorStats contains simulation statistics.
type SimulatorStats struct {
	TotalSimulations int64
	TotalDelay       time.Duration
	AverageDelay     time.Duration
	Enabled          bool
	LoadMultiplier   float64
	IsPeakHour       bool
}

// FormatStats returns a formatted string of statistics.
func (s *SimulatorStats) FormatStats() string {
	return fmt.Sprintf(
		"Simulations: %d, Total Delay: %v, Avg Delay: %v, Enabled: %v, Load: %.1fx, Peak: %v",
		s.TotalSimulations,
		s.TotalDelay,
		s.AverageDelay,
		s.Enabled,
		s.LoadMultiplier,
		s.IsPeakHour,
	)
}

// GetJitterCalculator returns the jitter calculator for configuration.
func (s *Simulator) GetJitterCalculator() *JitterCalculator {
	return s.jitter
}

// GetProfileRegistry returns the profile registry for configuration.
func (s *Simulator) GetProfileRegistry() *ProfileRegistry {
	return s.registry
}