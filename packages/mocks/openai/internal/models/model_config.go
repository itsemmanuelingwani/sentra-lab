// Package models provides core data structures for the OpenAI mock server.
// This file defines model configurations including latency profiles and pricing.
package models

import (
	"fmt"
	"time"
)

// ModelConfig defines the complete configuration for a model.
// This includes metadata, capabilities, latency profile, and pricing.
type ModelConfig struct {
	// ID is the model identifier (e.g., "gpt-4o")
	ID string

	// Object is always "model"
	Object string

	// Created is the Unix timestamp when the model was created
	Created int64

	// OwnedBy indicates who owns the model (e.g., "openai")
	OwnedBy string

	// ContextWindow is the maximum context length in tokens
	ContextWindow int

	// MaxOutputTokens is the maximum number of output tokens
	MaxOutputTokens int

	// Encoding is the tokenizer encoding to use (e.g., "cl100k_base", "o200k_base")
	Encoding string

	// SupportsVision indicates if the model supports vision inputs
	SupportsVision bool

	// SupportsFunctionCalling indicates if the model supports function calling
	SupportsFunctionCalling bool

	// SupportsJSON indicates if the model supports JSON mode
	SupportsJSON bool

	// Latency profile (for production-realistic simulation)
	BaseLatency     time.Duration // Time To First Token (TTFT)
	PerTokenLatency time.Duration // Latency per output token
	JitterPercent   float64       // Jitter as percentage (e.g., 0.25 = ±25%)

	// Pricing (in USD per 1 million tokens)
	InputPer1M       float64 // Input token price
	OutputPer1M      float64 // Output token price
	CachedInputPer1M float64 // Cached input token price (if supported)
}

// Validate validates the model configuration.
func (c ModelConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}
	if c.ContextWindow <= 0 {
		return fmt.Errorf("context window must be positive")
	}
	if c.MaxOutputTokens <= 0 {
		return fmt.Errorf("max output tokens must be positive")
	}
	if c.Encoding == "" {
		return fmt.Errorf("encoding cannot be empty")
	}
	if c.BaseLatency < 0 {
		return fmt.Errorf("base latency cannot be negative")
	}
	if c.PerTokenLatency < 0 {
		return fmt.Errorf("per-token latency cannot be negative")
	}
	if c.JitterPercent < 0 || c.JitterPercent > 1 {
		return fmt.Errorf("jitter percent must be between 0 and 1")
	}
	if c.InputPer1M < 0 {
		return fmt.Errorf("input price cannot be negative")
	}
	if c.OutputPer1M < 0 {
		return fmt.Errorf("output price cannot be negative")
	}
	return nil
}

// CalculateMaxContextTokens returns the maximum tokens that can fit in context.
// This is context window minus estimated output tokens.
func (c ModelConfig) CalculateMaxContextTokens(requestedMaxTokens int) int {
	maxOutput := requestedMaxTokens
	if maxOutput == 0 || maxOutput > c.MaxOutputTokens {
		maxOutput = c.MaxOutputTokens
	}
	return c.ContextWindow - maxOutput
}

// IsValidContextLength checks if the requested token count fits in context.
func (c ModelConfig) IsValidContextLength(inputTokens, outputTokens int) bool {
	return inputTokens+outputTokens <= c.ContextWindow
}

// ModelConfigs is the global registry of model configurations.
// This matches OpenAI's actual models as of November 2025.
var ModelConfigs = map[string]ModelConfig{
	"gpt-4o": {
		ID:                      "gpt-4o",
		Object:                  "model",
		Created:                 1715367049,
		OwnedBy:                 "openai",
		ContextWindow:           128000,
		MaxOutputTokens:         16384,
		Encoding:                "o200k_base",
		SupportsVision:          true,
		SupportsFunctionCalling: true,
		SupportsJSON:            true,
		BaseLatency:             500 * time.Millisecond,
		PerTokenLatency:         20 * time.Millisecond,
		JitterPercent:           0.25,
		InputPer1M:              2.50,
		OutputPer1M:             10.00,
		CachedInputPer1M:        1.25,
	},
	"gpt-4o-mini": {
		ID:                      "gpt-4o-mini",
		Object:                  "model",
		Created:                 1721172717,
		OwnedBy:                 "openai",
		ContextWindow:           128000,
		MaxOutputTokens:         16384,
		Encoding:                "o200k_base",
		SupportsVision:          true,
		SupportsFunctionCalling: true,
		SupportsJSON:            true,
		BaseLatency:             200 * time.Millisecond,
		PerTokenLatency:         34 * time.Millisecond,
		JitterPercent:           0.20,
		InputPer1M:              0.15,
		OutputPer1M:             0.60,
		CachedInputPer1M:        0.075,
	},
	"gpt-4-turbo": {
		ID:                      "gpt-4-turbo",
		Object:                  "model",
		Created:                 1712361441,
		OwnedBy:                 "openai",
		ContextWindow:           128000,
		MaxOutputTokens:         4096,
		Encoding:                "cl100k_base",
		SupportsVision:          true,
		SupportsFunctionCalling: true,
		SupportsJSON:            true,
		BaseLatency:             800 * time.Millisecond,
		PerTokenLatency:         50 * time.Millisecond,
		JitterPercent:           0.30,
		InputPer1M:              10.00,
		OutputPer1M:             30.00,
		CachedInputPer1M:        5.00,
	},
	"gpt-4": {
		ID:                      "gpt-4",
		Object:                  "model",
		Created:                 1687882411,
		OwnedBy:                 "openai",
		ContextWindow:           8192,
		MaxOutputTokens:         4096,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: true,
		SupportsJSON:            false,
		BaseLatency:             800 * time.Millisecond,
		PerTokenLatency:         196 * time.Millisecond,
		JitterPercent:           0.30,
		InputPer1M:              30.00,
		OutputPer1M:             60.00,
		CachedInputPer1M:        0,
	},
	"gpt-3.5-turbo": {
		ID:                      "gpt-3.5-turbo",
		Object:                  "model",
		Created:                 1677610602,
		OwnedBy:                 "openai",
		ContextWindow:           16385,
		MaxOutputTokens:         4096,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: true,
		SupportsJSON:            true,
		BaseLatency:             300 * time.Millisecond,
		PerTokenLatency:         73 * time.Millisecond,
		JitterPercent:           0.20,
		InputPer1M:              0.50,
		OutputPer1M:             1.50,
		CachedInputPer1M:        0,
	},
	"gpt-3.5-turbo-16k": {
		ID:                      "gpt-3.5-turbo-16k",
		Object:                  "model",
		Created:                 1683758102,
		OwnedBy:                 "openai",
		ContextWindow:           16385,
		MaxOutputTokens:         4096,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: true,
		SupportsJSON:            true,
		BaseLatency:             300 * time.Millisecond,
		PerTokenLatency:         73 * time.Millisecond,
		JitterPercent:           0.20,
		InputPer1M:              3.00,
		OutputPer1M:             4.00,
		CachedInputPer1M:        0,
	},
	"text-embedding-3-small": {
		ID:                      "text-embedding-3-small",
		Object:                  "model",
		Created:                 1705953180,
		OwnedBy:                 "openai",
		ContextWindow:           8191,
		MaxOutputTokens:         0, // Embeddings don't generate text
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: false,
		SupportsJSON:            false,
		BaseLatency:             100 * time.Millisecond,
		PerTokenLatency:         0,
		JitterPercent:           0.15,
		InputPer1M:              0.02,
		OutputPer1M:             0,
		CachedInputPer1M:        0,
	},
	"text-embedding-3-large": {
		ID:                      "text-embedding-3-large",
		Object:                  "model",
		Created:                 1705953180,
		OwnedBy:                 "openai",
		ContextWindow:           8191,
		MaxOutputTokens:         0,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: false,
		SupportsJSON:            false,
		BaseLatency:             200 * time.Millisecond,
		PerTokenLatency:         0,
		JitterPercent:           0.15,
		InputPer1M:              0.13,
		OutputPer1M:             0,
		CachedInputPer1M:        0,
	},
	"text-embedding-ada-002": {
		ID:                      "text-embedding-ada-002",
		Object:                  "model",
		Created:                 1671217299,
		OwnedBy:                 "openai",
		ContextWindow:           8191,
		MaxOutputTokens:         0,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: false,
		SupportsJSON:            false,
		BaseLatency:             150 * time.Millisecond,
		PerTokenLatency:         0,
		JitterPercent:           0.15,
		InputPer1M:              0.10,
		OutputPer1M:             0,
		CachedInputPer1M:        0,
	},
	"dall-e-3": {
		ID:                      "dall-e-3",
		Object:                  "model",
		Created:                 1698785189,
		OwnedBy:                 "openai",
		ContextWindow:           4000, // Prompt token limit
		MaxOutputTokens:         0,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: false,
		SupportsJSON:            false,
		BaseLatency:             15 * time.Second, // Images take longer
		PerTokenLatency:         0,
		JitterPercent:           0.20,
		InputPer1M:              0,    // Priced per image, not tokens
		OutputPer1M:             0,
		CachedInputPer1M:        0,
	},
	"dall-e-2": {
		ID:                      "dall-e-2",
		Object:                  "model",
		Created:                 1698785189,
		OwnedBy:                 "openai",
		ContextWindow:           1000,
		MaxOutputTokens:         0,
		Encoding:                "cl100k_base",
		SupportsVision:          false,
		SupportsFunctionCalling: false,
		SupportsJSON:            false,
		BaseLatency:             10 * time.Second,
		PerTokenLatency:         0,
		JitterPercent:           0.20,
		InputPer1M:              0,
		OutputPer1M:             0,
		CachedInputPer1M:        0,
	},
}

// GetModelConfig retrieves a model configuration by ID.
// Returns an error if the model doesn't exist.
func GetModelConfig(modelID string) (ModelConfig, error) {
	config, ok := ModelConfigs[modelID]
	if !ok {
		return ModelConfig{}, fmt.Errorf("model '%s' not found", modelID)
	}
	return config, nil
}

// GetAllModelConfigs returns a slice of all model configurations.
func GetAllModelConfigs() []ModelConfig {
	configs := make([]ModelConfig, 0, len(ModelConfigs))
	for _, config := range ModelConfigs {
		configs = append(configs, config)
	}
	return configs
}

// IsModelSupported checks if a model ID is supported.
func IsModelSupported(modelID string) bool {
	_, ok := ModelConfigs[modelID]
	return ok
}

// GetChatModels returns only chat completion models.
func GetChatModels() []ModelConfig {
	var chatModels []ModelConfig
	for _, config := range ModelConfigs {
		if config.MaxOutputTokens > 0 && config.Encoding != "" {
			// Chat models have output tokens and use tokenizers
			if config.ID != "text-embedding-3-small" &&
				config.ID != "text-embedding-3-large" &&
				config.ID != "text-embedding-ada-002" &&
				config.ID != "dall-e-3" &&
				config.ID != "dall-e-2" {
				chatModels = append(chatModels, config)
			}
		}
	}
	return chatModels
}

// GetEmbeddingModels returns only embedding models.
func GetEmbeddingModels() []ModelConfig {
	var embeddingModels []ModelConfig
	for _, config := range ModelConfigs {
		if config.MaxOutputTokens == 0 && config.InputPer1M > 0 {
			// Embedding models have no output tokens but have input pricing
			embeddingModels = append(embeddingModels, config)
		}
	}
	return embeddingModels
}

// GetImageModels returns only image generation models.
func GetImageModels() []ModelConfig {
	var imageModels []ModelConfig
	for id, config := range ModelConfigs {
		if id == "dall-e-3" || id == "dall-e-2" {
			imageModels = append(imageModels, config)
		}
	}
	return imageModels
}