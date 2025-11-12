// Package store provides storage implementations.
// This file implements Redis-backed storage for distributed scenarios.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Storage using Redis.
// This is suitable for distributed deployments and production use.
type RedisStore struct {
	client    *redis.Client
	startTime time.Time
}

// RedisConfig contains configuration for Redis connection.
type RedisConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6379")
	Addr string

	// Password for authentication (optional)
	Password string

	// DB is the Redis database number (0-15)
	DB int

	// PoolSize is the maximum number of connections
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int

	// DialTimeout is the timeout for establishing connections
	DialTimeout time.Duration

	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for write operations
	WriteTimeout time.Duration

	// PoolTimeout is the timeout for getting a connection from the pool
	PoolTimeout time.Duration
}

// DefaultRedisConfig returns default Redis configuration.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// NewRedisStore creates a new Redis storage instance.
func NewRedisStore(config RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.PoolTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, NewStorageError("Connect", "", fmt.Errorf("failed to connect to Redis: %w", err))
	}

	return &RedisStore{
		client:    client,
		startTime: time.Now(),
	}, nil
}

// NewRedisStoreFromURL creates a Redis store from a connection URL.
func NewRedisStoreFromURL(url string) (*RedisStore, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, NewStorageError("Connect", "", fmt.Errorf("invalid Redis URL: %w", err))
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, NewStorageError("Connect", "", fmt.Errorf("failed to connect to Redis: %w", err))
	}

	return &RedisStore{
		client:    client,
		startTime: time.Now(),
	}, nil
}

// Get retrieves a value by key.
func (r *RedisStore) Get(ctx context.Context, key string) (interface{}, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, NewStorageError("Get", key, fmt.Errorf("key not found"))
		}
		return nil, NewStorageError("Get", key, err)
	}

	// Deserialize JSON
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, NewStorageError("Get", key, fmt.Errorf("failed to deserialize value: %w", err))
	}

	return value, nil
}

// Set stores a value with an optional TTL.
func (r *RedisStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Serialize to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return NewStorageError("Set", key, fmt.Errorf("failed to serialize value: %w", err))
	}

	if ttl > 0 {
		err = r.client.Set(ctx, key, data, ttl).Err()
	} else {
		err = r.client.Set(ctx, key, data, 0).Err()
	}

	if err != nil {
		return NewStorageError("Set", key, err)
	}

	return nil
}

// Delete removes a key from storage.
func (r *RedisStore) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return NewStorageError("Delete", key, err)
	}
	return nil
}

// Exists checks if a key exists.
func (r *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, NewStorageError("Exists", key, err)
	}
	return count > 0, nil
}

// Increment atomically increments a numeric value.
func (r *RedisStore) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, NewStorageError("Increment", key, err)
	}
	return result, nil
}

// Decrement atomically decrements a numeric value.
func (r *RedisStore) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	result, err := r.client.DecrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, NewStorageError("Decrement", key, err)
	}
	return result, nil
}

// SetNX sets a value only if the key doesn't exist.
func (r *RedisStore) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	// Serialize to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return false, NewStorageError("SetNX", key, fmt.Errorf("failed to serialize value: %w", err))
	}

	result, err := r.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, NewStorageError("SetNX", key, err)
	}

	return result, nil
}

// GetMulti retrieves multiple values by keys.
func (r *RedisStore) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, NewStorageError("GetMulti", "", err)
	}

	values := make(map[string]interface{})
	for i, result := range results {
		if result == nil {
			continue // Key doesn't exist
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		var value interface{}
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			continue // Skip invalid values
		}

		values[keys[i]] = value
	}

	return values, nil
}

// SetMulti stores multiple key-value pairs.
func (r *RedisStore) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return NewStorageError("SetMulti", key, fmt.Errorf("failed to serialize value: %w", err))
		}

		if ttl > 0 {
			pipe.Set(ctx, key, data, ttl)
		} else {
			pipe.Set(ctx, key, data, 0)
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return NewStorageError("SetMulti", "", err)
	}

	return nil
}

// DeleteMulti removes multiple keys.
func (r *RedisStore) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return NewStorageError("DeleteMulti", "", err)
	}

	return nil
}

// Keys returns all keys matching the given pattern.
func (r *RedisStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, NewStorageError("Keys", "", err)
	}
	return keys, nil
}

// Expire sets a new TTL for an existing key.
func (r *RedisStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	result, err := r.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return NewStorageError("Expire", key, err)
	}

	if !result {
		return NewStorageError("Expire", key, fmt.Errorf("key not found"))
	}

	return nil
}

// TTL returns the remaining time to live for a key.
func (r *RedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, NewStorageError("TTL", key, err)
	}
	return ttl, nil
}

// Flush removes all keys.
func (r *RedisStore) Flush(ctx context.Context) error {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return NewStorageError("Flush", "", err)
	}
	return nil
}

// Close releases resources.
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Ping checks if storage is healthy.
func (r *RedisStore) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return NewStorageError("Ping", "", err)
	}
	return nil
}

// GetStats returns storage statistics.
func (r *RedisStore) GetStats(ctx context.Context) (StorageStats, error) {
	info, err := r.client.Info(ctx, "stats", "keyspace").Result()
	if err != nil {
		return StorageStats{}, NewStorageError("GetStats", "", err)
	}

	// Parse Redis INFO output (simplified)
	// In production, you'd parse this more thoroughly
	stats := StorageStats{
		TotalKeys:   0, // Would parse from keyspace info
		HitCount:    0, // Would parse from stats
		MissCount:   0,
		HitRate:     0,
		MemoryUsage: 0, // Would get from INFO memory
		Uptime:      time.Since(r.startTime),
	}

	// Get total keys
	dbSize, err := r.client.DBSize(ctx).Result()
	if err == nil {
		stats.TotalKeys = dbSize
	}

	return stats, nil
}

// GetTokens retrieves the current token count (for rate limiting).
func (r *RedisStore) GetTokens(ctx context.Context, key string) (float64, error) {
	result, err := r.client.Get(ctx, key).Float64()
	if err != nil {
		if err == redis.Nil {
			return 0, NewStorageError("GetTokens", key, fmt.Errorf("key not found"))
		}
		return 0, NewStorageError("GetTokens", key, err)
	}
	return result, nil
}

// SetTokens sets the token count.
func (r *RedisStore) SetTokens(ctx context.Context, key string, tokens float64, ttl time.Duration) error {
	if ttl > 0 {
		err := r.client.Set(ctx, key, tokens, ttl).Err()
		if err != nil {
			return NewStorageError("SetTokens", key, err)
		}
	} else {
		err := r.client.Set(ctx, key, tokens, 0).Err()
		if err != nil {
			return NewStorageError("SetTokens", key, err)
		}
	}
	return nil
}

// DecrementTokens atomically decrements tokens.
func (r *RedisStore) DecrementTokens(ctx context.Context, key string, tokens float64) (float64, error) {
	result, err := r.client.IncrByFloat(ctx, key, -tokens).Result()
	if err != nil {
		return 0, NewStorageError("DecrementTokens", key, err)
	}

	if result < 0 {
		// Rollback
		r.client.IncrByFloat(ctx, key, tokens)
		return 0, NewStorageError("DecrementTokens", key, fmt.Errorf("insufficient tokens"))
	}

	return result, nil
}

// IncrementTokens atomically increments tokens.
func (r *RedisStore) IncrementTokens(ctx context.Context, key string, tokens float64) (float64, error) {
	result, err := r.client.IncrByFloat(ctx, key, tokens).Result()
	if err != nil {
		return 0, NewStorageError("IncrementTokens", key, err)
	}
	return result, nil
}

// GetLastRefill retrieves the last refill timestamp.
func (r *RedisStore) GetLastRefill(ctx context.Context, key string) (time.Time, error) {
	value, err := r.Get(ctx, key)
	if err != nil {
		return time.Time{}, err
	}

	// Value should be a Unix timestamp (int64)
	timestamp, ok := value.(float64)
	if !ok {
		return time.Time{}, NewStorageError("GetLastRefill", key, fmt.Errorf("value is not a timestamp"))
	}

	return time.Unix(int64(timestamp), 0), nil
}

// SetLastRefill updates the last refill timestamp.
func (r *RedisStore) SetLastRefill(ctx context.Context, key string, timestamp time.Time) error {
	return r.Set(ctx, key, timestamp.Unix(), 0)
}

// Compile-time interface checks
var _ Storage = (*RedisStore)(nil)
var _ RateLimitStorage = (*RedisStore)(nil)
var _ StatsProvider = (*RedisStore)(nil)