// Package ratelimit provides rate limiting functionality matching OpenAI's production behavior.
// This file defines rate limit tiers and model-specific limits.
package ratelimit

import (
	"fmt"
	"sync"
)

// Tier represents a rate limit tier with specific limits per model.
// OpenAI uses tier-based rate limiting (Tier 1-5) based on usage history.
type Tier struct {
	// Name is the tier identifier (e.g., "tier1", "tier2")
	Name string

	// Description describes the tier requirements
	Description string

	// ModelLimits maps model IDs to their rate limits
	ModelLimits map[string]ModelLimit
}

// ModelLimit defines rate limits for a specific model.
type ModelLimit struct {
	// ModelID is the model identifier
	ModelID string

	// RPM is Requests Per Minute limit
	RPM int

	// TPM is Tokens Per Minute limit
	TPM int

	// RPD is Requests Per Day limit (optional, 0 = unlimited)
	RPD int

	// TPD is Tokens Per Day limit (optional, 0 = unlimited)
	TPD int
}

// TierRegistry manages rate limit tiers.
type TierRegistry struct {
	// tiers maps tier names to tier definitions
	tiers map[string]Tier

	// defaultTier is the tier used when API key has no specific tier
	defaultTier string

	// mu protects concurrent access
	mu sync.RWMutex
}

// NewTierRegistry creates a new tier registry with default OpenAI tiers.
func NewTierRegistry(defaultTier string) *TierRegistry {
	registry := &TierRegistry{
		tiers:       make(map[string]Tier),
		defaultTier: defaultTier,
	}

	// Load default OpenAI tiers (Nov 2025)
	registry.loadDefaultTiers()

	return registry
}

// loadDefaultTiers loads production OpenAI rate limit tiers.
// These match OpenAI's actual tier structure as of Nov 2025.
func (r *TierRegistry) loadDefaultTiers() {
	// Free Trial - Very limited
	r.tiers["free"] = Tier{
		Name:        "free",
		Description: "Free trial tier (3 months or $5 spend)",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     3,
				TPM:     40000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     3,
				TPM:     40000,
			},
			"gpt-3.5-turbo": {
				ModelID: "gpt-3.5-turbo",
				RPM:     3,
				TPM:     40000,
			},
		},
	}

	// Tier 1 - Pay-as-you-go
	r.tiers["tier1"] = Tier{
		Name:        "tier1",
		Description: "Tier 1: Pay-as-you-go",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     500,
				TPM:     800000,
				RPD:     10000,
			},
			"gpt-4o-mini": {
				ModelID: "gpt-4o-mini",
				RPM:     30000,
				TPM:     200000000,
			},
			"gpt-4-turbo": {
				ModelID: "gpt-4-turbo",
				RPM:     500,
				TPM:     300000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     500,
				TPM:     300000,
				RPD:     10000,
			},
			"gpt-3.5-turbo": {
				ModelID: "gpt-3.5-turbo",
				RPM:     3500,
				TPM:     200000,
			},
			"text-embedding-3-small": {
				ModelID: "text-embedding-3-small",
				RPM:     3000,
				TPM:     1000000,
			},
			"text-embedding-3-large": {
				ModelID: "text-embedding-3-large",
				RPM:     3000,
				TPM:     1000000,
			},
			"text-embedding-ada-002": {
				ModelID: "text-embedding-ada-002",
				RPM:     3000,
				TPM:     1000000,
			},
		},
	}

	// Tier 2 - $50+ spend
	r.tiers["tier2"] = Tier{
		Name:        "tier2",
		Description: "Tier 2: $50+ spend",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     5000,
				TPM:     2000000,
			},
			"gpt-4o-mini": {
				ModelID: "gpt-4o-mini",
				RPM:     30000,
				TPM:     200000000,
			},
			"gpt-4-turbo": {
				ModelID: "gpt-4-turbo",
				RPM:     5000,
				TPM:     1000000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     5000,
				TPM:     1000000,
			},
			"gpt-3.5-turbo": {
				ModelID: "gpt-3.5-turbo",
				RPM:     10000,
				TPM:     2000000,
			},
		},
	}

	// Tier 3 - $100+ spend
	r.tiers["tier3"] = Tier{
		Name:        "tier3",
		Description: "Tier 3: $100+ spend",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     10000,
				TPM:     4000000,
			},
			"gpt-4o-mini": {
				ModelID: "gpt-4o-mini",
				RPM:     30000,
				TPM:     200000000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     10000,
				TPM:     2000000,
			},
			"gpt-3.5-turbo": {
				ModelID: "gpt-3.5-turbo",
				RPM:     10000,
				TPM:     2000000,
			},
		},
	}

	// Tier 4 - $250+ spend
	r.tiers["tier4"] = Tier{
		Name:        "tier4",
		Description: "Tier 4: $250+ spend",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     30000,
				TPM:     10000000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     10000,
				TPM:     4000000,
			},
		},
	}

	// Tier 5 - $1000+ spend
	r.tiers["tier5"] = Tier{
		Name:        "tier5",
		Description: "Tier 5: $1000+ spend",
		ModelLimits: map[string]ModelLimit{
			"gpt-4o": {
				ModelID: "gpt-4o",
				RPM:     30000,
				TPM:     10000000,
			},
			"gpt-4": {
				ModelID: "gpt-4",
				RPM:     10000,
				TPM:     10000000,
			},
		},
	}
}

// GetTier retrieves a tier by name.
func (r *TierRegistry) GetTier(tierName string) (Tier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tier, ok := r.tiers[tierName]
	if !ok {
		return Tier{}, fmt.Errorf("tier not found: %s", tierName)
	}

	return tier, nil
}

// GetTierOrDefault retrieves a tier or returns the default tier.
func (r *TierRegistry) GetTierOrDefault(tierName string) Tier {
	tier, err := r.GetTier(tierName)
	if err != nil {
		tier, _ = r.GetTier(r.defaultTier)
	}
	return tier
}

// GetModelLimit retrieves rate limits for a specific model in a tier.
func (r *TierRegistry) GetModelLimit(tierName, modelID string) (ModelLimit, error) {
	tier, err := r.GetTier(tierName)
	if err != nil {
		return ModelLimit{}, err
	}

	limit, ok := tier.ModelLimits[modelID]
	if !ok {
		return ModelLimit{}, fmt.Errorf("no rate limits defined for model %s in tier %s", modelID, tierName)
	}

	return limit, nil
}

// GetModelLimitOrDefault retrieves model limit or returns default tier's limit.
func (r *TierRegistry) GetModelLimitOrDefault(tierName, modelID string) ModelLimit {
	limit, err := r.GetModelLimit(tierName, modelID)
	if err != nil {
		// Try default tier
		defaultTier, err := r.GetTier(r.defaultTier)
		if err != nil {
			// Return conservative defaults
			return ModelLimit{
				ModelID: modelID,
				RPM:     60,    // 1 per second
				TPM:     100000, // Conservative limit
			}
		}

		limit, ok := defaultTier.ModelLimits[modelID]
		if !ok {
			// Return conservative defaults
			return ModelLimit{
				ModelID: modelID,
				RPM:     60,
				TPM:     100000,
			}
		}

		return limit
	}

	return limit
}

// SetTier adds or updates a tier.
func (r *TierRegistry) SetTier(tier Tier) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tiers[tier.Name] = tier
}

// DeleteTier removes a tier.
func (r *TierRegistry) DeleteTier(tierName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tierName == r.defaultTier {
		return fmt.Errorf("cannot delete default tier")
	}

	delete(r.tiers, tierName)
	return nil
}

// ListTiers returns all available tier names.
func (r *TierRegistry) ListTiers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tiers := make([]string, 0, len(r.tiers))
	for name := range r.tiers {
		tiers = append(tiers, name)
	}

	return tiers
}

// GetAllTiers returns all tiers.
func (r *TierRegistry) GetAllTiers() map[string]Tier {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	tiers := make(map[string]Tier, len(r.tiers))
	for k, v := range r.tiers {
		tiers[k] = v
	}

	return tiers
}

// SetDefaultTier sets the default tier.
func (r *TierRegistry) SetDefaultTier(tierName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tiers[tierName]; !ok {
		return fmt.Errorf("tier does not exist: %s", tierName)
	}

	r.defaultTier = tierName
	return nil
}

// GetDefaultTier returns the default tier name.
func (r *TierRegistry) GetDefaultTier() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultTier
}

// Validate validates a tier configuration.
func (t *Tier) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tier name is required")
	}

	if len(t.ModelLimits) == 0 {
		return fmt.Errorf("tier must have at least one model limit")
	}

	for modelID, limit := range t.ModelLimits {
		if limit.ModelID != modelID {
			return fmt.Errorf("model ID mismatch in limit: %s != %s", limit.ModelID, modelID)
		}

		if err := limit.Validate(); err != nil {
			return fmt.Errorf("invalid limit for model %s: %w", modelID, err)
		}
	}

	return nil
}

// Validate validates a model limit.
func (m *ModelLimit) Validate() error {
	if m.ModelID == "" {
		return fmt.Errorf("model ID is required")
	}

	if m.RPM <= 0 {
		return fmt.Errorf("RPM must be positive")
	}

	if m.TPM <= 0 {
		return fmt.Errorf("TPM must be positive")
	}

	if m.RPD < 0 {
		return fmt.Errorf("RPD cannot be negative")
	}

	if m.TPD < 0 {
		return fmt.Errorf("TPD cannot be negative")
	}

	return nil
}

// String returns a string representation of the limit.
func (m *ModelLimit) String() string {
	return fmt.Sprintf("%s: %d RPM, %d TPM", m.ModelID, m.RPM, m.TPM)
}

// CompareTiers compares rate limits between two tiers.
func CompareTiers(tier1, tier2 Tier) TierComparison {
	comparison := TierComparison{
		Tier1: tier1.Name,
		Tier2: tier2.Name,
		Models: make(map[string]ModelComparison),
	}

	// Compare models present in both tiers
	for modelID, limit1 := range tier1.ModelLimits {
		if limit2, ok := tier2.ModelLimits[modelID]; ok {
			comparison.Models[modelID] = ModelComparison{
				ModelID:        modelID,
				Tier1RPM:       limit1.RPM,
				Tier2RPM:       limit2.RPM,
				RPMIncrease:    limit2.RPM - limit1.RPM,
				Tier1TPM:       limit1.TPM,
				Tier2TPM:       limit2.TPM,
				TPMIncrease:    limit2.TPM - limit1.TPM,
			}
		}
	}

	return comparison
}

// TierComparison represents a comparison between two tiers.
type TierComparison struct {
	Tier1  string
	Tier2  string
	Models map[string]ModelComparison
}

// ModelComparison represents a comparison for a specific model.
type ModelComparison struct {
	ModelID     string
	Tier1RPM    int
	Tier2RPM    int
	RPMIncrease int
	Tier1TPM    int
	Tier2TPM    int
	TPMIncrease int
}