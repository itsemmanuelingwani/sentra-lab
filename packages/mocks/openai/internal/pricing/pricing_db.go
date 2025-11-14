// Package pricing provides cost calculation for API usage.
// This file implements the pricing database with current OpenAI pricing (Nov 2025).
package pricing

import (
	"fmt"
	"sync"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// PricingDB manages model pricing information.
// Prices are in USD per 1 million tokens.
type PricingDB struct {
	// prices maps model IDs to pricing information
	prices map[string]ModelPricing

	// mu protects the prices map for concurrent access
	mu sync.RWMutex

	// currency is the pricing currency (always USD)
	currency string

	// lastUpdated tracks when pricing was last updated
	lastUpdated string
}

// ModelPricing contains pricing information for a specific model.
type ModelPricing struct {
	// ModelID is the model identifier
	ModelID string

	// InputPer1M is the cost per 1M input tokens in USD
	InputPer1M float64

	// OutputPer1M is the cost per 1M output tokens in USD
	OutputPer1M float64

	// CachedInputPer1M is the cost per 1M cached input tokens (if supported)
	CachedInputPer1M float64

	// ImagePricing contains pricing for image generation (if applicable)
	ImagePricing *ImagePricing

	// SupportsCachedInput indicates if the model supports cached input pricing
	SupportsCachedInput bool
}

// ImagePricing contains pricing for image generation.
type ImagePricing struct {
	// Standard is the price per image for standard quality
	Standard map[string]float64 // size -> price

	// HD is the price per image for HD quality
	HD map[string]float64 // size -> price
}

// NewPricingDB creates a new pricing database with default pricing.
func NewPricingDB() *PricingDB {
	db := &PricingDB{
		prices:      make(map[string]ModelPricing),
		currency:    "USD",
		lastUpdated: "2025-11-12",
	}

	// Load default pricing from model configs
	db.loadDefaultPricing()

	return db
}

// loadDefaultPricing loads pricing from model configurations.
func (db *PricingDB) loadDefaultPricing() {
	// Chat models
	db.prices["gpt-4o"] = ModelPricing{
		ModelID:             "gpt-4o",
		InputPer1M:          2.50,
		OutputPer1M:         10.00,
		CachedInputPer1M:    1.25,
		SupportsCachedInput: true,
	}

	db.prices["gpt-4o-mini"] = ModelPricing{
		ModelID:             "gpt-4o-mini",
		InputPer1M:          0.15,
		OutputPer1M:         0.60,
		CachedInputPer1M:    0.075,
		SupportsCachedInput: true,
	}

	db.prices["gpt-4-turbo"] = ModelPricing{
		ModelID:             "gpt-4-turbo",
		InputPer1M:          10.00,
		OutputPer1M:         30.00,
		CachedInputPer1M:    5.00,
		SupportsCachedInput: true,
	}

	db.prices["gpt-4"] = ModelPricing{
		ModelID:             "gpt-4",
		InputPer1M:          30.00,
		OutputPer1M:         60.00,
		SupportsCachedInput: false,
	}

	db.prices["gpt-3.5-turbo"] = ModelPricing{
		ModelID:             "gpt-3.5-turbo",
		InputPer1M:          0.50,
		OutputPer1M:         1.50,
		SupportsCachedInput: false,
	}

	db.prices["gpt-3.5-turbo-16k"] = ModelPricing{
		ModelID:     "gpt-3.5-turbo-16k",
		InputPer1M:  3.00,
		OutputPer1M: 4.00,
	}

	// Embedding models
	db.prices["text-embedding-3-small"] = ModelPricing{
		ModelID:     "text-embedding-3-small",
		InputPer1M:  0.02,
		OutputPer1M: 0, // No output for embeddings
	}

	db.prices["text-embedding-3-large"] = ModelPricing{
		ModelID:     "text-embedding-3-large",
		InputPer1M:  0.13,
		OutputPer1M: 0,
	}

	db.prices["text-embedding-ada-002"] = ModelPricing{
		ModelID:     "text-embedding-ada-002",
		InputPer1M:  0.10,
		OutputPer1M: 0,
	}

	// Image generation models
	db.prices["dall-e-3"] = ModelPricing{
		ModelID:     "dall-e-3",
		InputPer1M:  0, // Images are priced per image, not tokens
		OutputPer1M: 0,
		ImagePricing: &ImagePricing{
			Standard: map[string]float64{
				"1024x1024": 0.04,
				"1024x1792": 0.08,
				"1792x1024": 0.08,
			},
			HD: map[string]float64{
				"1024x1024": 0.08,
				"1024x1792": 0.12,
				"1792x1024": 0.12,
			},
		},
	}

	db.prices["dall-e-2"] = ModelPricing{
		ModelID:     "dall-e-2",
		InputPer1M:  0,
		OutputPer1M: 0,
		ImagePricing: &ImagePricing{
			Standard: map[string]float64{
				"1024x1024": 0.02,
				"512x512":   0.018,
				"256x256":   0.016,
			},
		},
	}
}

// GetPricing retrieves pricing for a model.
func (db *PricingDB) GetPricing(modelID string) (ModelPricing, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	pricing, ok := db.prices[modelID]
	if !ok {
		return ModelPricing{}, fmt.Errorf("pricing not found for model: %s", modelID)
	}

	return pricing, nil
}

// SetPricing updates pricing for a model.
func (db *PricingDB) SetPricing(modelID string, pricing ModelPricing) {
	db.mu.Lock()
	defer db.mu.Unlock()

	pricing.ModelID = modelID
	db.prices[modelID] = pricing
}

// GetAllPricing returns pricing for all models.
func (db *PricingDB) GetAllPricing() map[string]ModelPricing {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Create a copy to avoid race conditions
	pricingCopy := make(map[string]ModelPricing, len(db.prices))
	for k, v := range db.prices {
		pricingCopy[k] = v
	}

	return pricingCopy
}

// UpdatePricing updates pricing from a configuration map.
func (db *PricingDB) UpdatePricing(pricingMap map[string]ModelPricing) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for modelID, pricing := range pricingMap {
		pricing.ModelID = modelID
		db.prices[modelID] = pricing
	}
}

// LoadFromConfig loads pricing from configuration.
func (db *PricingDB) LoadFromConfig(configPricing map[string]struct {
	InputPer1M       float64 `yaml:"input_per_1m"`
	OutputPer1M      float64 `yaml:"output_per_1m"`
	CachedInputPer1M float64 `yaml:"cached_input_per_1m"`
}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for modelID, config := range configPricing {
		db.prices[modelID] = ModelPricing{
			ModelID:             modelID,
			InputPer1M:          config.InputPer1M,
			OutputPer1M:         config.OutputPer1M,
			CachedInputPer1M:    config.CachedInputPer1M,
			SupportsCachedInput: config.CachedInputPer1M > 0,
		}
	}

	return nil
}

// GetCurrency returns the pricing currency.
func (db *PricingDB) GetCurrency() string {
	return db.currency
}

// GetLastUpdated returns when pricing was last updated.
func (db *PricingDB) GetLastUpdated() string {
	return db.lastUpdated
}

// ValidateModel checks if a model exists in the pricing database.
func (db *PricingDB) ValidateModel(modelID string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if _, ok := db.prices[modelID]; !ok {
		return fmt.Errorf("unknown model: %s", modelID)
	}

	return nil
}

// GetModelsByType returns models of a specific type (chat, embedding, image).
func (db *PricingDB) GetModelsByType(modelType string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var modelIDs []string

	for modelID, pricing := range db.prices {
		switch modelType {
		case "chat":
			if pricing.OutputPer1M > 0 && pricing.ImagePricing == nil {
				modelIDs = append(modelIDs, modelID)
			}
		case "embedding":
			if pricing.OutputPer1M == 0 && pricing.ImagePricing == nil {
				modelIDs = append(modelIDs, modelID)
			}
		case "image":
			if pricing.ImagePricing != nil {
				modelIDs = append(modelIDs, modelID)
			}
		}
	}

	return modelIDs
}

// ComparePricing compares pricing between two models.
func (db *PricingDB) ComparePricing(model1, model2 string, inputTokens, outputTokens int) (PricingComparison, error) {
	pricing1, err := db.GetPricing(model1)
	if err != nil {
		return PricingComparison{}, err
	}

	pricing2, err := db.GetPricing(model2)
	if err != nil {
		return PricingComparison{}, err
	}

	cost1 := calculateCost(pricing1, inputTokens, outputTokens, false)
	cost2 := calculateCost(pricing2, inputTokens, outputTokens, false)

	savings := cost1 - cost2
	savingsPercent := 0.0
	if cost1 > 0 {
		savingsPercent = (savings / cost1) * 100
	}

	return PricingComparison{
		Model1:         model1,
		Model2:         model2,
		Cost1:          cost1,
		Cost2:          cost2,
		Savings:        savings,
		SavingsPercent: savingsPercent,
		Cheaper:        model2,
	}, nil
}

// PricingComparison represents a comparison between two models.
type PricingComparison struct {
	Model1         string
	Model2         string
	Cost1          float64
	Cost2          float64
	Savings        float64
	SavingsPercent float64
	Cheaper        string
}

// calculateCost is a helper function to calculate cost from pricing.
func calculateCost(pricing ModelPricing, inputTokens, outputTokens int, useCachedInput bool) float64 {
	inputCost := float64(inputTokens) * pricing.InputPer1M / 1_000_000

	if useCachedInput && pricing.SupportsCachedInput {
		inputCost = float64(inputTokens) * pricing.CachedInputPer1M / 1_000_000
	}

	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000

	return inputCost + outputCost
}