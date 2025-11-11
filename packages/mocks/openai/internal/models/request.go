// Package models provides core data structures for the OpenAI mock server.
// This file defines request types that match OpenAI's API request format exactly.
package models

import (
	"encoding/json"
	"fmt"
)

// Message represents a chat message in the conversation.
// This matches OpenAI's message format exactly.
type Message struct {
	// Role is the role of the message author (user, assistant, system, function)
	Role string `json:"role"`

	// Content is the content of the message
	Content string `json:"content"`

	// Name is the name of the author (optional, for function calls)
	Name *string `json:"name,omitempty"`

	// FunctionCall is the function call made by the assistant (optional)
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// FunctionCall represents a function call made by the assistant.
type FunctionCall struct {
	// Name is the name of the function to call
	Name string `json:"name"`

	// Arguments is a JSON string of arguments to pass to the function
	Arguments string `json:"arguments"`
}

// Function represents a function definition for function calling.
type Function struct {
	// Name is the name of the function
	Name string `json:"name"`

	// Description describes what the function does
	Description string `json:"description,omitempty"`

	// Parameters is a JSON Schema describing the function parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// ChatCompletionRequest represents a request to the /v1/chat/completions endpoint.
// This structure matches OpenAI's API exactly for production parity.
type ChatCompletionRequest struct {
	// Model is the model to use (e.g., "gpt-4o", "gpt-3.5-turbo")
	Model string `json:"model"`

	// Messages is the list of messages in the conversation
	Messages []Message `json:"messages"`

	// Temperature controls randomness (0.0 to 2.0, default 1.0)
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling (0.0 to 1.0)
	TopP *float64 `json:"top_p,omitempty"`

	// N is the number of completions to generate (default 1)
	N *int `json:"n,omitempty"`

	// Stream enables streaming responses via Server-Sent Events
	Stream bool `json:"stream,omitempty"`

	// Stop is a list of sequences where the API will stop generating
	Stop interface{} `json:"stop,omitempty"` // Can be string or []string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// PresencePenalty penalizes new tokens based on whether they appear in the text so far
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes new tokens based on their frequency in the text so far
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// LogitBias modifies the likelihood of specified tokens appearing
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// User is a unique identifier representing your end-user
	User *string `json:"user,omitempty"`

	// Functions is a list of functions the model may call
	Functions []Function `json:"functions,omitempty"`

	// FunctionCall controls how the model calls functions
	FunctionCall interface{} `json:"function_call,omitempty"` // Can be string or object

	// ResponseFormat specifies the format of the response
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Seed for deterministic sampling (beta)
	Seed *int `json:"seed,omitempty"`

	// Tools is a list of tools the model may call (new format)
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice controls which tool the model should use
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// ResponseFormat specifies the format of the model's output.
type ResponseFormat struct {
	// Type is the format type ("text" or "json_object")
	Type string `json:"type"`
}

// Tool represents a tool that the model can use.
type Tool struct {
	// Type is the type of tool ("function")
	Type string `json:"type"`

	// Function is the function definition
	Function Function `json:"function"`
}

// Validate validates the ChatCompletionRequest.
func (r *ChatCompletionRequest) Validate() error {
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(r.Messages) == 0 {
		return fmt.Errorf("messages array cannot be empty")
	}

	// Validate messages
	for i, msg := range r.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message[%d]: role is required", i)
		}
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" && msg.Role != "function" {
			return fmt.Errorf("message[%d]: invalid role '%s'", i, msg.Role)
		}
		if msg.Content == "" && msg.FunctionCall == nil {
			return fmt.Errorf("message[%d]: content or function_call is required", i)
		}
	}

	// Validate temperature
	if r.Temperature != nil {
		if *r.Temperature < 0 || *r.Temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
	}

	// Validate top_p
	if r.TopP != nil {
		if *r.TopP < 0 || *r.TopP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1")
		}
	}

	// Validate n
	if r.N != nil {
		if *r.N < 1 || *r.N > 128 {
			return fmt.Errorf("n must be between 1 and 128")
		}
	}

	// Validate max_tokens
	if r.MaxTokens < 0 {
		return fmt.Errorf("max_tokens cannot be negative")
	}

	// Validate presence_penalty
	if r.PresencePenalty != nil {
		if *r.PresencePenalty < -2 || *r.PresencePenalty > 2 {
			return fmt.Errorf("presence_penalty must be between -2 and 2")
		}
	}

	// Validate frequency_penalty
	if r.FrequencyPenalty != nil {
		if *r.FrequencyPenalty < -2 || *r.FrequencyPenalty > 2 {
			return fmt.Errorf("frequency_penalty must be between -2 and 2")
		}
	}

	return nil
}

// GetEffectiveTemperature returns the temperature to use (default 1.0).
func (r *ChatCompletionRequest) GetEffectiveTemperature() float64 {
	if r.Temperature != nil {
		return *r.Temperature
	}
	return 1.0
}

// GetEffectiveN returns the number of completions to generate (default 1).
func (r *ChatCompletionRequest) GetEffectiveN() int {
	if r.N != nil {
		return *r.N
	}
	return 1
}

// CompletionRequest represents a request to the /v1/completions endpoint (legacy).
type CompletionRequest struct {
	// Model is the model to use
	Model string `json:"model"`

	// Prompt is the text prompt(s) to generate completions for
	Prompt interface{} `json:"prompt"` // Can be string or []string

	// Suffix comes after the completion
	Suffix *string `json:"suffix,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP *float64 `json:"top_p,omitempty"`

	// N is the number of completions to generate
	N *int `json:"n,omitempty"`

	// Stream enables streaming responses
	Stream bool `json:"stream,omitempty"`

	// LogProbs includes log probabilities in the output
	LogProbs *int `json:"logprobs,omitempty"`

	// Echo echoes back the prompt in addition to the completion
	Echo bool `json:"echo,omitempty"`

	// Stop is a list of sequences where the API will stop generating
	Stop interface{} `json:"stop,omitempty"`

	// PresencePenalty penalizes new tokens
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes frequent tokens
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// BestOf generates multiple completions and returns the best one
	BestOf *int `json:"best_of,omitempty"`

	// LogitBias modifies token likelihoods
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// User is a unique identifier for the end-user
	User *string `json:"user,omitempty"`
}

// Validate validates the CompletionRequest.
func (r *CompletionRequest) Validate() error {
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}

	if r.Prompt == nil {
		return fmt.Errorf("prompt is required")
	}

	// Validate temperature
	if r.Temperature != nil {
		if *r.Temperature < 0 || *r.Temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
	}

	// Validate top_p
	if r.TopP != nil {
		if *r.TopP < 0 || *r.TopP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1")
		}
	}

	return nil
}

// EmbeddingRequest represents a request to the /v1/embeddings endpoint.
type EmbeddingRequest struct {
	// Model is the embedding model to use (e.g., "text-embedding-ada-002")
	Model string `json:"model"`

	// Input is the text(s) to get embeddings for
	Input interface{} `json:"input"` // Can be string or []string

	// User is a unique identifier for the end-user
	User *string `json:"user,omitempty"`

	// EncodingFormat specifies the format of the returned embeddings
	EncodingFormat *string `json:"encoding_format,omitempty"`

	// Dimensions is the number of dimensions for the embedding (optional)
	Dimensions *int `json:"dimensions,omitempty"`
}

// Validate validates the EmbeddingRequest.
func (r *EmbeddingRequest) Validate() error {
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}

	if r.Input == nil {
		return fmt.Errorf("input is required")
	}

	// Validate encoding format
	if r.EncodingFormat != nil {
		format := *r.EncodingFormat
		if format != "float" && format != "base64" {
			return fmt.Errorf("encoding_format must be 'float' or 'base64'")
		}
	}

	return nil
}

// ImageGenerationRequest represents a request to the /v1/images/generations endpoint.
type ImageGenerationRequest struct {
	// Prompt is the text description of the desired image
	Prompt string `json:"prompt"`

	// Model is the model to use (e.g., "dall-e-3", "dall-e-2")
	Model *string `json:"model,omitempty"`

	// N is the number of images to generate (1-10 for DALL-E 2, only 1 for DALL-E 3)
	N *int `json:"n,omitempty"`

	// Quality is the quality of the image ("standard" or "hd")
	Quality *string `json:"quality,omitempty"`

	// ResponseFormat is the format of the response ("url" or "b64_json")
	ResponseFormat *string `json:"response_format,omitempty"`

	// Size is the size of the generated images
	Size *string `json:"size,omitempty"`

	// Style is the style of the generated images ("vivid" or "natural")
	Style *string `json:"style,omitempty"`

	// User is a unique identifier for the end-user
	User *string `json:"user,omitempty"`
}

// Validate validates the ImageGenerationRequest.
func (r *ImageGenerationRequest) Validate() error {
	if r.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	// Validate n
	if r.N != nil {
		if *r.N < 1 || *r.N > 10 {
			return fmt.Errorf("n must be between 1 and 10")
		}
	}

	// Validate quality
	if r.Quality != nil {
		quality := *r.Quality
		if quality != "standard" && quality != "hd" {
			return fmt.Errorf("quality must be 'standard' or 'hd'")
		}
	}

	// Validate response_format
	if r.ResponseFormat != nil {
		format := *r.ResponseFormat
		if format != "url" && format != "b64_json" {
			return fmt.Errorf("response_format must be 'url' or 'b64_json'")
		}
	}

	// Validate style
	if r.Style != nil {
		style := *r.Style
		if style != "vivid" && style != "natural" {
			return fmt.Errorf("style must be 'vivid' or 'natural'")
		}
	}

	return nil
}

// GetEffectiveModel returns the model to use (default "dall-e-2").
func (r *ImageGenerationRequest) GetEffectiveModel() string {
	if r.Model != nil {
		return *r.Model
	}
	return "dall-e-2"
}

// GetEffectiveN returns the number of images to generate (default 1).
func (r *ImageGenerationRequest) GetEffectiveN() int {
	if r.N != nil {
		return *r.N
	}
	return 1
}

// ParseRequest parses a JSON request body into the appropriate request type.
func ParseRequest(data []byte, req interface{}) error {
	if err := json.Unmarshal(data, req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}