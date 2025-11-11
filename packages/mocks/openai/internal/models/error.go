// Package models provides core data structures for the OpenAI mock server.
// This file defines error types and structures that match OpenAI's API error format.
package models

import (
	"encoding/json"
	"fmt"
)

// ErrorType represents the type of error returned by the OpenAI API.
// These match OpenAI's actual error types for production parity.
type ErrorType string

const (
	// ErrorTypeRateLimit is returned when rate limits are exceeded
	ErrorTypeRateLimit ErrorType = "rate_limit_exceeded"

	// ErrorTypeServerError is returned for internal server errors
	ErrorTypeServerError ErrorType = "server_error"

	// ErrorTypeServiceUnavailable is returned when the service is overloaded
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable"

	// ErrorTypeBadRequest is returned for invalid request parameters
	ErrorTypeBadRequest ErrorType = "invalid_request_error"

	// ErrorTypeTimeout is returned when a request times out
	ErrorTypeTimeout ErrorType = "timeout"

	// ErrorTypeInvalidAuth is returned for authentication failures
	ErrorTypeInvalidAuth ErrorType = "invalid_api_key"

	// ErrorTypeInsufficientQuota is returned when quota is exhausted
	ErrorTypeInsufficientQuota ErrorType = "insufficient_quota"

	// ErrorTypeModelNotFound is returned when model doesn't exist
	ErrorTypeModelNotFound ErrorType = "model_not_found"

	// ErrorTypeContextLengthExceeded is returned when input is too long
	ErrorTypeContextLengthExceeded ErrorType = "context_length_exceeded"
)

// APIError represents an error in the OpenAI API format.
// This structure matches OpenAI's error response format exactly.
type APIError struct {
	// Type is the error type (e.g., "rate_limit_exceeded")
	Type ErrorType `json:"type"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Param is the parameter that caused the error (optional)
	Param *string `json:"param,omitempty"`

	// Code is an error code (optional)
	Code *string `json:"code,omitempty"`

	// StatusCode is the HTTP status code (not sent in JSON response)
	StatusCode int `json:"-"`

	// RetryAfter is the number of seconds to wait before retrying (not sent in JSON)
	RetryAfter int `json:"-"`
}

// Error implements the error interface for APIError.
func (e APIError) Error() string {
	return fmt.Sprintf("[%s] %s (status: %d)", e.Type, e.Message, e.StatusCode)
}

// ErrorResponse is the top-level error response structure.
// This matches OpenAI's error response format: { "error": { ... } }
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// NewAPIError creates a new APIError with the given parameters.
// This is a convenience constructor for creating errors.
func NewAPIError(errType ErrorType, message string, statusCode int) APIError {
	return APIError{
		Type:       errType,
		Message:    message,
		StatusCode: statusCode,
		RetryAfter: 0,
	}
}

// NewRateLimitError creates a rate limit exceeded error.
func NewRateLimitError(message string, retryAfter int) APIError {
	return APIError{
		Type:       ErrorTypeRateLimit,
		Message:    message,
		StatusCode: 429,
		RetryAfter: retryAfter,
	}
}

// NewServerError creates an internal server error.
func NewServerError(message string) APIError {
	return APIError{
		Type:       ErrorTypeServerError,
		Message:    message,
		StatusCode: 500,
		RetryAfter: 0,
	}
}

// NewServiceUnavailableError creates a service unavailable error.
func NewServiceUnavailableError(message string, retryAfter int) APIError {
	return APIError{
		Type:       ErrorTypeServiceUnavailable,
		Message:    message,
		StatusCode: 503,
		RetryAfter: retryAfter,
	}
}

// NewBadRequestError creates a bad request error.
func NewBadRequestError(message string, param *string) APIError {
	return APIError{
		Type:       ErrorTypeBadRequest,
		Message:    message,
		Param:      param,
		StatusCode: 400,
		RetryAfter: 0,
	}
}

// NewInvalidAuthError creates an authentication error.
func NewInvalidAuthError(message string) APIError {
	return APIError{
		Type:       ErrorTypeInvalidAuth,
		Message:    message,
		StatusCode: 401,
		RetryAfter: 0,
	}
}

// NewModelNotFoundError creates a model not found error.
func NewModelNotFoundError(model string) APIError {
	return APIError{
		Type:       ErrorTypeModelNotFound,
		Message:    fmt.Sprintf("The model '%s' does not exist", model),
		StatusCode: 404,
		RetryAfter: 0,
	}
}

// NewContextLengthError creates a context length exceeded error.
func NewContextLengthError(requestedTokens, maxTokens int) APIError {
	return APIError{
		Type: ErrorTypeContextLengthExceeded,
		Message: fmt.Sprintf(
			"This model's maximum context length is %d tokens. However, you requested %d tokens. Please reduce the length of the messages.",
			maxTokens,
			requestedTokens,
		),
		StatusCode: 400,
		RetryAfter: 0,
	}
}

// ToJSON converts an APIError to a JSON ErrorResponse.
func (e APIError) ToJSON() ([]byte, error) {
	response := ErrorResponse{Error: e}
	return json.Marshal(response)
}

// IsRetryable returns true if the error is retryable.
func (e APIError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit, ErrorTypeServiceUnavailable, ErrorTypeServerError:
		return true
	default:
		return false
	}
}

// GetRetryAfter returns the number of seconds to wait before retrying.
// Returns 0 if the error is not retryable.
func (e APIError) GetRetryAfter() int {
	if e.IsRetryable() {
		if e.RetryAfter > 0 {
			return e.RetryAfter
		}
		// Default retry delays based on error type
		switch e.Type {
		case ErrorTypeRateLimit:
			return 60 // 1 minute
		case ErrorTypeServiceUnavailable:
			return 30 // 30 seconds
		case ErrorTypeServerError:
			return 10 // 10 seconds
		default:
			return 0
		}
	}
	return 0
}

// Validate checks if the error has valid fields.
func (e APIError) Validate() error {
	if e.Type == "" {
		return fmt.Errorf("error type cannot be empty")
	}
	if e.Message == "" {
		return fmt.Errorf("error message cannot be empty")
	}
	if e.StatusCode < 400 || e.StatusCode > 599 {
		return fmt.Errorf("invalid HTTP status code: %d", e.StatusCode)
	}
	return nil
}