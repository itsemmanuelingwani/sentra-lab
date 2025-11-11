// Package client provides a Go client for the OpenAI mock server.
// This client can be used for testing and integrating with the mock API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Client is a client for the OpenAI mock server.
type Client struct {
	// BaseURL is the base URL of the mock server (e.g., "http://localhost:8080")
	BaseURL string

	// APIKey is the API key for authentication (optional for mock)
	APIKey string

	// HTTPClient is the underlying HTTP client
	HTTPClient *http.Client

	// Timeout is the request timeout (default: 60 seconds)
	Timeout time.Duration
}

// NewClient creates a new OpenAI mock client.
func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		Timeout: 60 * time.Second,
	}
}

// NewClientWithHTTPClient creates a new client with a custom HTTP client.
func NewClientWithHTTPClient(baseURL string, apiKey string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: httpClient,
		Timeout:    60 * time.Second,
	}
}

// CreateChatCompletion creates a chat completion.
func (c *Client) CreateChatCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Make HTTP request
	endpoint := c.BaseURL + "/v1/chat/completions"
	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp)
	}

	// Parse response
	var chatResp models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// CreateChatCompletionStream creates a streaming chat completion.
// Returns a channel that receives stream chunks.
func (c *Client) CreateChatCompletionStream(ctx context.Context, req models.ChatCompletionRequest) (<-chan StreamResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Force streaming mode
	req.Stream = true

	// Make HTTP request
	endpoint := c.BaseURL + "/v1/chat/completions"
	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, c.handleErrorResponse(resp)
	}

	// Create channel for stream results
	results := make(chan StreamResult)

	// Start goroutine to read stream
	go func() {
		defer resp.Body.Close()
		defer close(results)

		reader := NewSSEReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				results <- StreamResult{Error: ctx.Err()}
				return
			default:
				event, err := reader.ReadEvent()
				if err != nil {
					if err != io.EOF {
						results <- StreamResult{Error: err}
					}
					return
				}

				// Check for [DONE] marker
				if event == "[DONE]" {
					return
				}

				// Parse chunk
				var chunk models.StreamChunk
				if err := json.Unmarshal([]byte(event), &chunk); err != nil {
					results <- StreamResult{Error: fmt.Errorf("failed to parse chunk: %w", err)}
					return
				}

				results <- StreamResult{Chunk: &chunk}
			}
		}
	}()

	return results, nil
}

// StreamResult represents a result from a streaming response.
type StreamResult struct {
	Chunk *models.StreamChunk
	Error error
}

// CreateCompletion creates a legacy completion.
func (c *Client) CreateCompletion(ctx context.Context, req models.CompletionRequest) (*models.CompletionResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Make HTTP request
	endpoint := c.BaseURL + "/v1/completions"
	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp)
	}

	// Parse response
	var compResp models.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&compResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &compResp, nil
}

// CreateEmbedding creates embeddings.
func (c *Client) CreateEmbedding(ctx context.Context, req models.EmbeddingRequest) (*models.EmbeddingResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Make HTTP request
	endpoint := c.BaseURL + "/v1/embeddings"
	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp)
	}

	// Parse response
	var embResp models.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &embResp, nil
}

// CreateImage generates images.
func (c *Client) CreateImage(ctx context.Context, req models.ImageGenerationRequest) (*models.ImageResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Make HTTP request
	endpoint := c.BaseURL + "/v1/images/generations"
	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp)
	}

	// Parse response
	var imgResp models.ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &imgResp, nil
}

// ListModels lists available models.
func (c *Client) ListModels(ctx context.Context) (*models.ModelsResponse, error) {
	endpoint := c.BaseURL + "/v1/models"
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp)
	}

	// Parse response
	var modelsResp models.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &modelsResp, nil
}

// doRequest performs an HTTP request with proper headers and error handling.
func (c *Client) doRequest(ctx context.Context, method string, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	// Make request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// handleErrorResponse parses and returns an error from an HTTP error response.
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response: %w", resp.StatusCode, err)
	}

	var errResp models.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("API error: %w", errResp.Error)
}

// SSEReader reads Server-Sent Events from a stream.
type SSEReader struct {
	reader io.Reader
	buffer []byte
}

// NewSSEReader creates a new SSE reader.
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{
		reader: r,
		buffer: make([]byte, 0, 4096),
	}
}

// ReadEvent reads the next SSE event.
// Returns the event data or an error.
func (r *SSEReader) ReadEvent() (string, error) {
	buf := make([]byte, 4096)
	for {
		n, err := r.reader.Read(buf)
		if err != nil {
			return "", err
		}

		r.buffer = append(r.buffer, buf[:n]...)

		// Look for complete event (ends with \n\n)
		for {
			idx := bytes.Index(r.buffer, []byte("\n\n"))
			if idx == -1 {
				break
			}

			// Extract event
			event := r.buffer[:idx]
			r.buffer = r.buffer[idx+2:]

			// Parse event
			if bytes.HasPrefix(event, []byte("data: ")) {
				data := string(bytes.TrimPrefix(event, []byte("data: ")))
				return data, nil
			}
		}
	}
}

// Close closes the underlying reader if it implements io.Closer.
func (r *SSEReader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}