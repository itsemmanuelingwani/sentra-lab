// Package tokenizer provides token counting using tiktoken for production-accurate token counts.
// This file defines the public interface for token counting.
package tokenizer

import (
	"context"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Counter is the public interface for token counting.
// Implementations must provide accurate token counts matching OpenAI's tiktoken library.
type Counter interface {
	// Count counts tokens in a list of messages for the given model.
	// This includes special tokens for role markers and formatting.
	Count(ctx context.Context, messages []models.Message, model string) (int, error)

	// CountText counts tokens in plain text for the given model.
	// This is used for embeddings and completions.
	CountText(ctx context.Context, text string, model string) (int, error)

	// CountWithMetadata counts tokens and returns additional metadata.
	CountWithMetadata(ctx context.Context, messages []models.Message, model string) (TokenCount, error)

	// EstimateOutputTokens estimates the number of output tokens based on max_tokens.
	// Returns the minimum of max_tokens and the model's maximum output token limit.
	EstimateOutputTokens(ctx context.Context, maxTokens int, model string) (int, error)

	// ValidateContextLength checks if the input + output tokens fit within the model's context window.
	ValidateContextLength(ctx context.Context, inputTokens, outputTokens int, model string) error
}

// TokenCount represents detailed token count information.
type TokenCount struct {
	// InputTokens is the number of tokens in the input messages
	InputTokens int

	// EstimatedOutputTokens is the estimated number of output tokens
	EstimatedOutputTokens int

	// TotalTokens is the sum of input and estimated output tokens
	TotalTokens int

	// Model is the model used for counting
	Model string

	// Encoding is the tokenizer encoding used (e.g., "cl100k_base")
	Encoding string

	// ContextWindow is the maximum context length for the model
	ContextWindow int

	// RemainingTokens is the number of tokens remaining in the context window
	RemainingTokens int
}

// CounterStats contains statistics about tokenization operations.
type CounterStats struct {
	// TotalCounts is the total number of token counting operations
	TotalCounts int64

	// CacheHits is the number of cache hits
	CacheHits int64

	// CacheMisses is the number of cache misses
	CacheMisses int64

	// CacheHitRate is the cache hit rate as a percentage (0-100)
	CacheHitRate float64

	// AverageTokensPerMessage is the average number of tokens per message
	AverageTokensPerMessage float64

	// TotalTokensCounted is the cumulative number of tokens counted
	TotalTokensCounted int64
}

// StatsProvider provides statistics about the tokenizer.
type StatsProvider interface {
	// GetStats returns tokenizer statistics
	GetStats(ctx context.Context) (CounterStats, error)

	// ResetStats resets all statistics
	ResetStats(ctx context.Context) error
}