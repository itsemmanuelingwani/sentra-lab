// Package pricing provides cost calculation.
// This file implements the cost calculator for API requests.
package pricing

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Calculator calculates costs for API usage.
type Calculator struct {
	// db is the pricing database
	db *PricingDB

	// stats tracks cumulative costs
	totalCost       atomic.Uint64 // Stored as cents (multiply by 100)
	totalRequests   atomic.Int64
	totalInputTokens atomic.Int64
	totalOutputTokens atomic.Int64
}

// NewCalculator creates a new cost calculator.
func NewCalculator(db *PricingDB) *Calculator {
	return &Calculator{
		db: db,
	}
}

// Calculate calculates the cost for a request.
func (c *Calculator) Calculate(ctx context.Context, modelID string, inputTokens, outputTokens int) (Cost, error) {
	// Get pricing
	pricing, err := c.db.GetPricing(modelID)
	if err != nil {
		return Cost{}, err
	}

	// Calculate input cost
	inputCost := float64(inputTokens) * pricing.InputPer1M / 1_000_000

	// Calculate output cost
	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000

	// Total cost
	totalCost := inputCost + outputCost

	// Update statistics
	c.addToTotal(totalCost)
	c.totalRequests.Add(1)
	c.totalInputTokens.Add(int64(inputTokens))
	c.totalOutputTokens.Add(int64(outputTokens))

	return Cost{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Currency:     c.db.GetCurrency(),
		Model:        modelID,
	}, nil
}

// CalculateWithCachedInput calculates cost with cached input pricing.
func (c *Calculator) CalculateWithCachedInput(ctx context.Context, modelID string, cachedInputTokens, newInputTokens, outputTokens int) (Cost, error) {
	// Get pricing
	pricing, err := c.db.GetPricing(modelID)
	if err != nil {
		return Cost{}, err
	}

	if !pricing.SupportsCachedInput {
		// Fall back to regular calculation
		return c.Calculate(ctx, modelID, cachedInputTokens+newInputTokens, outputTokens)
	}

	// Calculate cached input cost
	cachedInputCost := float64(cachedInputTokens) * pricing.CachedInputPer1M / 1_000_000

	// Calculate new input cost
	newInputCost := float64(newInputTokens) * pricing.InputPer1M / 1_000_000

	// Calculate output cost
	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000

	// Total cost
	totalInputCost := cachedInputCost + newInputCost
	totalCost := totalInputCost + outputCost

	// Update statistics
	c.addToTotal(totalCost)
	c.totalRequests.Add(1)
	c.totalInputTokens.Add(int64(cachedInputTokens + newInputTokens))
	c.totalOutputTokens.Add(int64(outputTokens))

	return Cost{
		InputTokens:        cachedInputTokens + newInputTokens,
		OutputTokens:       outputTokens,
		TotalTokens:        cachedInputTokens + newInputTokens + outputTokens,
		InputCost:          totalInputCost,
		OutputCost:         outputCost,
		TotalCost:          totalCost,
		Currency:           c.db.GetCurrency(),
		Model:              modelID,
		CachedInputTokens:  cachedInputTokens,
		CachedInputCost:    cachedInputCost,
		SupportsCachedInput: true,
	}, nil
}

// CalculateImageCost calculates cost for image generation.
func (c *Calculator) CalculateImageCost(ctx context.Context, modelID string, size string, quality string, numImages int) (ImageCost, error) {
	// Get pricing
	pricing, err := c.db.GetPricing(modelID)
	if err != nil {
		return ImageCost{}, err
	}

	if pricing.ImagePricing == nil {
		return ImageCost{}, fmt.Errorf("model %s does not support image generation", modelID)
	}

	// Get price per image
	var pricePerImage float64
	if quality == "hd" && pricing.ImagePricing.HD != nil {
		pricePerImage = pricing.ImagePricing.HD[size]
	} else {
		pricePerImage = pricing.ImagePricing.Standard[size]
	}

	if pricePerImage == 0 {
		return ImageCost{}, fmt.Errorf("pricing not found for size %s and quality %s", size, quality)
	}

	// Calculate total cost
	totalCost := pricePerImage * float64(numImages)

	// Update statistics
	c.addToTotal(totalCost)
	c.totalRequests.Add(1)

	return ImageCost{
		Model:         modelID,
		Size:          size,
		Quality:       quality,
		NumImages:     numImages,
		PricePerImage: pricePerImage,
		TotalCost:     totalCost,
		Currency:      c.db.GetCurrency(),
	}, nil
}

// EstimateCost estimates cost without recording statistics.
func (c *Calculator) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (Cost, error) {
	// Get pricing
	pricing, err := c.db.GetPricing(modelID)
	if err != nil {
		return Cost{}, err
	}

	// Calculate costs (without updating stats)
	inputCost := float64(inputTokens) * pricing.InputPer1M / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000
	totalCost := inputCost + outputCost

	return Cost{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Currency:     c.db.GetCurrency(),
		Model:        modelID,
	}, nil
}

// GetTotalCost returns the cumulative cost across all requests.
func (c *Calculator) GetTotalCost() float64 {
	cents := c.totalCost.Load()
	return float64(cents) / 100.0
}

// GetStats returns calculator statistics.
func (c *Calculator) GetStats(ctx context.Context) CalculatorStats {
	return CalculatorStats{
		TotalCost:         c.GetTotalCost(),
		TotalRequests:     c.totalRequests.Load(),
		TotalInputTokens:  c.totalInputTokens.Load(),
		TotalOutputTokens: c.totalOutputTokens.Load(),
		TotalTokens:       c.totalInputTokens.Load() + c.totalOutputTokens.Load(),
		Currency:          c.db.GetCurrency(),
	}
}

// ResetStats resets all statistics.
func (c *Calculator) ResetStats(ctx context.Context) {
	c.totalCost.Store(0)
	c.totalRequests.Store(0)
	c.totalInputTokens.Store(0)
	c.totalOutputTokens.Store(0)
}

// addToTotal adds cost to the total (stored as cents to avoid float precision issues).
func (c *Calculator) addToTotal(cost float64) {
	cents := uint64(cost * 100)
	c.totalCost.Add(cents)
}

// Cost represents the cost breakdown for a request.
type Cost struct {
	// Input tokens
	InputTokens int
	InputCost   float64

	// Output tokens
	OutputTokens int
	OutputCost   float64

	// Total
	TotalTokens int
	TotalCost   float64

	// Currency (always USD)
	Currency string

	// Model used
	Model string

	// Cached input (optional)
	CachedInputTokens   int
	CachedInputCost     float64
	SupportsCachedInput bool
}

// FormatCost formats the cost as a string.
func (c Cost) FormatCost() string {
	return fmt.Sprintf("$%.6f", c.TotalCost)
}

// FormatBreakdown formats the cost breakdown as a string.
func (c Cost) FormatBreakdown() string {
	if c.SupportsCachedInput && c.CachedInputTokens > 0 {
		return fmt.Sprintf(
			"Input: %d tokens ($%.6f, %d cached @ $%.6f), Output: %d tokens ($%.6f), Total: $%.6f",
			c.InputTokens,
			c.InputCost,
			c.CachedInputTokens,
			c.CachedInputCost,
			c.OutputTokens,
			c.OutputCost,
			c.TotalCost,
		)
	}

	return fmt.Sprintf(
		"Input: %d tokens ($%.6f), Output: %d tokens ($%.6f), Total: $%.6f",
		c.InputTokens,
		c.InputCost,
		c.OutputTokens,
		c.OutputCost,
		c.TotalCost,
	)
}

// ImageCost represents the cost for image generation.
type ImageCost struct {
	Model         string
	Size          string
	Quality       string
	NumImages     int
	PricePerImage float64
	TotalCost     float64
	Currency      string
}

// FormatCost formats the image cost as a string.
func (ic ImageCost) FormatCost() string {
	return fmt.Sprintf("$%.6f", ic.TotalCost)
}

// FormatBreakdown formats the image cost breakdown.
func (ic ImageCost) FormatBreakdown() string {
	return fmt.Sprintf(
		"%d image(s) @ $%.4f each (%s, %s) = $%.6f",
		ic.NumImages,
		ic.PricePerImage,
		ic.Size,
		ic.Quality,
		ic.TotalCost,
	)
}

// CalculatorStats contains statistics about cost calculations.
type CalculatorStats struct {
	TotalCost         float64
	TotalRequests     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalTokens       int64
	Currency          string
}

// AverageCostPerRequest calculates the average cost per request.
func (cs CalculatorStats) AverageCostPerRequest() float64 {
	if cs.TotalRequests == 0 {
		return 0
	}
	return cs.TotalCost / float64(cs.TotalRequests)
}

// AverageCostPerToken calculates the average cost per token.
func (cs CalculatorStats) AverageCostPerToken() float64 {
	if cs.TotalTokens == 0 {
		return 0
	}
	return cs.TotalCost / float64(cs.TotalTokens)
}

// FormatStats formats statistics as a string.
func (cs CalculatorStats) FormatStats() string {
	return fmt.Sprintf(
		"Total: $%.6f (%d requests, %d tokens, avg $%.6f/request)",
		cs.TotalCost,
		cs.TotalRequests,
		cs.TotalTokens,
		cs.AverageCostPerRequest(),
	)
}