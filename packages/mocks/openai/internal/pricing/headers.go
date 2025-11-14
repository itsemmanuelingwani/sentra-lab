// Package pricing provides cost calculation.
// This file implements HTTP headers for cost information in API responses.
package pricing

import (
	"fmt"
	"net/http"
)

// AddCostHeaders adds cost-related headers to HTTP response.
// These are custom headers prefixed with X-Sentra- for the mock server.
func AddCostHeaders(w http.ResponseWriter, cost Cost) {
	// Total cost
	w.Header().Set("X-Sentra-Cost-Total", fmt.Sprintf("%.6f", cost.TotalCost))
	w.Header().Set("X-Sentra-Cost-Currency", cost.Currency)

	// Token breakdown
	w.Header().Set("X-Sentra-Tokens-Input", fmt.Sprintf("%d", cost.InputTokens))
	w.Header().Set("X-Sentra-Tokens-Output", fmt.Sprintf("%d", cost.OutputTokens))
	w.Header().Set("X-Sentra-Tokens-Total", fmt.Sprintf("%d", cost.TotalTokens))

	// Cost breakdown
	w.Header().Set("X-Sentra-Cost-Input", fmt.Sprintf("%.6f", cost.InputCost))
	w.Header().Set("X-Sentra-Cost-Output", fmt.Sprintf("%.6f", cost.OutputCost))

	// Model used
	w.Header().Set("X-Sentra-Model", cost.Model)

	// Cached input (if applicable)
	if cost.SupportsCachedInput && cost.CachedInputTokens > 0 {
		w.Header().Set("X-Sentra-Tokens-Cached-Input", fmt.Sprintf("%d", cost.CachedInputTokens))
		w.Header().Set("X-Sentra-Cost-Cached-Input", fmt.Sprintf("%.6f", cost.CachedInputCost))
		w.Header().Set("X-Sentra-Cache-Enabled", "true")
	}
}

// AddImageCostHeaders adds cost headers for image generation.
func AddImageCostHeaders(w http.ResponseWriter, cost ImageCost) {
	w.Header().Set("X-Sentra-Cost-Total", fmt.Sprintf("%.6f", cost.TotalCost))
	w.Header().Set("X-Sentra-Cost-Currency", cost.Currency)
	w.Header().Set("X-Sentra-Cost-Per-Image", fmt.Sprintf("%.4f", cost.PricePerImage))
	w.Header().Set("X-Sentra-Images-Count", fmt.Sprintf("%d", cost.NumImages))
	w.Header().Set("X-Sentra-Image-Size", cost.Size)
	w.Header().Set("X-Sentra-Image-Quality", cost.Quality)
	w.Header().Set("X-Sentra-Model", cost.Model)
}

// AddEstimatedCostHeaders adds estimated cost headers (for rate limiting).
func AddEstimatedCostHeaders(w http.ResponseWriter, estimatedCost float64, model string) {
	w.Header().Set("X-Sentra-Cost-Estimated", fmt.Sprintf("%.6f", estimatedCost))
	w.Header().Set("X-Sentra-Cost-Currency", "USD")
	w.Header().Set("X-Sentra-Model", model)
	w.Header().Set("X-Sentra-Cost-Type", "estimated")
}

// AddUsageHeaders adds cumulative usage headers.
func AddUsageHeaders(w http.ResponseWriter, stats CalculatorStats) {
	w.Header().Set("X-Sentra-Usage-Total-Cost", fmt.Sprintf("%.6f", stats.TotalCost))
	w.Header().Set("X-Sentra-Usage-Total-Requests", fmt.Sprintf("%d", stats.TotalRequests))
	w.Header().Set("X-Sentra-Usage-Total-Tokens", fmt.Sprintf("%d", stats.TotalTokens))
	w.Header().Set("X-Sentra-Usage-Average-Cost", fmt.Sprintf("%.6f", stats.AverageCostPerRequest()))
}

// AddUserUsageHeaders adds per-user usage headers.
func AddUserUsageHeaders(w http.ResponseWriter, usage *UserUsage) {
	w.Header().Set("X-Sentra-User-Total-Cost", fmt.Sprintf("%.6f", usage.TotalCost))
	w.Header().Set("X-Sentra-User-Total-Requests", fmt.Sprintf("%d", usage.TotalRequests))
	w.Header().Set("X-Sentra-User-Total-Tokens", fmt.Sprintf("%d", usage.TotalInputTokens+usage.TotalOutputTokens))
	w.Header().Set("X-Sentra-User-First-Request", usage.FirstRequest.Format("2006-01-02T15:04:05Z"))
	w.Header().Set("X-Sentra-User-Last-Request", usage.LastRequest.Format("2006-01-02T15:04:05Z"))
}

// AddModelUsageHeaders adds per-model usage headers.
func AddModelUsageHeaders(w http.ResponseWriter, usage *ModelUsage) {
	w.Header().Set("X-Sentra-Model-Total-Cost", fmt.Sprintf("%.6f", usage.TotalCost))
	w.Header().Set("X-Sentra-Model-Total-Requests", fmt.Sprintf("%d", usage.TotalRequests))
	w.Header().Set("X-Sentra-Model-Total-Tokens", fmt.Sprintf("%d", usage.TotalInputTokens+usage.TotalOutputTokens))
	w.Header().Set("X-Sentra-Model-Average-Cost", fmt.Sprintf("%.6f", usage.AverageCost))
}

// CostHeadersConfig controls which cost headers to include.
type CostHeadersConfig struct {
	// IncludeCost includes cost breakdown headers
	IncludeCost bool

	// IncludeUsage includes cumulative usage headers
	IncludeUsage bool

	// IncludeUserUsage includes per-user usage headers
	IncludeUserUsage bool

	// IncludeModelUsage includes per-model usage headers
	IncludeModelUsage bool
}

// DefaultCostHeadersConfig returns default configuration.
func DefaultCostHeadersConfig() CostHeadersConfig {
	return CostHeadersConfig{
		IncludeCost:       true,
		IncludeUsage:      false, // Disabled by default for privacy
		IncludeUserUsage:  false,
		IncludeModelUsage: false,
	}
}

// AddAllCostHeaders adds all configured cost headers.
func AddAllCostHeaders(w http.ResponseWriter, cost Cost, stats *CalculatorStats, userUsage *UserUsage, modelUsage *ModelUsage, config CostHeadersConfig) {
	if config.IncludeCost {
		AddCostHeaders(w, cost)
	}

	if config.IncludeUsage && stats != nil {
		AddUsageHeaders(w, *stats)
	}

	if config.IncludeUserUsage && userUsage != nil {
		AddUserUsageHeaders(w, userUsage)
	}

	if config.IncludeModelUsage && modelUsage != nil {
		AddModelUsageHeaders(w, modelUsage)
	}
}

// ParseCostHeaders parses cost headers from an HTTP response.
// This is useful for clients reading cost information.
func ParseCostHeaders(headers http.Header) (*Cost, error) {
	cost := &Cost{
		Currency: headers.Get("X-Sentra-Cost-Currency"),
		Model:    headers.Get("X-Sentra-Model"),
	}

	// Parse total cost
	if totalCost := headers.Get("X-Sentra-Cost-Total"); totalCost != "" {
		fmt.Sscanf(totalCost, "%f", &cost.TotalCost)
	}

	// Parse tokens
	fmt.Sscanf(headers.Get("X-Sentra-Tokens-Input"), "%d", &cost.InputTokens)
	fmt.Sscanf(headers.Get("X-Sentra-Tokens-Output"), "%d", &cost.OutputTokens)
	fmt.Sscanf(headers.Get("X-Sentra-Tokens-Total"), "%d", &cost.TotalTokens)

	// Parse cost breakdown
	fmt.Sscanf(headers.Get("X-Sentra-Cost-Input"), "%f", &cost.InputCost)
	fmt.Sscanf(headers.Get("X-Sentra-Cost-Output"), "%f", &cost.OutputCost)

	// Parse cached input (if present)
	if headers.Get("X-Sentra-Cache-Enabled") == "true" {
		cost.SupportsCachedInput = true
		fmt.Sscanf(headers.Get("X-Sentra-Tokens-Cached-Input"), "%d", &cost.CachedInputTokens)
		fmt.Sscanf(headers.Get("X-Sentra-Cost-Cached-Input"), "%f", &cost.CachedInputCost)
	}

	return cost, nil
}

// GetCostSummary returns a human-readable cost summary from headers.
func GetCostSummary(headers http.Header) string {
	cost, err := ParseCostHeaders(headers)
	if err != nil || cost == nil {
		return "Cost information not available"
	}

	return fmt.Sprintf(
		"Cost: %s (%d input + %d output tokens)",
		cost.FormatCost(),
		cost.InputTokens,
		cost.OutputTokens,
	)
}