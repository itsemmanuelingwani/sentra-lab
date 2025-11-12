// Package store provides storage interfaces and implementations for caching and state management.
// This package supports both in-memory and Redis-backed storage for rate limiters, token caches, etc.
package store

import (
	"context"
	"time"
)

// Storage is the main interface for key-value storage.
// Implementations include in-memory (for local development) and Redis (for distributed/production).
type Storage interface {
	// Get retrieves a value by key. Returns nil if key doesn't exist or is expired.
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value with an optional TTL. If TTL is 0, the value never expires.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key from storage.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in storage.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment atomically increments a numeric value and returns the new value.
	// If the key doesn't exist, it's initialized to 0 before incrementing.
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Decrement atomically decrements a numeric value and returns the new value.
	// If the key doesn't exist, it's initialized to 0 before decrementing.
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	// SetNX sets a value only if the key doesn't exist (Set if Not eXists).
	// Returns true if the value was set, false if the key already existed.
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)

	// GetMulti retrieves multiple values by keys. Returns a map of key -> value.
	// Missing keys are not included in the result map.
	GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error)

	// SetMulti stores multiple key-value pairs with the same TTL.
	SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// DeleteMulti removes multiple keys from storage.
	DeleteMulti(ctx context.Context, keys []string) error

	// Keys returns all keys matching the given pattern.
	// Pattern syntax depends on implementation (glob for memory, Redis pattern for Redis).
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Expire sets a new TTL for an existing key.
	// Returns error if key doesn't exist.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// TTL returns the remaining time to live for a key.
	// Returns -1 if key has no expiration, -2 if key doesn't exist.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Flush removes all keys from storage. Use with caution!
	Flush(ctx context.Context) error

	// Close releases any resources held by the storage implementation.
	Close() error

	// Ping checks if the storage backend is healthy.
	Ping(ctx context.Context) error
}

// CacheStorage is a specialized interface for caching with additional features.
type CacheStorage interface {
	Storage

	// GetWithTTL retrieves a value and its remaining TTL.
	GetWithTTL(ctx context.Context, key string) (value interface{}, ttl time.Duration, err error)

	// Refresh updates the TTL of a key without changing its value.
	Refresh(ctx context.Context, key string, ttl time.Duration) error

	// Size returns the number of keys in the cache.
	Size(ctx context.Context) (int64, error)

	// MemoryUsage returns approximate memory usage in bytes (if supported).
	MemoryUsage(ctx context.Context) (int64, error)
}

// RateLimitStorage is a specialized interface for rate limiting operations.
type RateLimitStorage interface {
	Storage

	// GetTokens retrieves the current token count for a rate limiter.
	GetTokens(ctx context.Context, key string) (float64, error)

	// SetTokens sets the token count for a rate limiter.
	SetTokens(ctx context.Context, key string, tokens float64, ttl time.Duration) error

	// DecrementTokens atomically decrements tokens and returns the new value.
	// Returns error if resulting value would be negative.
	DecrementTokens(ctx context.Context, key string, tokens float64) (float64, error)

	// IncrementTokens atomically increments tokens (for refill operations).
	IncrementTokens(ctx context.Context, key string, tokens float64) (float64, error)

	// GetLastRefill retrieves the last refill timestamp for a rate limiter.
	GetLastRefill(ctx context.Context, key string) (time.Time, error)

	// SetLastRefill updates the last refill timestamp.
	SetLastRefill(ctx context.Context, key string, timestamp time.Time) error
}

// StorageStats contains statistics about storage usage.
type StorageStats struct {
	// TotalKeys is the total number of keys in storage
	TotalKeys int64

	// HitCount is the number of cache hits (if supported)
	HitCount int64

	// MissCount is the number of cache misses (if supported)
	MissCount int64

	// HitRate is the cache hit rate as a percentage (0-100)
	HitRate float64

	// MemoryUsage is the approximate memory usage in bytes
	MemoryUsage int64

	// Uptime is the duration since storage was initialized
	Uptime time.Duration
}

// GetStats returns statistics about the storage implementation.
type StatsProvider interface {
	GetStats(ctx context.Context) (StorageStats, error)
}

// StorageError represents a storage-specific error.
type StorageError struct {
	// Op is the operation that failed (e.g., "Get", "Set")
	Op string

	// Key is the key involved in the operation
	Key string

	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Key != "" {
		return "storage." + e.Op + "(" + e.Key + "): " + e.Err.Error()
	}
	return "storage." + e.Op + ": " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *StorageError) Unwrap() error {
	return e.Err
}

// NewStorageError creates a new storage error.
func NewStorageError(op string, key string, err error) *StorageError {
	return &StorageError{
		Op:  op,
		Key: key,
		Err: err,
	}
}

// Common storage errors
var (
	// ErrKeyNotFound is returned when a key doesn't exist
	ErrKeyNotFound = &StorageError{Op: "Get", Err: nil}

	// ErrKeyExists is returned when SetNX fails because key already exists
	ErrKeyExists = &StorageError{Op: "SetNX", Err: nil}

	// ErrInvalidValue is returned when a value has the wrong type
	ErrInvalidValue = &StorageError{Op: "Get", Err: nil}

	// ErrConnectionFailed is returned when storage backend is unreachable
	ErrConnectionFailed = &StorageError{Op: "Connect", Err: nil}

	// ErrOperationTimeout is returned when an operation times out
	ErrOperationTimeout = &StorageError{Op: "Timeout", Err: nil}
)

// IsNotFound returns true if the error is a "key not found" error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(*StorageError); ok {
		return se.Op == "Get" && se.Err == nil
	}
	return false
}

// IsTimeout returns true if the error is a timeout error.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(*StorageError); ok {
		return se.Op == "Timeout"
	}
	return false
}

// SerializableValue is an interface for values that can be serialized/deserialized.
type SerializableValue interface {
	// MarshalBinary converts the value to binary format
	MarshalBinary() ([]byte, error)

	// UnmarshalBinary restores the value from binary format
	UnmarshalBinary(data []byte) error
}