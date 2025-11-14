// Package ratelimit provides rate limiting.
// This file implements HTTP headers for rate limit information matching OpenAI's API.
package ratelimit

import (
	"fmt"
	"net/http"
	"time"
)

// AddRateLimitHeaders adds OpenAI-compatible rate limit headers to HTTP response.
// These headers match OpenAI's actual header format exactly.
func AddRateLimitHeaders(w http.ResponseWriter, result *LimitCheckResult, info *LimitInfo) {
	// Request-based headers
	w.Header().Set("x-ratelimit-limit-requests", fmt.Sprintf("%d", info.RequestLimit))
	w.Header().Set("x-ratelimit-remaining-requests", fmt.Sprintf("%d", result.RequestsRemaining))
	w.Header().Set("x-ratelimit-reset-requests", formatResetTime(result.RequestsResetIn))

	// Token-based headers
	w.Header().Set("x-ratelimit-limit-tokens", fmt.Sprintf("%d", info.TokenLimit))
	w.Header().Set("x-ratelimit-remaining-tokens", fmt.Sprintf("%d", result.TokensRemaining))
	w.Header().Set("x-ratelimit-reset-tokens", formatResetTime(result.TokensResetIn))

	// Additional helpful headers (Sentra-specific)
	w.Header().Set("X-Sentra-Rate-Limit-Tier", result.Tier)
	w.Header().Set("X-Sentra-Rate-Limit-Model", result.ModelID)
}

// AddRateLimitExceededHeaders adds headers for 429 (rate limit exceeded) responses.
func AddRateLimitExceededHeaders(w http.ResponseWriter, result *LimitCheckResult, info *LimitInfo) {
	// Standard rate limit headers
	AddRateLimitHeaders(w, result, info)

	// Retry-After header (seconds until reset)
	retryAfter := result.RequestsResetIn
	if result.LimitingFactor == TokenLimit {
		retryAfter = result.TokensResetIn
	}

	// Retry-After header in seconds
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

	// Additional context
	w.Header().Set("X-Sentra-Limiting-Factor", string(result.LimitingFactor))
}

// formatResetTime formats the reset time as a timestamp string.
// OpenAI uses format: "1.23s" (seconds with decimal)
func formatResetTime(duration time.Duration) string {
	seconds := duration.Seconds()
	return fmt.Sprintf("%.2fs", seconds)
}

// ParseRateLimitHeaders parses rate limit information from response headers.
// This is useful for clients reading rate limit info.
func ParseRateLimitHeaders(headers http.Header) (*HeaderRateLimitInfo, error) {
	info := &HeaderRateLimitInfo{}

	// Parse request limits
	fmt.Sscanf(headers.Get("x-ratelimit-limit-requests"), "%d", &info.RequestLimit)
	fmt.Sscanf(headers.Get("x-ratelimit-remaining-requests"), "%d", &info.RequestsRemaining)

	// Parse token limits
	fmt.Sscanf(headers.Get("x-ratelimit-limit-tokens"), "%d", &info.TokenLimit)
	fmt.Sscanf(headers.Get("x-ratelimit-remaining-tokens"), "%d", &info.TokensRemaining)

	// Parse reset times
	info.RequestsResetIn = parseResetTime(headers.Get("x-ratelimit-reset-requests"))
	info.TokensResetIn = parseResetTime(headers.Get("x-ratelimit-reset-tokens"))

	// Parse additional info
	info.Tier = headers.Get("X-Sentra-Rate-Limit-Tier")
	info.Model = headers.Get("X-Sentra-Rate-Limit-Model")
	info.LimitingFactor = LimitType(headers.Get("X-Sentra-Limiting-Factor"))

	return info, nil
}

// parseResetTime parses a reset time string to duration.
func parseResetTime(resetStr string) time.Duration {
	var seconds float64
	fmt.Sscanf(resetStr, "%fs", &seconds)
	return time.Duration(seconds * float64(time.Second))
}

// HeaderRateLimitInfo contains rate limit information parsed from headers.
type HeaderRateLimitInfo struct {
	RequestLimit      int
	RequestsRemaining int
	RequestsResetIn   time.Duration
	TokenLimit        int
	TokensRemaining   int
	TokensResetIn     time.Duration
	Tier              string
	Model             string
	LimitingFactor    LimitType
}

// IsLimited returns true if the request would be rate limited.
func (h *HeaderRateLimitInfo) IsLimited() bool {
	return h.RequestsRemaining == 0 || h.TokensRemaining == 0
}

// GetRetryAfter returns the retry-after duration.
func (h *HeaderRateLimitInfo) GetRetryAfter() time.Duration {
	if h.RequestsRemaining == 0 {
		return h.RequestsResetIn
	}
	if h.TokensRemaining == 0 {
		return h.TokensResetIn
	}
	return 0
}

// String returns a human-readable summary of rate limit info.
func (h *HeaderRateLimitInfo) String() string {
	return fmt.Sprintf(
		"Requests: %d/%d (reset in %v), Tokens: %d/%d (reset in %v)",
		h.RequestsRemaining,
		h.RequestLimit,
		h.RequestsResetIn,
		h.TokensRemaining,
		h.TokenLimit,
		h.TokensResetIn,
	)
}