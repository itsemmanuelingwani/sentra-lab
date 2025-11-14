// Package behavior provides production-realistic behavior simulation for API responses.
// This file implements context-aware error injection matching real API failure patterns.
package behavior

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// ErrorInjector simulates production error patterns.
// Errors are NOT purely random - they're context-aware based on load, quota, etc.
type ErrorInjector struct {
	// enabled controls whether error injection is active
	enabled atomic.Bool

	// baseErrorRate is the baseline error probability (0.0 to 1.0)
	baseErrorRate atomic.Value // float64

	// burstErrorRate is the error rate during high load
	burstErrorRate atomic.Value // float64

	// quotaErrorRate is the error rate when quota is high
	quotaErrorRate atomic.Value // float64

	// loadThreshold is the RPS threshold for considering load "high"
	loadThreshold atomic.Int64

	// quotaThreshold is the percentage of quota used to trigger high error rate
	quotaThreshold atomic.Value // float64

	// errorDefinitions defines the types of errors and their probabilities
	errorDefs []ErrorDefinition

	// stats tracks error injection statistics
	totalChecks atomic.Int64
	totalErrors atomic.Int64
}

// ErrorDefinition defines an error type and its selection probability.
type ErrorDefinition struct {
	Type        models.ErrorType
	StatusCode  int
	Probability float64 // Weight for selection (0.0 to 1.0)
	RetryAfter  int     // Seconds to wait before retry
}

// ErrorInjectorConfig configures error injection behavior.
type ErrorInjectorConfig struct {
	Enabled         bool
	BaseErrorRate   float64
	BurstErrorRate  float64
	QuotaErrorRate  float64
	LoadThreshold   int64
	QuotaThreshold  float64
	ErrorDefinitions []ErrorDefinition
}

// DefaultErrorInjectorConfig returns production-realistic configuration.
func DefaultErrorInjectorConfig() ErrorInjectorConfig {
	return ErrorInjectorConfig{
		Enabled:        true,
		BaseErrorRate:  0.005,  // 0.5% baseline
		BurstErrorRate: 0.05,   // 5% during burst traffic
		QuotaErrorRate: 0.10,   // 10% when quota high
		LoadThreshold:  100,    // 100 requests/second
		QuotaThreshold: 0.90,   // 90% of quota used
		ErrorDefinitions: []ErrorDefinition{
			{
				Type:        models.ErrorTypeRateLimit,
				StatusCode:  429,
				Probability: 0.60, // 60% of errors are rate limits
				RetryAfter:  60,
			},
			{
				Type:        models.ErrorTypeServerError,
				StatusCode:  500,
				Probability: 0.30, // 30% are server errors
				RetryAfter:  10,
			},
			{
				Type:        models.ErrorTypeServiceUnavailable,
				StatusCode:  503,
				Probability: 0.10, // 10% are service unavailable
				RetryAfter:  30,
			},
		},
	}
}

// NewErrorInjector creates a new error injector.
func NewErrorInjector(config ErrorInjectorConfig) *ErrorInjector {
	injector := &ErrorInjector{
		errorDefs: config.ErrorDefinitions,
	}

	injector.enabled.Store(config.Enabled)
	injector.baseErrorRate.Store(config.BaseErrorRate)
	injector.burstErrorRate.Store(config.BurstErrorRate)
	injector.quotaErrorRate.Store(config.QuotaErrorRate)
	injector.loadThreshold.Store(config.LoadThreshold)
	injector.quotaThreshold.Store(config.QuotaThreshold)

	return injector
}

// ShouldInjectError determines if an error should be injected based on context.
func (ei *ErrorInjector) ShouldInjectError(ctx context.Context, currentRPS int64, quotaUsed float64) (*models.APIError, bool) {
	ei.totalChecks.Add(1)

	if !ei.enabled.Load() {
		return nil, false
	}

	// Determine error rate based on context
	errorRate := ei.calculateErrorRate(currentRPS, quotaUsed)

	// Roll dice
	if rand.Float64() >= errorRate {
		return nil, false // No error
	}

	// Select error type based on probabilities
	errorDef := ei.selectErrorType()

	// Create error
	apiError := ei.createError(errorDef)

	ei.totalErrors.Add(1)

	return &apiError, true
}

// calculateErrorRate determines error rate based on current conditions.
func (ei *ErrorInjector) calculateErrorRate(currentRPS int64, quotaUsed float64) float64 {
	baseRate := ei.baseErrorRate.Load().(float64)

	// Check if in burst traffic conditions
	if currentRPS > ei.loadThreshold.Load() {
		return ei.burstErrorRate.Load().(float64)
	}

	// Check if quota pressure is high
	if quotaUsed > ei.quotaThreshold.Load().(float64) {
		return ei.quotaErrorRate.Load().(float64)
	}

	// Return baseline error rate
	return baseRate
}

// selectErrorType selects an error type based on weighted probabilities.
func (ei *ErrorInjector) selectErrorType() ErrorDefinition {
	if len(ei.errorDefs) == 0 {
		// Fallback to default error
		return ErrorDefinition{
			Type:        models.ErrorTypeServerError,
			StatusCode:  500,
			Probability: 1.0,
			RetryAfter:  10,
		}
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, def := range ei.errorDefs {
		totalWeight += def.Probability
	}

	// Weighted random selection
	r := rand.Float64() * totalWeight
	cumulative := 0.0

	for _, def := range ei.errorDefs {
		cumulative += def.Probability
		if r <= cumulative {
			return def
		}
	}

	// Fallback to last definition
	return ei.errorDefs[len(ei.errorDefs)-1]
}

// createError creates an API error from a definition.
func (ei *ErrorInjector) createError(def ErrorDefinition) models.APIError {
	var message string

	switch def.Type {
	case models.ErrorTypeRateLimit:
		message = "Rate limit reached for requests. Please retry after a brief wait."
	case models.ErrorTypeServerError:
		message = "The server had an error while processing your request. Sorry about that!"
	case models.ErrorTypeServiceUnavailable:
		message = "The server is currently overloaded with other requests. Please retry after a brief wait."
	default:
		message = "An error occurred while processing your request."
	}

	return models.APIError{
		Type:       def.Type,
		Message:    message,
		StatusCode: def.StatusCode,
		RetryAfter: def.RetryAfter,
	}
}

// Enable enables error injection.
func (ei *ErrorInjector) Enable() {
	ei.enabled.Store(true)
}

// Disable disables error injection.
func (ei *ErrorInjector) Disable() {
	ei.enabled.Store(false)
}

// IsEnabled returns whether error injection is enabled.
func (ei *ErrorInjector) IsEnabled() bool {
	return ei.enabled.Load()
}

// SetBaseErrorRate sets the baseline error rate.
func (ei *ErrorInjector) SetBaseErrorRate(rate float64) {
	if rate < 0 {
		rate = 0
	} else if rate > 1 {
		rate = 1
	}
	ei.baseErrorRate.Store(rate)
}

// GetBaseErrorRate returns the current baseline error rate.
func (ei *ErrorInjector) GetBaseErrorRate() float64 {
	return ei.baseErrorRate.Load().(float64)
}

// SetBurstErrorRate sets the error rate during burst traffic.
func (ei *ErrorInjector) SetBurstErrorRate(rate float64) {
	if rate < 0 {
		rate = 0
	} else if rate > 1 {
		rate = 1
	}
	ei.burstErrorRate.Store(rate)
}

// SetLoadThreshold sets the RPS threshold for burst conditions.
func (ei *ErrorInjector) SetLoadThreshold(threshold int64) {
	ei.loadThreshold.Store(threshold)
}

// GetStats returns error injection statistics.
func (ei *ErrorInjector) GetStats() ErrorInjectorStats {
	totalChecks := ei.totalChecks.Load()
	totalErrors := ei.totalErrors.Load()

	var errorRate float64
	if totalChecks > 0 {
		errorRate = float64(totalErrors) / float64(totalChecks) * 100
	}

	return ErrorInjectorStats{
		TotalChecks:    totalChecks,
		TotalErrors:    totalErrors,
		ErrorRate:      errorRate,
		BaseErrorRate:  ei.baseErrorRate.Load().(float64),
		BurstErrorRate: ei.burstErrorRate.Load().(float64),
		Enabled:        ei.enabled.Load(),
	}
}

// ResetStats resets statistics.
func (ei *ErrorInjector) ResetStats() {
	ei.totalChecks.Store(0)
	ei.totalErrors.Store(0)
}

// ErrorInjectorStats contains error injection statistics.
type ErrorInjectorStats struct {
	TotalChecks    int64
	TotalErrors    int64
	ErrorRate      float64
	BaseErrorRate  float64
	BurstErrorRate float64
	Enabled        bool
}