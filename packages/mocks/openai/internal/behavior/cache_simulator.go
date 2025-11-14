// Package behavior provides behavior simulation.
// This file simulates response caching for faster repeated requests.
package behavior

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/metrics"
	"github.com/sentra-lab/mocks/openai/internal/models"
	"github.com/sentra-lab/mocks/openai/internal/store"
)

// CacheSimulator simulates response caching behavior.
// Production APIs cache similar requests for performance.
type CacheSimulator struct {
	// storage backs the cache
	storage store.Storage

	// enabled controls whether caching is active
	enabled atomic.Bool

	// ttl is the cache entry time-to-live
	ttl time.Duration

	// speedup is the latency reduction for cache hits (percentage)
	speedup atomic.Value // float64

	// stats tracks cache performance
	totalQueries atomic.Int64
	cacheHits    atomic.Int64
	cacheMisses  atomic.Int64
}

// CacheSimulatorConfig configures cache simulation.
type CacheSimulatorConfig struct {
	Enabled bool
	Storage store.Storage
	TTL     time.Duration
	Speedup float64 // 0.0 to 1.0 (e.g., 0.10 = 10% faster)
}

// NewCacheSimulator creates a new cache simulator.
func NewCacheSimulator(config CacheSimulatorConfig) *CacheSimulator {
	if config.TTL == 0 {
		config.TTL = 1 * time.Hour
	}
	if config.Speedup == 0 {
		config.Speedup = 0.10 // 10% faster by default
	}

	simulator := &CacheSimulator{
		storage: config.Storage,
		ttl:     config.TTL,
	}

	simulator.enabled.Store(config.Enabled)
	simulator.speedup.Store(config.Speedup)

	return simulator
}

// CheckCache checks if a request is cached.
func (cs *CacheSimulator) CheckCache(ctx context.Context, req models.ChatCompletionRequest) (bool, error) {
	cs.totalQueries.Add(1)

	if !cs.enabled.Load() {
		cs.cacheMisses.Add(1)
		return false, nil
	}

	// Generate cache key
	key := cs.generateCacheKey(req)

	// Check if cached
	_, err := cs.storage.Get(ctx, key)
	if err != nil {
		// Cache miss
		cs.cacheMisses.Add(1)
		metrics.RecordCacheMiss("response")
		return false, nil
	}

	// Cache hit
	cs.cacheHits.Add(1)
	metrics.RecordCacheHit("response")
	return true, nil
}

// StoreInCache stores a response in cache.
func (cs *CacheSimulator) StoreInCache(ctx context.Context, req models.ChatCompletionRequest, resp models.ChatCompletionResponse) error {
	if !cs.enabled.Load() {
		return nil
	}

	key := cs.generateCacheKey(req)
	return cs.storage.Set(ctx, key, resp, cs.ttl)
}

// GetLatencyReduction returns the latency reduction for cache hits.
func (cs *CacheSimulator) GetLatencyReduction() float64 {
	return cs.speedup.Load().(float64)
}

// generateCacheKey generates a cache key from request.
func (cs *CacheSimulator) generateCacheKey(req models.ChatCompletionRequest) string {
	h := sha256.New()
	h.Write([]byte(req.Model))
	for _, msg := range req.Messages {
		h.Write([]byte(msg.Role))
		h.Write([]byte(msg.Content))
	}
	if req.Temperature != nil {
		h.Write([]byte("temp"))
	}
	hash := hex.EncodeToString(h.Sum(nil))
	return "cache:response:" + hash[:32]
}

// Enable enables caching.
func (cs *CacheSimulator) Enable() {
	cs.enabled.Store(true)
}

// Disable disables caching.
func (cs *CacheSimulator) Disable() {
	cs.enabled.Store(false)
}

// GetStats returns cache statistics.
func (cs *CacheSimulator) GetStats() CacheStats {
	total := cs.totalQueries.Load()
	hits := cs.cacheHits.Load()
	misses := cs.cacheMisses.Load()

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return CacheStats{
		TotalQueries: total,
		CacheHits:    hits,
		CacheMisses:  misses,
		HitRate:      hitRate,
		Enabled:      cs.enabled.Load(),
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalQueries int64
	CacheHits    int64
	CacheMisses  int64
	HitRate      float64
	Enabled      bool
}