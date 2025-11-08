package test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sentra-lab/cli/internal/grpc"
)

type Runner struct {
	engineClient *grpc.EngineClient
	parallel     int
	failFast     bool
}

func NewRunner(engineClient *grpc.EngineClient, parallel int, failFast bool) *Runner {
	return &Runner{
		engineClient: engineClient,
		parallel:     parallel,
		failFast:     failFast,
	}
}

func (r *Runner) RunScenarios(ctx context.Context, scenarios []string, progressFn func(string, string, float64)) ([]*TestResult, error) {
	results := make([]*TestResult, len(scenarios))
	resultsMu := sync.Mutex{}

	semaphore := make(chan struct{}, r.parallel)
	var wg sync.WaitGroup
	errChan := make(chan error, len(scenarios))
	stopChan := make(chan struct{})

	for i, scenario := range scenarios {
		wg.Add(1)

		go func(idx int, scenarioPath string) {
			defer wg.Done()

			select {
			case <-stopChan:
				resultsMu.Lock()
				results[idx] = &TestResult{
					Scenario: scenarioPath,
					Status:   "skipped",
				}
				resultsMu.Unlock()
				return
			case semaphore <- struct{}{}:
			}

			defer func() { <-semaphore }()

			progressFn(scenarioPath, "running", 0.0)

			result, err := r.runScenario(ctx, scenarioPath, progressFn)

			resultsMu.Lock()
			results[idx] = result
			resultsMu.Unlock()

			if err != nil {
				errChan <- err

				if r.failFast {
					close(stopChan)
				}
			}

			status := result.Status
			progressFn(scenarioPath, status, 1.0)

		}(i, scenario)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if r.failFast && len(errors) > 0 {
		return results, errors[0]
	}

	return results, nil
}

func (r *Runner) runScenario(ctx context.Context, scenarioPath string, progressFn func(string, string, float64)) (*TestResult, error) {
	startTime := time.Now()

	result := &TestResult{
		Scenario:  scenarioPath,
		StartedAt: startTime,
	}

	req := &grpc.StartSimulationRequest{
		ScenarioPath: scenarioPath,
		Config: grpc.SimulationConfig{
			RecordFullTrace: true,
			EnableCostTracking: true,
		},
	}

	run, err := r.engineClient.StartSimulation(ctx, req)
	if err != nil {
		result.Status = "failed"
		result.Failures = append(result.Failures, fmt.Sprintf("Failed to start simulation: %v", err))
		result.CompletedAt = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.RunID = run.ID

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			result.Status = "failed"
			result.Failures = append(result.Failures, "Context canceled")
			result.CompletedAt = time.Now()
			result.Duration = time.Since(startTime)
			return result, ctx.Err()

		case <-ticker.C:
			status, err := r.engineClient.GetSimulationStatus(ctx, run.ID)
			if err != nil {
				result.Status = "failed"
				result.Failures = append(result.Failures, fmt.Sprintf("Failed to get status: %v", err))
				result.CompletedAt = time.Now()
				result.Duration = time.Since(startTime)
				return result, err
			}

			progressFn(scenarioPath, status.Status, status.Progress)

			if status.Status == "completed" || status.Status == "failed" {
				result.Status = status.Status
				if status.Status == "completed" {
					result.Status = "passed"
				}
				result.Duration = status.Duration
				result.CostUSD = status.CostUSD
				result.Assertions = status.Assertions
				result.Failures = status.Failures
				result.CompletedAt = time.Now()

				return result, nil
			}
		}
	}
}
