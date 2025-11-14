// Package pricing provides cost calculation.
// This file implements usage tracking and per-user/per-model cost tracking.
package pricing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/store"
)

// Tracker tracks usage and costs per user/API key and per model.
type Tracker struct {
	// calculator is the cost calculator
	calculator *Calculator

	// storage for persistent tracking
	storage store.Storage

	// mu protects in-memory aggregations
	mu sync.RWMutex

	// userCosts tracks costs per user/API key
	userCosts map[string]*UserUsage

	// modelCosts tracks costs per model
	modelCosts map[string]*ModelUsage

	// hourlyUsage tracks usage per hour for rate limiting
	hourlyUsage map[string]*HourlyUsage
}

// UserUsage tracks usage for a single user/API key.
type UserUsage struct {
	APIKey           string
	TotalCost        float64
	TotalRequests    int64
	TotalInputTokens int64
	TotalOutputTokens int64
	FirstRequest     time.Time
	LastRequest      time.Time
	ModelBreakdown   map[string]*ModelUsage
}

// ModelUsage tracks usage for a single model.
type ModelUsage struct {
	Model            string
	TotalCost        float64
	TotalRequests    int64
	TotalInputTokens int64
	TotalOutputTokens int64
	AverageCost      float64
}

// HourlyUsage tracks usage within an hour for rate monitoring.
type HourlyUsage struct {
	Hour         time.Time
	Requests     int64
	TokensUsed   int64
	Cost         float64
}

// NewTracker creates a new usage tracker.
func NewTracker(calculator *Calculator, storage store.Storage) *Tracker {
	return &Tracker{
		calculator:  calculator,
		storage:     storage,
		userCosts:   make(map[string]*UserUsage),
		modelCosts:  make(map[string]*ModelUsage),
		hourlyUsage: make(map[string]*HourlyUsage),
	}
}

// Track records usage for a request.
func (t *Tracker) Track(ctx context.Context, apiKey string, model string, cost Cost) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// Track user usage
	if err := t.trackUser(apiKey, model, cost, now); err != nil {
		return fmt.Errorf("failed to track user usage: %w", err)
	}

	// Track model usage
	if err := t.trackModel(model, cost); err != nil {
		return fmt.Errorf("failed to track model usage: %w", err)
	}

	// Track hourly usage
	if err := t.trackHourly(apiKey, cost, now); err != nil {
		return fmt.Errorf("failed to track hourly usage: %w", err)
	}

	// Persist to storage (async, best-effort)
	go t.persistUsage(ctx, apiKey, model, cost)

	return nil
}

// trackUser tracks usage for a user.
func (t *Tracker) trackUser(apiKey string, model string, cost Cost, now time.Time) error {
	usage, ok := t.userCosts[apiKey]
	if !ok {
		usage = &UserUsage{
			APIKey:         apiKey,
			FirstRequest:   now,
			ModelBreakdown: make(map[string]*ModelUsage),
		}
		t.userCosts[apiKey] = usage
	}

	// Update totals
	usage.TotalCost += cost.TotalCost
	usage.TotalRequests++
	usage.TotalInputTokens += int64(cost.InputTokens)
	usage.TotalOutputTokens += int64(cost.OutputTokens)
	usage.LastRequest = now

	// Update model breakdown
	modelUsage, ok := usage.ModelBreakdown[model]
	if !ok {
		modelUsage = &ModelUsage{Model: model}
		usage.ModelBreakdown[model] = modelUsage
	}

	modelUsage.TotalCost += cost.TotalCost
	modelUsage.TotalRequests++
	modelUsage.TotalInputTokens += int64(cost.InputTokens)
	modelUsage.TotalOutputTokens += int64(cost.OutputTokens)
	modelUsage.AverageCost = modelUsage.TotalCost / float64(modelUsage.TotalRequests)

	return nil
}

// trackModel tracks aggregate usage per model.
func (t *Tracker) trackModel(model string, cost Cost) error {
	usage, ok := t.modelCosts[model]
	if !ok {
		usage = &ModelUsage{Model: model}
		t.modelCosts[model] = usage
	}

	usage.TotalCost += cost.TotalCost
	usage.TotalRequests++
	usage.TotalInputTokens += int64(cost.InputTokens)
	usage.TotalOutputTokens += int64(cost.OutputTokens)
	usage.AverageCost = usage.TotalCost / float64(usage.TotalRequests)

	return nil
}

// trackHourly tracks usage by hour.
func (t *Tracker) trackHourly(apiKey string, cost Cost, now time.Time) error {
	hour := now.Truncate(time.Hour)
	key := fmt.Sprintf("%s:%s", apiKey, hour.Format(time.RFC3339))

	usage, ok := t.hourlyUsage[key]
	if !ok {
		usage = &HourlyUsage{Hour: hour}
		t.hourlyUsage[key] = usage
	}

	usage.Requests++
	usage.TokensUsed += int64(cost.TotalTokens)
	usage.Cost += cost.TotalCost

	return nil
}

// persistUsage persists usage to storage.
func (t *Tracker) persistUsage(ctx context.Context, apiKey string, model string, cost Cost) {
	// Store in storage with TTL (7 days)
	key := fmt.Sprintf("usage:%s:%s:%d", apiKey, model, time.Now().Unix())
	t.storage.Set(ctx, key, cost, 7*24*time.Hour)
}

// GetUserUsage retrieves usage for a user.
func (t *Tracker) GetUserUsage(ctx context.Context, apiKey string) (*UserUsage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	usage, ok := t.userCosts[apiKey]
	if !ok {
		return nil, fmt.Errorf("no usage found for API key: %s", apiKey)
	}

	// Return a copy to avoid race conditions
	usageCopy := *usage
	usageCopy.ModelBreakdown = make(map[string]*ModelUsage)
	for k, v := range usage.ModelBreakdown {
		modelCopy := *v
		usageCopy.ModelBreakdown[k] = &modelCopy
	}

	return &usageCopy, nil
}

// GetModelUsage retrieves aggregate usage for a model.
func (t *Tracker) GetModelUsage(ctx context.Context, model string) (*ModelUsage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	usage, ok := t.modelCosts[model]
	if !ok {
		return nil, fmt.Errorf("no usage found for model: %s", model)
	}

	// Return a copy
	usageCopy := *usage
	return &usageCopy, nil
}

// GetHourlyUsage retrieves hourly usage for a user.
func (t *Tracker) GetHourlyUsage(ctx context.Context, apiKey string, hour time.Time) (*HourlyUsage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hour = hour.Truncate(time.Hour)
	key := fmt.Sprintf("%s:%s", apiKey, hour.Format(time.RFC3339))

	usage, ok := t.hourlyUsage[key]
	if !ok {
		return &HourlyUsage{Hour: hour}, nil // Return empty usage
	}

	// Return a copy
	usageCopy := *usage
	return &usageCopy, nil
}

// GetAllUserUsage retrieves usage for all users.
func (t *Tracker) GetAllUserUsage(ctx context.Context) map[string]*UserUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create copies
	result := make(map[string]*UserUsage)
	for k, v := range t.userCosts {
		usageCopy := *v
		usageCopy.ModelBreakdown = make(map[string]*ModelUsage)
		for mk, mv := range v.ModelBreakdown {
			modelCopy := *mv
			usageCopy.ModelBreakdown[mk] = &modelCopy
		}
		result[k] = &usageCopy
	}

	return result
}

// GetAllModelUsage retrieves aggregate usage for all models.
func (t *Tracker) GetAllModelUsage(ctx context.Context) map[string]*ModelUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create copies
	result := make(map[string]*ModelUsage)
	for k, v := range t.modelCosts {
		usageCopy := *v
		result[k] = &usageCopy
	}

	return result
}

// GetTopUsers returns the top N users by cost.
func (t *Tracker) GetTopUsers(ctx context.Context, n int) []*UserUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Convert to slice
	users := make([]*UserUsage, 0, len(t.userCosts))
	for _, usage := range t.userCosts {
		usageCopy := *usage
		users = append(users, &usageCopy)
	}

	// Sort by cost (simple bubble sort for small N)
	for i := 0; i < len(users) && i < n; i++ {
		for j := i + 1; j < len(users); j++ {
			if users[j].TotalCost > users[i].TotalCost {
				users[i], users[j] = users[j], users[i]
			}
		}
	}

	if n > len(users) {
		n = len(users)
	}

	return users[:n]
}

// GetTopModels returns the top N models by cost.
func (t *Tracker) GetTopModels(ctx context.Context, n int) []*ModelUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Convert to slice
	models := make([]*ModelUsage, 0, len(t.modelCosts))
	for _, usage := range t.modelCosts {
		usageCopy := *usage
		models = append(models, &usageCopy)
	}

	// Sort by cost
	for i := 0; i < len(models) && i < n; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[j].TotalCost > models[i].TotalCost {
				models[i], models[j] = models[j], models[i]
			}
		}
	}

	if n > len(models) {
		n = len(models)
	}

	return models[:n]
}

// ResetUserUsage resets usage for a specific user.
func (t *Tracker) ResetUserUsage(ctx context.Context, apiKey string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.userCosts, apiKey)
	return nil
}

// ResetAllUsage resets all usage tracking.
func (t *Tracker) ResetAllUsage(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.userCosts = make(map[string]*UserUsage)
	t.modelCosts = make(map[string]*ModelUsage)
	t.hourlyUsage = make(map[string]*HourlyUsage)

	return nil
}

// CleanupOldHourlyUsage removes hourly usage older than 24 hours.
func (t *Tracker) CleanupOldHourlyUsage(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for key, usage := range t.hourlyUsage {
		if usage.Hour.Before(cutoff) {
			delete(t.hourlyUsage, key)
		}
	}
}

// StartCleanupJob starts a background job to clean up old hourly usage.
func (t *Tracker) StartCleanupJob(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				t.CleanupOldHourlyUsage(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}