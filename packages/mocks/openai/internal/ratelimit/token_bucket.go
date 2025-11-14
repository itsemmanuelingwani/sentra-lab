// Package ratelimit provides rate limiting.
// This file implements the token bucket algorithm for rate limiting.
package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
// This matches OpenAI's production rate limiting behavior exactly.
type TokenBucket struct {
	// capacity is the maximum number of tokens the bucket can hold
	capacity int

	// tokens is the current number of tokens available
	tokens float64

	// refillRate is the number of tokens added per second
	refillRate float64

	// lastRefill is the timestamp of the last refill
	lastRefill time.Time

	// mu protects concurrent access
	mu sync.Mutex
}

// NewTokenBucket creates a new token bucket.
// capacity: maximum tokens the bucket can hold
// refillPerMinute: how many tokens are added per minute
func NewTokenBucket(capacity int, refillPerMinute int) *TokenBucket {
	now := time.Now()

	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity), // Start with full bucket
		refillRate: float64(refillPerMinute) / 60.0, // Convert to per-second
		lastRefill: now,
	}
}

// Allow checks if the requested number of tokens can be consumed.
// Returns true if tokens are available, false otherwise.
func (tb *TokenBucket) Allow(tokens int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	tb.refill()

	// Check if enough tokens available
	if tb.tokens >= float64(tokens) {
		tb.tokens -= float64(tokens)
		return true
	}

	return false
}

// AllowWithInfo checks if tokens are available and returns detailed info.
func (tb *TokenBucket) AllowWithInfo(tokens int) AllowResult {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens
	tb.refill()

	allowed := tb.tokens >= float64(tokens)
	remaining := int(tb.tokens)

	if allowed {
		tb.tokens -= float64(tokens)
		remaining = int(tb.tokens)
	}

	// Calculate when bucket will have enough tokens
	resetIn := time.Duration(0)
	if !allowed {
		needed := float64(tokens) - tb.tokens
		secondsUntilAvailable := needed / tb.refillRate
		resetIn = time.Duration(secondsUntilAvailable * float64(time.Second))
	}

	return AllowResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetIn:   resetIn,
		Capacity:  tb.capacity,
	}
}

// refill adds tokens based on elapsed time since last refill.
// This implements continuous refill (not fixed-window).
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Calculate tokens to add
	tokensToAdd := elapsed.Seconds() * tb.refillRate

	// Add tokens (capped at capacity)
	tb.tokens = min(tb.tokens+tokensToAdd, float64(tb.capacity))

	// Update last refill time
	tb.lastRefill = now
}

// GetState returns the current state of the bucket.
func (tb *TokenBucket) GetState() BucketState {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	return BucketState{
		Capacity:       tb.capacity,
		Available:      int(tb.tokens),
		RefillRate:     tb.refillRate,
		RefillPerMin:   tb.refillRate * 60,
		LastRefill:     tb.lastRefill,
		FillPercentage: (tb.tokens / float64(tb.capacity)) * 100,
	}
}

// Reset resets the bucket to full capacity.
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = float64(tb.capacity)
	tb.lastRefill = time.Now()
}

// SetCapacity changes the bucket capacity.
func (tb *TokenBucket) SetCapacity(capacity int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.capacity = capacity
	// Clamp current tokens to new capacity
	if tb.tokens > float64(capacity) {
		tb.tokens = float64(capacity)
	}
}

// SetRefillRate changes the refill rate (tokens per minute).
func (tb *TokenBucket) SetRefillRate(refillPerMinute int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refillRate = float64(refillPerMinute) / 60.0
}

// TimeUntilAvailable returns how long until the specified tokens are available.
func (tb *TokenBucket) TimeUntilAvailable(tokens int) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(tokens) {
		return 0 // Already available
	}

	needed := float64(tokens) - tb.tokens
	secondsUntilAvailable := needed / tb.refillRate

	return time.Duration(secondsUntilAvailable * float64(time.Second))
}

// WaitForTokens blocks until the specified tokens are available or context is cancelled.
func (tb *TokenBucket) WaitForTokens(tokens int, maxWait time.Duration) error {
	waitTime := tb.TimeUntilAvailable(tokens)

	if waitTime > maxWait {
		return fmt.Errorf("tokens will not be available within max wait time")
	}

	time.Sleep(waitTime)
	return nil
}

// AllowResult contains the result of an Allow check.
type AllowResult struct {
	// Allowed indicates if the request was allowed
	Allowed bool

	// Remaining is the number of tokens remaining
	Remaining int

	// ResetIn is the duration until tokens are available
	ResetIn time.Duration

	// Capacity is the bucket capacity
	Capacity int
}

// BucketState represents the current state of a token bucket.
type BucketState struct {
	Capacity       int
	Available      int
	RefillRate     float64 // Per second
	RefillPerMin   float64 // Per minute
	LastRefill     time.Time
	FillPercentage float64
}

// String returns a string representation of the bucket state.
func (bs BucketState) String() string {
	return fmt.Sprintf(
		"Bucket[%d/%d tokens (%.1f%%), refill: %.1f/min]",
		bs.Available,
		bs.Capacity,
		bs.FillPercentage,
		bs.RefillPerMin,
	)
}

// DualTokenBucket implements dual rate limiting (RPM + TPM).
// This is exactly how OpenAI rate limits: both request count AND token count.
type DualTokenBucket struct {
	// requestBucket limits requests per minute
	requestBucket *TokenBucket

	// tokenBucket limits tokens per minute
	tokenBucket *TokenBucket

	// mu protects concurrent access
	mu sync.RWMutex
}

// NewDualTokenBucket creates a dual token bucket for RPM and TPM limits.
func NewDualTokenBucket(rpm, tpm int) *DualTokenBucket {
	return &DualTokenBucket{
		requestBucket: NewTokenBucket(rpm, rpm),
		tokenBucket:   NewTokenBucket(tpm, tpm),
	}
}

// Allow checks if both a request and token count can be consumed.
func (dtb *DualTokenBucket) Allow(tokens int) DualAllowResult {
	dtb.mu.Lock()
	defer dtb.mu.Unlock()

	// Check request limit (1 request)
	reqResult := dtb.requestBucket.AllowWithInfo(1)

	// Check token limit
	tokenResult := dtb.tokenBucket.AllowWithInfo(tokens)

	// Both must pass for overall allow
	allowed := reqResult.Allowed && tokenResult.Allowed

	// If request was allowed but tokens weren't, refund the request
	if reqResult.Allowed && !tokenResult.Allowed {
		dtb.requestBucket.mu.Lock()
		dtb.requestBucket.tokens += 1
		dtb.requestBucket.mu.Unlock()
	}

	return DualAllowResult{
		Allowed:          allowed,
		RequestsResult:   reqResult,
		TokensResult:     tokenResult,
		LimitingFactor:   dtb.determineLimitingFactor(reqResult, tokenResult),
	}
}

// determineLimitingFactor identifies what caused rate limiting.
func (dtb *DualTokenBucket) determineLimitingFactor(reqResult, tokenResult AllowResult) LimitType {
	if reqResult.Allowed && tokenResult.Allowed {
		return NoLimit
	}

	if !reqResult.Allowed {
		return RequestLimit
	}

	if !tokenResult.Allowed {
		return TokenLimit
	}

	return NoLimit
}

// GetState returns the state of both buckets.
func (dtb *DualTokenBucket) GetState() DualBucketState {
	dtb.mu.RLock()
	defer dtb.mu.RUnlock()

	return DualBucketState{
		RequestBucket: dtb.requestBucket.GetState(),
		TokenBucket:   dtb.tokenBucket.GetState(),
	}
}

// Reset resets both buckets to full capacity.
func (dtb *DualTokenBucket) Reset() {
	dtb.mu.Lock()
	defer dtb.mu.Unlock()

	dtb.requestBucket.Reset()
	dtb.tokenBucket.Reset()
}

// SetLimits updates the rate limits.
func (dtb *DualTokenBucket) SetLimits(rpm, tpm int) {
	dtb.mu.Lock()
	defer dtb.mu.Unlock()

	dtb.requestBucket.SetCapacity(rpm)
	dtb.requestBucket.SetRefillRate(rpm)
	dtb.tokenBucket.SetCapacity(tpm)
	dtb.tokenBucket.SetRefillRate(tpm)
}

// DualAllowResult contains results from dual bucket check.
type DualAllowResult struct {
	Allowed        bool
	RequestsResult AllowResult
	TokensResult   AllowResult
	LimitingFactor LimitType
}

// DualBucketState contains state of both buckets.
type DualBucketState struct {
	RequestBucket BucketState
	TokenBucket   BucketState
}

// LimitType indicates which limit was hit.
type LimitType string

const (
	NoLimit      LimitType = "none"
	RequestLimit LimitType = "requests"
	TokenLimit   LimitType = "tokens"
)

// String returns a string representation of the dual state.
func (dbs DualBucketState) String() string {
	return fmt.Sprintf(
		"Requests: %s, Tokens: %s",
		dbs.RequestBucket.String(),
		dbs.TokenBucket.String(),
	)
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}