// Package latency provides latency simulation.
// This file implements streaming-specific latency patterns for SSE responses.
package latency

import (
	"context"
	"time"
)

// StreamingSimulator manages latency for streaming responses.
// Streaming has different characteristics than non-streaming requests.
type StreamingSimulator struct {
	// simulator is the base latency simulator
	simulator *Simulator

	// chunkStrategy determines how chunks are delayed
	chunkStrategy ChunkDelayStrategy
}

// ChunkDelayStrategy defines how streaming chunk delays are calculated.
type ChunkDelayStrategy string

const (
	// ConstantChunkDelay applies constant delay between all chunks
	ConstantChunkDelay ChunkDelayStrategy = "constant"

	// ProgressiveChunkDelay increases delay as generation progresses (realistic)
	ProgressiveChunkDelay ChunkDelayStrategy = "progressive"

	// RandomChunkDelay applies random delay between chunks
	RandomChunkDelay ChunkDelayStrategy = "random"

	// BurstChunkDelay sends chunks in bursts with pauses
	BurstChunkDelay ChunkDelayStrategy = "burst"
)

// NewStreamingSimulator creates a new streaming latency simulator.
func NewStreamingSimulator(simulator *Simulator, strategy ChunkDelayStrategy) *StreamingSimulator {
	if strategy == "" {
		strategy = ProgressiveChunkDelay // Default to most realistic
	}

	return &StreamingSimulator{
		simulator:     simulator,
		chunkStrategy: strategy,
	}
}

// CalculateChunkDelays calculates delays for each chunk in a streaming response.
func (ss *StreamingSimulator) CalculateChunkDelays(ctx context.Context, modelID string, numChunks int) ([]time.Duration, error) {
	if numChunks == 0 {
		return []time.Duration{}, nil
	}

	// Get base delays from simulator
	delays, err := ss.simulator.SimulateStreaming(ctx, modelID, numChunks)
	if err != nil {
		return nil, err
	}

	// Apply chunk strategy
	switch ss.chunkStrategy {
	case ConstantChunkDelay:
		return ss.constantDelays(delays)
	case ProgressiveChunkDelay:
		return ss.progressiveDelays(delays)
	case RandomChunkDelay:
		return ss.randomDelays(delays)
	case BurstChunkDelay:
		return ss.burstDelays(delays)
	default:
		return delays, nil
	}
}

// constantDelays applies constant delay between chunks.
func (ss *StreamingSimulator) constantDelays(baseDelays []time.Duration) ([]time.Duration, error) {
	if len(baseDelays) == 0 {
		return baseDelays, nil
	}

	// First chunk keeps original delay (TTFT)
	// Subsequent chunks get average of remaining delays
	avgDelay := time.Duration(0)
	for i := 1; i < len(baseDelays); i++ {
		avgDelay += baseDelays[i]
	}
	if len(baseDelays) > 1 {
		avgDelay /= time.Duration(len(baseDelays) - 1)
	}

	result := make([]time.Duration, len(baseDelays))
	result[0] = baseDelays[0] // Keep TTFT
	for i := 1; i < len(baseDelays); i++ {
		result[i] = avgDelay
	}

	return result, nil
}

// progressiveDelays increases delay slightly as generation progresses.
// This simulates the model "thinking more" for later tokens.
func (ss *StreamingSimulator) progressiveDelays(baseDelays []time.Duration) ([]time.Duration, error) {
	if len(baseDelays) == 0 {
		return baseDelays, nil
	}

	result := make([]time.Duration, len(baseDelays))
	result[0] = baseDelays[0] // Keep TTFT

	// Apply progressive scaling: 1.0x -> 1.2x over the response
	for i := 1; i < len(baseDelays); i++ {
		progress := float64(i) / float64(len(baseDelays))
		scale := 1.0 + (progress * 0.2) // 1.0 to 1.2
		result[i] = time.Duration(float64(baseDelays[i]) * scale)
	}

	return result, nil
}

// randomDelays applies random variance to chunk delays.
func (ss *StreamingSimulator) randomDelays(baseDelays []time.Duration) ([]time.Duration, error) {
	result := make([]time.Duration, len(baseDelays))

	for i, delay := range baseDelays {
		// Apply random jitter (±30% for chunks)
		jittered := ss.simulator.jitter.ApplyJitter(delay, 0.30)
		result[i] = jittered
	}

	return result, nil
}

// burstDelays sends chunks in bursts with pauses.
// Pattern: fast, fast, fast, pause, fast, fast, fast, pause...
func (ss *StreamingSimulator) burstDelays(baseDelays []time.Duration) ([]time.Duration, error) {
	if len(baseDelays) == 0 {
		return baseDelays, nil
	}

	result := make([]time.Duration, len(baseDelays))
	result[0] = baseDelays[0] // Keep TTFT

	burstSize := 3 // Send 3 chunks quickly, then pause
	pauseMultiplier := 3.0 // Pause is 3x longer than normal delay

	for i := 1; i < len(baseDelays); i++ {
		position := (i - 1) % (burstSize + 1)

		if position < burstSize {
			// Fast chunk (50% of base delay)
			result[i] = baseDelays[i] / 2
		} else {
			// Pause chunk (3x base delay)
			result[i] = time.Duration(float64(baseDelays[i]) * pauseMultiplier)
		}
	}

	return result, nil
}

// StreamChunk represents a single chunk to send in a stream.
type StreamChunk struct {
	// Content is the chunk content
	Content string

	// Delay is how long to wait before sending this chunk
	Delay time.Duration

	// Index is the chunk index
	Index int

	// IsFirst indicates if this is the first chunk (includes role)
	IsFirst bool

	// IsLast indicates if this is the last chunk (includes finish_reason)
	IsLast bool
}

// GenerateStreamPlan generates a complete streaming plan.
func (ss *StreamingSimulator) GenerateStreamPlan(ctx context.Context, modelID string, content string, chunkSize int) ([]StreamChunk, error) {
	// Split content into chunks
	chunks := splitContent(content, chunkSize)
	numChunks := len(chunks)

	// Calculate delays
	delays, err := ss.CalculateChunkDelays(ctx, modelID, numChunks)
	if err != nil {
		return nil, err
	}

	// Build plan
	plan := make([]StreamChunk, numChunks)
	for i := 0; i < numChunks; i++ {
		plan[i] = StreamChunk{
			Content: chunks[i],
			Delay:   delays[i],
			Index:   i,
			IsFirst: i == 0,
			IsLast:  i == numChunks-1,
		}
	}

	return plan, nil
}

// splitContent splits content into chunks of approximately chunkSize.
func splitContent(content string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 3 // Default: ~3 tokens per chunk
	}

	// Simple word-based splitting
	words := splitWords(content)
	if len(words) == 0 {
		return []string{content}
	}

	var chunks []string
	var currentChunk string
	wordCount := 0

	for _, word := range words {
		if wordCount > 0 {
			currentChunk += " "
		}
		currentChunk += word
		wordCount++

		if wordCount >= chunkSize {
			chunks = append(chunks, currentChunk)
			currentChunk = ""
			wordCount = 0
		}
	}

	// Add remaining content
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// splitWords splits text into words preserving punctuation.
func splitWords(text string) []string {
	var words []string
	var currentWord string

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(char)
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}

// StreamMetrics tracks streaming performance metrics.
type StreamMetrics struct {
	// TTFT is Time To First Token (first chunk delay)
	TTFT time.Duration

	// TotalStreamTime is the total time for all chunks
	TotalStreamTime time.Duration

	// NumChunks is the number of chunks sent
	NumChunks int

	// AverageChunkDelay is the average delay between chunks
	AverageChunkDelay time.Duration

	// MaxChunkDelay is the maximum chunk delay
	MaxChunkDelay time.Duration

	// MinChunkDelay is the minimum chunk delay (excluding first)
	MinChunkDelay time.Duration
}

// CalculateStreamMetrics calculates metrics from a stream plan.
func CalculateStreamMetrics(plan []StreamChunk) StreamMetrics {
	if len(plan) == 0 {
		return StreamMetrics{}
	}

	metrics := StreamMetrics{
		TTFT:          plan[0].Delay,
		NumChunks:     len(plan),
		MinChunkDelay: time.Hour, // Start high
	}

	totalDelay := time.Duration(0)
	for i, chunk := range plan {
		totalDelay += chunk.Delay

		if i > 0 { // Skip first chunk for chunk-specific metrics
			if chunk.Delay > metrics.MaxChunkDelay {
				metrics.MaxChunkDelay = chunk.Delay
			}
			if chunk.Delay < metrics.MinChunkDelay {
				metrics.MinChunkDelay = chunk.Delay
			}
		}
	}

	metrics.TotalStreamTime = totalDelay
	if len(plan) > 1 {
		metrics.AverageChunkDelay = totalDelay / time.Duration(len(plan))
	}

	return metrics
}

// SetChunkStrategy changes the chunk delay strategy.
func (ss *StreamingSimulator) SetChunkStrategy(strategy ChunkDelayStrategy) {
	ss.chunkStrategy = strategy
}

// GetChunkStrategy returns the current chunk delay strategy.
func (ss *StreamingSimulator) GetChunkStrategy() ChunkDelayStrategy {
	return ss.chunkStrategy
}