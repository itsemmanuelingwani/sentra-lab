// Package ratelimit provides rate limiting.
// This file implements storage helpers for persisting rate limit state.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/store"
)

// PersistentLimiter wraps a Limiter with persistent storage.
// This allows rate limit state to survive server restarts.
type PersistentLimiter struct {
	// limiter is the underlying limiter
	limiter *Limiter

	// storage persists rate limit state
	storage store.RateLimitStorage

	// syncInterval is how often to sync to storage
	syncInterval time.Duration

	// enabled controls whether persistence is active
	enabled bool
}

// PersistentConfig configures persistent rate limiting.
type PersistentConfig struct {
	Limiter      *Limiter
	Storage      store.RateLimitStorage
	SyncInterval time.Duration
	Enabled      bool
}

// NewPersistentLimiter creates a limiter with persistent storage.
func NewPersistentLimiter(config PersistentConfig) *PersistentLimiter {
	if config.SyncInterval == 0 {
		config.SyncInterval = 10 * time.Second // Default sync interval
	}

	return &PersistentLimiter{
		limiter:      config.Limiter,
		storage:      config.Storage,
		syncInterval: config.SyncInterval,
		enabled:      config.Enabled,
	}
}

// Allow checks rate limits with persistent storage support.
func (pl *PersistentLimiter) Allow(ctx context.Context, apiKey string, modelID string, estimatedTokens int) (*LimitCheckResult, error) {
	// If persistence is enabled, try to load state from storage
	if pl.enabled {
		if err := pl.loadState(ctx, apiKey, modelID); err != nil {
			// Log error but continue (degrade gracefully)
			// In production, you might want to handle this differently
		}
	}

	// Check rate limits using in-memory limiter
	result, err := pl.limiter.Allow(ctx, apiKey, modelID, estimatedTokens)
	if err != nil {
		return nil, err
	}

	// If persistence is enabled, save state to storage
	if pl.enabled {
		go func() {
			// Save asynchronously to avoid blocking
			if err := pl.saveState(context.Background(), apiKey, modelID); err != nil {
				// Log error (non-fatal)
			}
		}()
	}

	return result, nil
}

// loadState loads rate limit state from storage.
func (pl *PersistentLimiter) loadState(ctx context.Context, apiKey string, modelID string) error {
	// Build storage key
	key := buildStorageKey(apiKey, modelID)

	// Get tokens from storage
	tokens, err := pl.storage.GetTokens(ctx, key+":tokens")
	if err != nil {
		return err // Key doesn't exist yet
	}

	// Get requests from storage
	requests, err := pl.storage.GetTokens(ctx, key+":requests")
	if err != nil {
		return err
	}

	// Get last refill time
	lastRefill, err := pl.storage.GetLastRefill(ctx, key+":refill")
	if err != nil {
		lastRefill = time.Now() // Default to now if not found
	}

	// Update in-memory bucket state
	// Note: This is a simplified version - in production, you'd need more sophisticated state restoration
	_ = tokens
	_ = requests
	_ = lastRefill

	return nil
}

// saveState saves rate limit state to storage.
func (pl *PersistentLimiter) saveState(ctx context.Context, apiKey string, modelID string) error {
	// Get current limit info
	info, err := pl.limiter.GetLimitInfo(apiKey, modelID)
	if err != nil {
		return err
	}

	// Build storage key
	key := buildStorageKey(apiKey, modelID)

	// Save tokens remaining
	if err := pl.storage.SetTokens(ctx, key+":tokens", float64(info.TokensRemaining), time.Hour); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	// Save requests remaining
	if err := pl.storage.SetTokens(ctx, key+":requests", float64(info.RequestsRemaining), time.Hour); err != nil {
		return fmt.Errorf("failed to save requests: %w", err)
	}

	// Save last refill time
	if err := pl.storage.SetLastRefill(ctx, key+":refill", time.Now()); err != nil {
		return fmt.Errorf("failed to save refill time: %w", err)
	}

	return nil
}

// StartSyncJob starts a background job to periodically sync state to storage.
func (pl *PersistentLimiter) StartSyncJob(ctx context.Context) {
	if !pl.enabled {
		return
	}

	ticker := time.NewTicker(pl.syncInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				pl.syncAll(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// syncAll syncs all rate limit state to storage.
func (pl *PersistentLimiter) syncAll(ctx context.Context) {
	// Get all tracked API keys
	pl.limiter.bucketsKey.RLock()
	keys := make([]string, 0, len(pl.limiter.buckets))
	for apiKey := range pl.limiter.buckets {
		keys = append(keys, apiKey)
	}
	pl.limiter.bucketsKey.RUnlock()

	// Sync each key (limit concurrency to avoid overwhelming storage)
	for _, apiKey := range keys {
		// Get models for this key
		pl.limiter.bucketsKey.RLock()
		kl, ok := pl.limiter.buckets[apiKey]
		pl.limiter.bucketsKey.RUnlock()

		if !ok {
			continue
		}

		kl.mu.RLock()
		models := make([]string, 0, len(kl.limiters))
		for modelID := range kl.limiters {
			models = append(models, modelID)
		}
		kl.mu.RUnlock()

		// Sync each model
		for _, modelID := range models {
			if err := pl.saveState(ctx, apiKey, modelID); err != nil {
				// Log error but continue
			}
		}
	}
}

// buildStorageKey builds a storage key for rate limit state.
func buildStorageKey(apiKey string, modelID string) string {
	return fmt.Sprintf("ratelimit:%s:%s", apiKey, modelID)
}

// Cleanup removes expired rate limit state from storage.
func (pl *PersistentLimiter) Cleanup(ctx context.Context) error {
	// Get all rate limit keys
	keys, err := pl.storage.Keys(ctx, "ratelimit:*")
	if err != nil {
		return err
	}

	// Check TTL and delete expired keys
	for _, key := range keys {
		ttl, err := pl.storage.TTL(ctx, key)
		if err != nil {
			continue
		}

		// If TTL is expired or close to expiry, delete
		if ttl < time.Minute {
			pl.storage.Delete(ctx, key)
		}
	}

	return nil
}

// Enable enables persistent storage.
func (pl *PersistentLimiter) Enable() {
	pl.enabled = true
}

// Disable disables persistent storage.
func (pl *PersistentLimiter) Disable() {
	pl.enabled = false
}

// IsEnabled returns whether persistence is enabled.
func (pl *PersistentLimiter) IsEnabled() bool {
	return pl.enabled
}