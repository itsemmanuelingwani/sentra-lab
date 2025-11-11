// Package models provides core data structures for the OpenAI mock server.
// This file defines response types that match OpenAI's API response format exactly.
package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Usage represents token usage information in API responses.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used
	TotalTokens int `json:"total_tokens"`
}

// Choice represents a single completion choice in chat completions.
type Choice struct {
	// Index is the index of this choice in the list
	Index int `json:"index"`

	// Message is the generated message
	Message Message `json:"message"`

	// FinishReason indicates why the completion finished
	// ("stop", "length", "content_filter", "function_call")
	FinishReason string `json:"finish_reason"`

	// LogProbs contains log probability information (optional)
	LogProbs *LogProbs `json:"logprobs,omitempty"`
}

// LogProbs contains log probability information for tokens.
type LogProbs struct {
	// Content is the log probability information for content tokens
	Content []TokenLogProb `json:"content"`
}

// TokenLogProb contains log probability information for a single token.
type TokenLogProb struct {
	// Token is the token string
	Token string `json:"token"`

	// LogProb is the log probability of the token
	LogProb float64 `json:"logprob"`

	// Bytes are the bytes of the token (optional)
	Bytes []int `json:"bytes,omitempty"`

	// TopLogProbs are alternative tokens with their log probabilities
	TopLogProbs []TopLogProb `json:"top_logprobs,omitempty"`
}

// TopLogProb represents an alternative token with its log probability.
type TopLogProb struct {
	// Token is the token string
	Token string `json:"token"`

	// LogProb is the log probability of the token
	LogProb float64 `json:"logprob"`

	// Bytes are the bytes of the token (optional)
	Bytes []int `json:"bytes,omitempty"`
}

// ChatCompletionResponse represents a response from the /v1/chat/completions endpoint.
// This structure matches OpenAI's API exactly for production parity.
type ChatCompletionResponse struct {
	// ID is the unique identifier for this completion
	ID string `json:"id"`

	// Object is always "chat.completion"
	Object string `json:"object"`

	// Created is the Unix timestamp when the completion was created
	Created int64 `json:"created"`

	// Model is the model used for the completion
	Model string `json:"model"`

	// SystemFingerprint is a fingerprint for the system configuration
	SystemFingerprint *string `json:"system_fingerprint,omitempty"`

	// Choices is the list of completion choices
	Choices []Choice `json:"choices"`

	// Usage contains token usage information
	Usage Usage `json:"usage"`
}

// NewChatCompletionResponse creates a new ChatCompletionResponse with default values.
func NewChatCompletionResponse(model string, message Message, usage Usage) *ChatCompletionResponse {
	return &ChatCompletionResponse{
		ID:      generateID("chatcmpl"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}
}

// ToJSON converts the response to JSON bytes.
func (r *ChatCompletionResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// StreamChoice represents a single choice in a streaming chat completion.
type StreamChoice struct {
	// Index is the index of this choice in the list
	Index int `json:"index"`

	// Delta contains the incremental message update
	Delta Delta `json:"delta"`

	// FinishReason indicates why the completion finished (null until final chunk)
	FinishReason *string `json:"finish_reason"`

	// LogProbs contains log probability information (optional)
	LogProbs *LogProbs `json:"logprobs,omitempty"`
}

// Delta represents an incremental update in streaming responses.
type Delta struct {
	// Role is the role of the message (only in first chunk)
	Role string `json:"role,omitempty"`

	// Content is the incremental content
	Content string `json:"content,omitempty"`

	// FunctionCall is the incremental function call (optional)
	FunctionCall *FunctionCall `json:"function_call,omitempty"`

	// ToolCalls are incremental tool calls (optional)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call in the response.
type ToolCall struct {
	// Index is the index of the tool call
	Index int `json:"index"`

	// ID is the identifier for the tool call
	ID string `json:"id"`

	// Type is the type of tool call ("function")
	Type string `json:"type"`

	// Function is the function call
	Function FunctionCall `json:"function"`
}

// StreamChunk represents a single chunk in a streaming chat completion response.
// This is sent as a Server-Sent Event (SSE).
type StreamChunk struct {
	// ID is the unique identifier for this completion
	ID string `json:"id"`

	// Object is always "chat.completion.chunk"
	Object string `json:"object"`

	// Created is the Unix timestamp when the chunk was created
	Created int64 `json:"created"`

	// Model is the model used for the completion
	Model string `json:"model"`

	// SystemFingerprint is a fingerprint for the system configuration
	SystemFingerprint *string `json:"system_fingerprint,omitempty"`

	// Choices is the list of streaming choices
	Choices []StreamChoice `json:"choices"`
}

// NewStreamChunk creates a new StreamChunk with the given parameters.
func NewStreamChunk(id string, model string, delta Delta, finishReason *string) *StreamChunk {
	return &StreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        delta,
				FinishReason: finishReason,
			},
		},
	}
}

// ToSSE converts the chunk to Server-Sent Event format.
func (c *StreamChunk) ToSSE() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("data: %s\n\n", string(data)), nil
}

// CompletionChoice represents a single choice in legacy completions.
type CompletionChoice struct {
	// Text is the generated text
	Text string `json:"text"`

	// Index is the index of this choice
	Index int `json:"index"`

	// LogProbs contains log probability information (optional)
	LogProbs *LogProbs `json:"logprobs,omitempty"`

	// FinishReason indicates why the completion finished
	FinishReason string `json:"finish_reason"`
}

// CompletionResponse represents a response from the /v1/completions endpoint (legacy).
type CompletionResponse struct {
	// ID is the unique identifier for this completion
	ID string `json:"id"`

	// Object is always "text_completion"
	Object string `json:"object"`

	// Created is the Unix timestamp when the completion was created
	Created int64 `json:"created"`

	// Model is the model used for the completion
	Model string `json:"model"`

	// Choices is the list of completion choices
	Choices []CompletionChoice `json:"choices"`

	// Usage contains token usage information
	Usage Usage `json:"usage"`
}

// NewCompletionResponse creates a new CompletionResponse with default values.
func NewCompletionResponse(model string, text string, usage Usage) *CompletionResponse {
	return &CompletionResponse{
		ID:      generateID("cmpl"),
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []CompletionChoice{
			{
				Text:         text,
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}
}

// Embedding represents a single embedding vector.
type Embedding struct {
	// Object is always "embedding"
	Object string `json:"object"`

	// Embedding is the embedding vector
	Embedding []float64 `json:"embedding"`

	// Index is the index of this embedding in the list
	Index int `json:"index"`
}

// EmbeddingResponse represents a response from the /v1/embeddings endpoint.
type EmbeddingResponse struct {
	// Object is always "list"
	Object string `json:"object"`

	// Data is the list of embeddings
	Data []Embedding `json:"data"`

	// Model is the model used for the embeddings
	Model string `json:"model"`

	// Usage contains token usage information
	Usage Usage `json:"usage"`
}

// NewEmbeddingResponse creates a new EmbeddingResponse.
func NewEmbeddingResponse(model string, embeddings [][]float64, usage Usage) *EmbeddingResponse {
	data := make([]Embedding, len(embeddings))
	for i, emb := range embeddings {
		data[i] = Embedding{
			Object:    "embedding",
			Embedding: emb,
			Index:     i,
		}
	}

	return &EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  model,
		Usage:  usage,
	}
}

// ImageData represents a generated image.
type ImageData struct {
	// URL is the URL of the generated image
	URL string `json:"url,omitempty"`

	// B64JSON is the base64-encoded JSON of the image
	B64JSON string `json:"b64_json,omitempty"`

	// RevisedPrompt is the revised prompt used to generate the image
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageResponse represents a response from the /v1/images/generations endpoint.
type ImageResponse struct {
	// Created is the Unix timestamp when the images were created
	Created int64 `json:"created"`

	// Data is the list of generated images
	Data []ImageData `json:"data"`
}

// NewImageResponse creates a new ImageResponse.
func NewImageResponse(urls []string) *ImageResponse {
	data := make([]ImageData, len(urls))
	for i, url := range urls {
		data[i] = ImageData{
			URL: url,
		}
	}

	return &ImageResponse{
		Created: time.Now().Unix(),
		Data:    data,
	}
}

// Model represents a model object returned by the /v1/models endpoint.
type Model struct {
	// ID is the model identifier
	ID string `json:"id"`

	// Object is always "model"
	Object string `json:"object"`

	// Created is the Unix timestamp when the model was created
	Created int64 `json:"created"`

	// OwnedBy indicates who owns the model
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents a response from the /v1/models endpoint.
type ModelsResponse struct {
	// Object is always "list"
	Object string `json:"object"`

	// Data is the list of models
	Data []Model `json:"data"`
}

// NewModelsResponse creates a new ModelsResponse from a list of model configs.
func NewModelsResponse(configs []ModelConfig) *ModelsResponse {
	data := make([]Model, len(configs))
	for i, config := range configs {
		data[i] = Model{
			ID:      config.ID,
			Object:  "model",
			Created: config.Created,
			OwnedBy: config.OwnedBy,
		}
	}

	return &ModelsResponse{
		Object: "list",
		Data:   data,
	}
}

// generateID generates a unique ID with the given prefix.
func generateID(prefix string) string {
	// Format: prefix-<unix-timestamp>-<random-suffix>
	// Example: chatcmpl-1234567890-abc123
	timestamp := time.Now().Unix()
	suffix := generateRandomString(16)
	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, suffix)
}

// generateRandomString generates a random alphanumeric string of the given length.
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		// Use time-based pseudo-randomness for reproducibility in tests
		b[i] = charset[(time.Now().UnixNano()+int64(i))%int64(len(charset))]
	}
	return string(b)
}