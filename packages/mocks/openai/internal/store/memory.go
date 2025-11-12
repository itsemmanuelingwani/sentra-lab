// Package store provides storage implementations.
// This file implements in-memory storage using a thread-safe map with TTL support.
package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// item represents a stored value with expiration.
type item struct {
	value  interface{}
	expiry time.Time
}

// isExpired checks if the item has expired.
func (i *item) isExpired() bool {
	if i.expiry.IsZero() {
		return false // No expiration
	}
	return time.Now().After(i.expiry)
}

// MemoryStore implements Storage using an in-memory map.
// This is suitable for local development and testing.
// All operations are thread-safe using a RWMutex.
type MemoryStore struct {
	data      map[string]*item
	mu        sync.RWMutex
	startTime time.Time
	hitCount  int64
	missCount int64
	closed    bool
}

// NewMemoryStore creates a new in-memory storage instance.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data:      make(map[string]*item),
		startTime: time.Now(),
	}

	// Start background cleanup goroutine
	go store.cleanupExpired()

	return store
}

// Get retrieves a value by key.
func (m *MemoryStore) Get(ctx context.Context, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, NewStorageError("Get", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok {
		m.missCount++
		return nil, NewStorageError("Get", key, fmt.Errorf("key not found"))
	}

	if item.isExpired() {
		m.missCount++
		return nil, NewStorageError("Get", key, fmt.Errorf("key expired"))
	}

	m.hitCount++
	return item.value, nil
}

// Set stores a value with an optional TTL.
func (m *MemoryStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("Set", key, fmt.Errorf("storage is closed"))
	}

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	m.data[key] = &item{
		value:  value,
		expiry: expiry,
	}

	return nil
}

// Delete removes a key from storage.
func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("Delete", key, fmt.Errorf("storage is closed"))
	}

	delete(m.data, key)
	return nil
}

// Exists checks if a key exists.
func (m *MemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, NewStorageError("Exists", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok {
		return false, nil
	}

	if item.isExpired() {
		return false, nil
	}

	return true, nil
}

// Increment atomically increments a numeric value.
func (m *MemoryStore) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, NewStorageError("Increment", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok || item.isExpired() {
		// Initialize to 0 if key doesn't exist
		m.data[key] = &item{value: delta, expiry: time.Time{}}
		return delta, nil
	}

	// Convert current value to int64
	current, ok := item.value.(int64)
	if !ok {
		return 0, NewStorageError("Increment", key, fmt.Errorf("value is not int64"))
	}

	newValue := current + delta
	item.value = newValue

	return newValue, nil
}

// Decrement atomically decrements a numeric value.
func (m *MemoryStore) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return m.Increment(ctx, key, -delta)
}

// SetNX sets a value only if the key doesn't exist.
func (m *MemoryStore) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return false, NewStorageError("SetNX", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if ok && !item.isExpired() {
		return false, nil // Key already exists
	}

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	m.data[key] = &item{
		value:  value,
		expiry: expiry,
	}

	return true, nil
}

// GetMulti retrieves multiple values by keys.
func (m *MemoryStore) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, NewStorageError("GetMulti", "", fmt.Errorf("storage is closed"))
	}

	result := make(map[string]interface{})
	for _, key := range keys {
		item, ok := m.data[key]
		if ok && !item.isExpired() {
			result[key] = item.value
			m.hitCount++
		} else {
			m.missCount++
		}
	}

	return result, nil
}

// SetMulti stores multiple key-value pairs.
func (m *MemoryStore) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("SetMulti", "", fmt.Errorf("storage is closed"))
	}

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	for key, value := range items {
		m.data[key] = &item{
			value:  value,
			expiry: expiry,
		}
	}

	return nil
}

// DeleteMulti removes multiple keys.
func (m *MemoryStore) DeleteMulti(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("DeleteMulti", "", fmt.Errorf("storage is closed"))
	}

	for _, key := range keys {
		delete(m.data, key)
	}

	return nil
}

// Keys returns all keys matching the given pattern.
func (m *MemoryStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, NewStorageError("Keys", "", fmt.Errorf("storage is closed"))
	}

	var keys []string
	for key, item := range m.data {
		if !item.isExpired() {
			matched, err := filepath.Match(pattern, key)
			if err != nil {
				return nil, NewStorageError("Keys", "", err)
			}
			if matched {
				keys = append(keys, key)
			}
		}
	}

	return keys, nil
}

// Expire sets a new TTL for an existing key.
func (m *MemoryStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("Expire", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok || item.isExpired() {
		return NewStorageError("Expire", key, fmt.Errorf("key not found"))
	}

	if ttl > 0 {
		item.expiry = time.Now().Add(ttl)
	} else {
		item.expiry = time.Time{} // No expiration
	}

	return nil
}

// TTL returns the remaining time to live for a key.
func (m *MemoryStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, NewStorageError("TTL", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok {
		return -2 * time.Second, nil // Key doesn't exist
	}

	if item.expiry.IsZero() {
		return -1 * time.Second, nil // No expiration
	}

	if item.isExpired() {
		return -2 * time.Second, nil // Already expired
	}

	return time.Until(item.expiry), nil
}

// Flush removes all keys.
func (m *MemoryStore) Flush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return NewStorageError("Flush", "", fmt.Errorf("storage is closed"))
	}

	m.data = make(map[string]*item)
	m.hitCount = 0
	m.missCount = 0

	return nil
}

// Close releases resources.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.data = nil

	return nil
}

// Ping checks if storage is healthy.
func (m *MemoryStore) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("storage is closed")
	}

	return nil
}

// GetStats returns storage statistics.
func (m *MemoryStore) GetStats(ctx context.Context) (StorageStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalKeys := int64(len(m.data))
	hitRate := float64(0)
	if m.hitCount+m.missCount > 0 {
		hitRate = float64(m.hitCount) / float64(m.hitCount+m.missCount) * 100
	}

	return StorageStats{
		TotalKeys:   totalKeys,
		HitCount:    m.hitCount,
		MissCount:   m.missCount,
		HitRate:     hitRate,
		MemoryUsage: 0, // Not implemented for memory store
		Uptime:      time.Since(m.startTime),
	}, nil
}

// cleanupExpired periodically removes expired items.
func (m *MemoryStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return
		}

		now := time.Now()
		for key, item := range m.data {
			if !item.expiry.IsZero() && now.After(item.expiry) {
				delete(m.data, key)
			}
		}

		m.mu.Unlock()
	}
}

// GetTokens retrieves the current token count (for rate limiting).
func (m *MemoryStore) GetTokens(ctx context.Context, key string) (float64, error) {
	value, err := m.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	tokens, ok := value.(float64)
	if !ok {
		return 0, NewStorageError("GetTokens", key, fmt.Errorf("value is not float64"))
	}

	return tokens, nil
}

// SetTokens sets the token count.
func (m *MemoryStore) SetTokens(ctx context.Context, key string, tokens float64, ttl time.Duration) error {
	return m.Set(ctx, key, tokens, ttl)
}

// DecrementTokens atomically decrements tokens.
func (m *MemoryStore) DecrementTokens(ctx context.Context, key string, tokens float64) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, NewStorageError("DecrementTokens", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok || item.isExpired() {
		return 0, NewStorageError("DecrementTokens", key, fmt.Errorf("key not found"))
	}

	current, ok := item.value.(float64)
	if !ok {
		return 0, NewStorageError("DecrementTokens", key, fmt.Errorf("value is not float64"))
	}

	newValue := current - tokens
	if newValue < 0 {
		return 0, NewStorageError("DecrementTokens", key, fmt.Errorf("insufficient tokens"))
	}

	item.value = newValue
	return newValue, nil
}

// IncrementTokens atomically increments tokens.
func (m *MemoryStore) IncrementTokens(ctx context.Context, key string, tokens float64) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, NewStorageError("IncrementTokens", key, fmt.Errorf("storage is closed"))
	}

	item, ok := m.data[key]
	if !ok || item.isExpired() {
		// Initialize if not exists
		m.data[key] = &item{value: tokens, expiry: time.Time{}}
		return tokens, nil
	}

	current, ok := item.value.(float64)
	if !ok {
		return 0, NewStorageError("IncrementTokens", key, fmt.Errorf("value is not float64"))
	}

	newValue := current + tokens
	item.value = newValue

	return newValue, nil
}

// GetLastRefill retrieves the last refill timestamp.
func (m *MemoryStore) GetLastRefill(ctx context.Context, key string) (time.Time, error) {
	value, err := m.Get(ctx, key)
	if err != nil {
		return time.Time{}, err
	}

	timestamp, ok := value.(time.Time)
	if !ok {
		return time.Time{}, NewStorageError("GetLastRefill", key, fmt.Errorf("value is not time.Time"))
	}

	return timestamp, nil
}

// SetLastRefill updates the last refill timestamp.
func (m *MemoryStore) SetLastRefill(ctx context.Context, key string, timestamp time.Time) error {
	return m.Set(ctx, key, timestamp, 0) // No expiration for timestamps
}

// Compile-time interface checks
var _ Storage = (*MemoryStore)(nil)
var _ RateLimitStorage = (*MemoryStore)(nil)
var _ StatsProvider = (*MemoryStore)(nil)