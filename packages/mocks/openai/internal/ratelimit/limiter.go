// Package ratelimit provides rate limiting.
// This file implements the main rate limiter with per-API-key tracking.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sentra-lab/mocks/openai/internal/metrics"
	"github.com/sentra-lab/mocks/openai/internal/store"
)

// Limiter manages rate limiting for multiple API keys.
// Each API key has its own dual token bucket (RPM + TPM).
type Limiter struct {
	// tierRegistry provides rate limit configurations
	tierRegistry *TierRegistry

	// storage persists rate limit state
	storage store.RateLimitStorage

	// buckets caches in-memory token buckets per API key
	buckets map[string]*keyLimiter
	bucketsKey sync.RWMutex

	// enabled controls whether rate limiting is active
	enabled atomic.Bool

	// stats tracks rate limiting statistics
	totalChecks atomic.Int64
	totalAllowed atomic.Int64
	totalDenied atomic.Int64
}

// keyLimiter contains rate limiters for a single API key.
type keyLimiter struct {
	// apiKey is the API key
	apiKey string

	// tier is the rate limit tier
	tier string

	// limiters maps model IDs to dual token buckets
	limiters map[string]*DualTokenBucket
	mu       sync.RWMutex
}

// LimiterConfig configures the rate limiter.
type LimiterConfig struct {
	// Enabled controls whether rate limiting is active
	Enabled bool

	// TierRegistry provides tier configurations
	TierRegistry *TierRegistry

	// Storage persists rate limit state
	Storage store.RateLimitStorage

	// DefaultTier is used when API key has no specific tier
	DefaultTier string
}

// NewLimiter creates a new rate limiter.
func NewLimiter(config LimiterConfig) *Limiter {
	limiter := &Limiter{
		tierRegistry: config.TierRegistry,
		storage:      config.Storage,
		buckets:      make(map[string]*keyLimiter),
	}

	limiter.enabled.Store(config.Enabled)

	return limiter
}

// Allow checks if a request is allowed based on rate limits.
func (l *Limiter) Allow(ctx context.Context, apiKey string, modelID string, estimatedTokens int) (*LimitCheckResult, error) {
	l.totalChecks.Add(1)

	if !l.enabled.Load() {
		// Rate limiting disabled - always allow
		return &LimitCheckResult{
			Allowed:        true,
			LimitingFactor: NoLimit,
		}, nil
	}

	// Get or create limiter for this API key
	keyLim, err := l.getKeyLimiter(apiKey)
	if err != nil {
		return nil, err
	}

	// Get or create bucket for this model
	bucket, err := keyLim.getBucket(modelID, l.tierRegistry)
	if err != nil {
		return nil, err
	}

	// Check rate limits
	result := bucket.Allow(estimatedTokens)

	// Build result
	checkResult := &LimitCheckResult{
		Allowed:          result.Allowed,
		RequestsRemaining: result.RequestsResult.Remaining,
		TokensRemaining:   result.TokensResult.Remaining,
		RequestsResetIn:   result.RequestsResult.ResetIn,
		TokensResetIn:     result.TokensResult.ResetIn,
		LimitingFactor:    result.LimitingFactor,
		APIKey:            apiKey,
		ModelID:           modelID,
		Tier:              keyLim.tier,
	}

	// Update statistics
	if result.Allowed {
		l.totalAllowed.Add(1)
	} else {
		l.totalDenied.Add(1)

		// Record metrics
		limitType := string(result.LimitingFactor)
		metrics.RecordRateLimitHit(apiKey, limitType, checkResult.RequestsRemaining)
	}

	// Log rate limit hit
	if !result.Allowed {
		metrics.LogRateLimitExceeded(ctx, apiKey, string(result.LimitingFactor), result.RequestsResult.ResetIn)
	}

	return checkResult, nil
}

// getKeyLimiter retrieves or creates a limiter for an API key.
func (l *Limiter) getKeyLimiter(apiKey string) (*keyLimiter, error) {
	// Fast path: check if already cached
	l.bucketsKey.RLock()
	kl, ok := l.buckets[apiKey]
	l.bucketsKey.RUnlock()

	if ok {
		return kl, nil
	}

	// Slow path: create new key limiter
	l.bucketsKey.Lock()
	defer l.bucketsKey.Unlock()

	// Double-check after acquiring write lock
	if kl, ok := l.buckets[apiKey]; ok {
		return kl, nil
	}

	// Determine tier for this API key
	tier := l.determineTier(apiKey)

	// Create new key limiter
	kl = &keyLimiter{
		apiKey:   apiKey,
		tier:     tier,
		limiters: make(map[string]*DualTokenBucket),
	}

	l.buckets[apiKey] = kl

	return kl, nil
}

// determineTier determines the tier for an API key.
// In production, this would lookup the tier from a database.
func (l *Limiter) determineTier(apiKey string) string {
	// For mock, we can use a simple mapping or default tier
	// In production, this would query a database or config
	return l.tierRegistry.GetDefaultTier()
}

// getBucket retrieves or creates a dual token bucket for a model.
func (kl *keyLimiter) getBucket(modelID string, registry *TierRegistry) (*DualTokenBucket, error) {
	// Fast path: check if already cached
	kl.mu.RLock()
	bucket, ok := kl.limiters[modelID]
	kl.mu.RUnlock()

	if ok {
		return bucket, nil
	}

	// Slow path: create new bucket
	kl.mu.Lock()
	defer kl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, ok := kl.limiters[modelID]; ok {
		return bucket, nil
	}

	// Get rate limits for this model in this tier
	limits := registry.GetModelLimitOrDefault(kl.tier, modelID)

	// Create dual token bucket
	bucket = NewDualTokenBucket(limits.RPM, limits.TPM)
	kl.limiters[modelID] = bucket

	return bucket, nil
}

// SetTier sets the tier for an API key.
func (l *Limiter) SetTier(apiKey string, tier string) error {
	// Validate tier exists
	if _, err := l.tierRegistry.GetTier(tier); err != nil {
		return fmt.Errorf("invalid tier: %w", err)
	}

	l.bucketsKey.Lock()
	defer l.bucketsKey.Unlock()

	// If key limiter exists, update its tier and clear buckets
	if kl, ok := l.buckets[apiKey]; ok {
		kl.tier = tier
		kl.mu.Lock()
		kl.limiters = make(map[string]*DualTokenBucket)
		kl.mu.Unlock()
	}

	return nil
}

// GetLimitInfo returns rate limit information for an API key and model.
func (l *Limiter) GetLimitInfo(apiKey string, modelID string) (*LimitInfo, error) {
	keyLim, err := l.getKeyLimiter(apiKey)
	if err != nil {
		return nil, err
	}

	bucket, err := keyLim.getBucket(modelID, l.tierRegistry)
	if err != nil {
		return nil, err
	}

	state := bucket.GetState()
	limits := l.tierRegistry.GetModelLimitOrDefault(keyLim.tier, modelID)

	return &LimitInfo{
		APIKey:            apiKey,
		ModelID:           modelID,
		Tier:              keyLim.tier,
		RequestLimit:      limits.RPM,
		TokenLimit:        limits.TPM,
		RequestsRemaining: state.RequestBucket.Available,
		TokensRemaining:   state.TokenBucket.Available,
		RequestsFillRate:  int(state.RequestBucket.RefillPerMin),
		TokensFillRate:    int(state.TokenBucket.RefillPerMin),
	}, nil
}

// Reset resets rate limits for an API key.
func (l *Limiter) Reset(apiKey string) error {
	l.bucketsKey.Lock()
	defer l.bucketsKey.Unlock()

	// Remove from cache (will be recreated on next request)
	delete(l.buckets, apiKey)

	return nil
}

// ResetAll resets all rate limits.
func (l *Limiter) ResetAll() {
	l.bucketsKey.Lock()
	defer l.bucketsKey.Unlock()

	l.buckets = make(map[string]*keyLimiter)
	l.totalChecks.Store(0)
	l.totalAllowed.Store(0)
	l.totalDenied.Store(0)
}

// Enable enables rate limiting.
func (l *Limiter) Enable() {
	l.enabled.Store(true)
}

// Disable disables rate limiting.
func (l *Limiter) Disable() {
	l.enabled.Store(false)
}

// IsEnabled returns whether rate limiting is enabled.
func (l *Limiter) IsEnabled() bool {
	return l.enabled.Load()
}

// GetStats returns rate limiter statistics.
func (l *Limiter) GetStats() LimiterStats {
	totalChecks := l.totalChecks.Load()
	totalAllowed := l.totalAllowed.Load()
	totalDenied := l.totalDenied.Load()

	var allowRate float64
	if totalChecks > 0 {
		allowRate = float64(totalAllowed) / float64(totalChecks) * 100
	}

	l.bucketsKey.RLock()
	numKeys := len(l.buckets)
	l.bucketsKey.RUnlock()

	return LimiterStats{
		TotalChecks:  totalChecks,
		TotalAllowed: totalAllowed,
		TotalDenied:  totalDenied,
		AllowRate:    allowRate,
		TrackedKeys:  numKeys,
		Enabled:      l.enabled.Load(),
	}
}

// LimitCheckResult contains the result of a rate limit check.
type LimitCheckResult struct {
	Allowed           bool
	RequestsRemaining int
	TokensRemaining   int
	RequestsResetIn   time.Duration
	TokensResetIn     time.Duration
	LimitingFactor    LimitType
	APIKey            string
	ModelID           string
	Tier              string
}

// LimitInfo contains rate limit information.
type LimitInfo struct {
	APIKey            string
	ModelID           string
	Tier              string
	RequestLimit      int
	TokenLimit        int
	RequestsRemaining int
	TokensRemaining   int
	RequestsFillRate  int
	TokensFillRate    int
}

// LimiterStats contains statistics about the rate limiter.
type LimiterStats struct {
	TotalChecks  int64
	TotalAllowed int64
	TotalDenied  int64
	AllowRate    float64
	TrackedKeys  int
	Enabled      bool
}

// FormatStats returns a formatted string of statistics.
func (ls LimiterStats) FormatStats() string {
	return fmt.Sprintf(
		"Checks: %d, Allowed: %d, Denied: %d (%.1f%% allow), Keys: %d, Enabled: %v",
		ls.TotalChecks,
		ls.TotalAllowed,
		ls.TotalDenied,
		ls.AllowRate,
		ls.TrackedKeys,
		ls.Enabled,
	)
}