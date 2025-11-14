// Package tokenizer provides token counting.
// This file implements a fast token estimator for rate limiting without full tokenization.
package tokenizer

import (
	"context"
	"unicode/utf8"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Estimator provides fast token count estimation without full tokenization.
// This is used for rate limiting where approximate counts are acceptable.
// The estimation uses the character/4 heuristic that OpenAI uses for rate limiting.
type Estimator struct {
	// No state needed - all methods are stateless
}

// NewEstimator creates a new token estimator.
func NewEstimator() *Estimator {
	return &Estimator{}
}

// Estimate estimates token count using character-based heuristic.
// Formula: (character_count / 4) + max_tokens
// This matches OpenAI's rate limiting estimation algorithm.
func (e *Estimator) Estimate(ctx context.Context, messages []models.Message, maxTokens int, model string) (int, error) {
	// Get model configuration
	config, err := models.GetModelConfig(model)
	if err != nil {
		return 0, err
	}

	// Count characters in all messages
	charCount := 0
	for _, msg := range messages {
		charCount += utf8.RuneCountInString(msg.Content)
		// Add characters for role and formatting
		charCount += utf8.RuneCountInString(msg.Role) + 10 // Approximate formatting overhead
	}

	// Estimate input tokens (character_count / 4)
	estimatedInput := charCount / 4

	// Estimate output tokens
	estimatedOutput := maxTokens
	if maxTokens <= 0 || maxTokens > config.MaxOutputTokens {
		estimatedOutput = config.MaxOutputTokens
	}

	// Total estimated tokens
	totalEstimated := estimatedInput + estimatedOutput

	return totalEstimated, nil
}

// EstimateText estimates tokens for plain text.
func (e *Estimator) EstimateText(ctx context.Context, text string, model string) (int, error) {
	charCount := utf8.RuneCountInString(text)
	estimatedTokens := charCount / 4

	// Ensure at least 1 token for non-empty text
	if charCount > 0 && estimatedTokens == 0 {
		estimatedTokens = 1
	}

	return estimatedTokens, nil
}

// EstimateInputTokens estimates only input tokens (without output).
func (e *Estimator) EstimateInputTokens(ctx context.Context, messages []models.Message, model string) (int, error) {
	charCount := 0
	for _, msg := range messages {
		charCount += utf8.RuneCountInString(msg.Content)
		charCount += utf8.RuneCountInString(msg.Role) + 10
	}

	estimatedTokens := charCount / 4

	// Ensure at least 1 token for non-empty messages
	if len(messages) > 0 && estimatedTokens == 0 {
		estimatedTokens = 1
	}

	return estimatedTokens, nil
}

// EstimateWithConfidence estimates tokens and provides confidence level.
func (e *Estimator) EstimateWithConfidence(ctx context.Context, messages []models.Message, maxTokens int, model string) (EstimatedCount, error) {
	estimated, err := e.Estimate(ctx, messages, maxTokens, model)
	if err != nil {
		return EstimatedCount{}, err
	}

	// Calculate confidence based on text characteristics
	// Shorter texts have higher variance, so lower confidence
	totalChars := 0
	for _, msg := range messages {
		totalChars += utf8.RuneCountInString(msg.Content)
	}

	// Confidence: 0.7-0.9 based on text length
	// Longer texts are more predictable
	confidence := 0.7
	if totalChars > 1000 {
		confidence = 0.85
	} else if totalChars > 500 {
		confidence = 0.80
	} else if totalChars > 100 {
		confidence = 0.75
	}

	// Calculate error margin (±20% for low confidence, ±10% for high)
	errorMargin := estimated / 5 // 20%
	if confidence > 0.8 {
		errorMargin = estimated / 10 // 10%
	}

	return EstimatedCount{
		Estimated:   estimated,
		LowerBound:  estimated - errorMargin,
		UpperBound:  estimated + errorMargin,
		Confidence:  confidence,
		Method:      "character-based",
	}, nil
}

// EstimatedCount represents an estimated token count with confidence bounds.
type EstimatedCount struct {
	// Estimated is the estimated token count
	Estimated int

	// LowerBound is the lower bound of the estimate
	LowerBound int

	// UpperBound is the upper bound of the estimate
	UpperBound int

	// Confidence is the confidence level (0.0 to 1.0)
	Confidence float64

	// Method is the estimation method used
	Method string
}

// CompareWithActual compares an estimate with the actual count.
// Returns the error percentage: positive if overestimated, negative if underestimated.
func (e *EstimatedCount) CompareWithActual(actual int) float64 {
	if actual == 0 {
		return 0
	}
	return float64(e.Estimated-actual) / float64(actual) * 100
}

// IsWithinBounds checks if the actual count is within the confidence bounds.
func (e *EstimatedCount) IsWithinBounds(actual int) bool {
	return actual >= e.LowerBound && actual <= e.UpperBound
}

// EstimatorStats contains statistics about estimation accuracy.
type EstimatorStats struct {
	// TotalEstimations is the total number of estimations performed
	TotalEstimations int64

	// AverageError is the average error percentage
	AverageError float64

	// WithinBounds is the percentage of estimates within confidence bounds
	WithinBounds float64

	// Overestimates is the number of overestimates
	Overestimates int64

	// Underestimates is the number of underestimates
	Underestimates int64
}

// FastEstimate provides a very fast token estimate for rate limiting.
// This sacrifices accuracy for speed (no model lookup, simple division).
func FastEstimate(text string) int {
	charCount := len(text) // Use byte length for speed
	return charCount / 4
}

// FastEstimateMessages provides a fast estimate for multiple messages.
func FastEstimateMessages(messages []models.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 10
	}
	return totalChars / 4
}

// EstimateRateLimitTokens estimates tokens for rate limiting purposes.
// This is the function that should be used by rate limiters for fast checks.
func EstimateRateLimitTokens(messages []models.Message, maxTokens int) int {
	inputEstimate := FastEstimateMessages(messages)
	return inputEstimate + maxTokens
}