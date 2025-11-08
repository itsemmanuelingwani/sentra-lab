package start

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ParallelHealthChecker struct {
	checker    *HealthChecker
	maxWorkers int
}

func NewParallelHealthChecker(maxWorkers int) *ParallelHealthChecker {
	return &ParallelHealthChecker{
		checker:    NewHealthChecker(),
		maxWorkers: maxWorkers,
	}
}

func (phc *ParallelHealthChecker) CheckAllServices(ctx context.Context, services []ServiceConfig) (map[string]*HealthResult, error) {
	results := make(map[string]*HealthResult)
	resultsMu := sync.Mutex{}

	semaphore := make(chan struct{}, phc.maxWorkers)
	var wg sync.WaitGroup
	errChan := make(chan error, len(services))

	for _, svc := range services {
		wg.Add(1)
		go func(service ServiceConfig) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := phc.checkService(ctx, service)

			resultsMu.Lock()
			results[service.Name] = result
			resultsMu.Unlock()

			if !result.Healthy {
				errChan <- fmt.Errorf("service %s unhealthy: %v", service.Name, result.Error)
			}
		}(svc)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("health checks failed for %d services", len(errors))
	}

	return results, nil
}

func (phc *ParallelHealthChecker) checkService(ctx context.Context, service ServiceConfig) *HealthResult {
	startTime := time.Now()

	var err error
	var attempts int

	switch service.HealthCheck.Type {
	case "http":
		err = phc.checker.CheckHTTP(ctx, service.HealthCheck.URL)
	case "tcp":
		err = phc.checker.CheckTCP(ctx, service.HealthCheck.Host, service.HealthCheck.Port)
	case "grpc":
		err = phc.checker.CheckGRPC(ctx, service.HealthCheck.Address)
	default:
		err = fmt.Errorf("unknown health check type: %s", service.HealthCheck.Type)
	}

	duration := time.Since(startTime)

	return &HealthResult{
		ServiceName: service.Name,
		Healthy:     err == nil,
		Duration:    duration,
		Attempts:    attempts,
		Error:       err,
		CheckedAt:   time.Now(),
	}
}

type HealthResult struct {
	ServiceName string
	Healthy     bool
	Duration    time.Duration
	Attempts    int
	Error       error
	CheckedAt   time.Time
}

func (hr *HealthResult) String() string {
	status := "✓ healthy"
	if !hr.Healthy {
		status = "✗ unhealthy"
	}

	msg := fmt.Sprintf("%s (%dms)", status, hr.Duration.Milliseconds())
	if hr.Error != nil {
		msg += fmt.Sprintf(" - %v", hr.Error)
	}

	return msg
}

type HealthMonitor struct {
	checker  *ParallelHealthChecker
	services []ServiceConfig
	interval time.Duration
	mu       sync.RWMutex
	results  map[string]*HealthResult
	stopCh   chan struct{}
}

func NewHealthMonitor(services []ServiceConfig, interval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		checker:  NewParallelHealthChecker(len(services)),
		services: services,
		interval: interval,
		results:  make(map[string]*HealthResult),
		stopCh:   make(chan struct{}),
	}
}

func (hm *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	hm.check(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.check(ctx)
		}
	}
}

func (hm *HealthMonitor) check(ctx context.Context) {
	results, _ := hm.checker.CheckAllServices(ctx, hm.services)

	hm.mu.Lock()
	hm.results = results
	hm.mu.Unlock()
}

func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
}

func (hm *HealthMonitor) GetResults() map[string]*HealthResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make(map[string]*HealthResult, len(hm.results))
	for k, v := range hm.results {
		results[k] = v
	}

	return results
}

func (hm *HealthMonitor) IsHealthy(serviceName string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result, ok := hm.results[serviceName]
	if !ok {
		return false
	}

	return result.Healthy
}

func (hm *HealthMonitor) AllHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	for _, result := range hm.results {
		if !result.Healthy {
			return false
		}
	}

	return len(hm.results) > 0
}
