// Package tokenizer provides token counting.
// This file implements a caching layer for token counts to optimize performance.
package tokenizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/models"
	"github.com/sentra-lab/mocks/openai/internal/store"
)

// CachedTokenizer wraps a Tokenizer with caching to improve performance.
// Cache keys are based on message content + model to ensure accuracy.
type CachedTokenizer struct {
	// tokenizer is the underlying tokenizer
	tokenizer *Tokenizer

	// cache is the storage backend for cached token counts
	cache store.Storage

	// ttl is the time-to-live for cache entries
	ttl time.Duration

	// stats tracks cache performance
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
}

// CachedTokenizerConfig contains configuration for cached tokenizer.
type CachedTokenizerConfig struct {
	// TTL is the cache entry time-to-live (default: 5 minutes)
	TTL time.Duration

	// CacheSize is the maximum number of cache entries (for memory store)
	CacheSize int
}

// DefaultCachedTokenizerConfig returns default configuration.
func DefaultCachedTokenizerConfig() CachedTokenizerConfig {
	return CachedTokenizerConfig{
		TTL:       5 * time.Minute,
		CacheSize: 10000,
	}
}

// NewCachedTokenizer creates a new cached tokenizer.
func NewCachedTokenizer(tokenizer *Tokenizer, cache store.Storage, config CachedTokenizerConfig) *CachedTokenizer {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}

	return &CachedTokenizer{
		tokenizer: tokenizer,
		cache:     cache,
		ttl:       config.TTL,
	}
}

// Count counts tokens with caching.
func (c *CachedTokenizer) Count(ctx context.Context, messages []models.Message, model string) (int, error) {
	// Generate cache key
	cacheKey := c.generateCacheKey(messages, model)

	// Try to get from cache
	if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
		if count, ok := cached.(int64); ok {
			c.cacheHits.Add(1)
			return int(count), nil
		}
	}

	// Cache miss - count tokens
	c.cacheMisses.Add(1)
	count, err := c.tokenizer.Count(ctx, messages, model)
	if err != nil {
		return 0, err
	}

	// Store in cache (ignore errors - cache is best-effort)
	c.cache.Set(ctx, cacheKey, int64(count), c.ttl)

	return count, nil
}

// CountText counts tokens in text with caching.
func (c *CachedTokenizer) CountText(ctx context.Context, text string, model string) (int, error) {
	// Generate cache key
	cacheKey := c.generateTextCacheKey(text, model)

	// Try to get from cache
	if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
		if count, ok := cached.(int64); ok {
			c.cacheHits.Add(1)
			return int(count), nil
		}
	}

	// Cache miss - count tokens
	c.cacheMisses.Add(1)
	count, err := c.tokenizer.CountText(ctx, text, model)
	if err != nil {
		return 0, err
	}

	// Store in cache
	c.cache.Set(ctx, cacheKey, int64(count), c.ttl)

	return count, nil
}

// CountWithMetadata counts tokens and returns metadata (with caching).
func (c *CachedTokenizer) CountWithMetadata(ctx context.Context, messages []models.Message, model string) (TokenCount, error) {
	// Get cached count
	inputTokens, err := c.Count(ctx, messages, model)
	if err != nil {
		return TokenCount{}, err
	}

	// Get model configuration for metadata
	config, err := models.GetModelConfig(model)
	if err != nil {
		return TokenCount{}, fmt.Errorf("unknown model: %w", err)
	}

	// Calculate remaining tokens
	remainingTokens := config.ContextWindow - inputTokens

	return TokenCount{
		InputTokens:           inputTokens,
		EstimatedOutputTokens: 0, // Caller should set this
		TotalTokens:           inputTokens,
		Model:                 model,
		Encoding:              config.Encoding,
		ContextWindow:         config.ContextWindow,
		RemainingTokens:       remainingTokens,
	}, nil
}

// EstimateOutputTokens estimates output tokens (delegates to underlying tokenizer).
func (c *CachedTokenizer) EstimateOutputTokens(ctx context.Context, maxTokens int, model string) (int, error) {
	return c.tokenizer.EstimateOutputTokens(ctx, maxTokens, model)
}

// ValidateContextLength validates context length (delegates to underlying tokenizer).
func (c *CachedTokenizer) ValidateContextLength(ctx context.Context, inputTokens, outputTokens int, model string) error {
	return c.tokenizer.ValidateContextLength(ctx, inputTokens, outputTokens, model)
}

// GetStats returns combined statistics from cache and tokenizer.
func (c *CachedTokenizer) GetStats(ctx context.Context) (CounterStats, error) {
	// Get base stats from tokenizer
	stats, err := c.tokenizer.GetStats(ctx)
	if err != nil {
		return CounterStats{}, err
	}

	// Add cache stats
	cacheHits := c.cacheHits.Load()
	cacheMisses := c.cacheMisses.Load()
	total := cacheHits + cacheMisses

	stats.CacheHits = cacheHits
	stats.CacheMisses = cacheMisses

	if total > 0 {
		stats.CacheHitRate = float64(cacheHits) / float64(total) * 100
	}

	return stats, nil
}

// ResetStats resets all statistics.
func (c *CachedTokenizer) ResetStats(ctx context.Context) error {
	c.cacheHits.Store(0)
	c.cacheMisses.Store(0)
	return c.tokenizer.ResetStats(ctx)
}

// ClearCache clears all cached token counts.
func (c *CachedTokenizer) ClearCache(ctx context.Context) error {
	// Get all cache keys with token prefix
	keys, err := c.cache.Keys(ctx, "token:*")
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	// Delete all token cache entries
	if len(keys) > 0 {
		return c.cache.DeleteMulti(ctx, keys)
	}

	return nil
}

// generateCacheKey generates a cache key for messages.
// Key format: token:<hash(messages+model)>
func (c *CachedTokenizer) generateCacheKey(messages []models.Message, model string) string {
	h := sha256.New()

	// Hash model
	h.Write([]byte(model))
	h.Write([]byte(":"))

	// Hash messages (role + content)
	for _, msg := range messages {
		h.Write([]byte(msg.Role))
		h.Write([]byte(":"))
		h.Write([]byte(msg.Content))
		h.Write([]byte("|"))
	}

	hash := hex.EncodeToString(h.Sum(nil))
	return "token:" + hash[:32] // Use first 32 chars of hash
}

// generateTextCacheKey generates a cache key for plain text.
func (c *CachedTokenizer) generateTextCacheKey(text string, model string) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write([]byte(":"))
	h.Write([]byte(text))

	hash := hex.EncodeToString(h.Sum(nil))
	return "token:text:" + hash[:32]
}

// Close releases resources.
func (c *CachedTokenizer) Close() error {
	return c.tokenizer.Close()
}

// Compile-time interface checks
var _ Counter = (*CachedTokenizer)(nil)
var _ StatsProvider = (*CachedTokenizer)(nil)