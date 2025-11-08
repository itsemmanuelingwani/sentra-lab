package start

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type HealthChecker struct {
	maxAttempts int
	interval    time.Duration
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		maxAttempts: 60,
		interval:    1 * time.Second,
	}
}

func (hc *HealthChecker) CheckHTTP(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for attempt := 0; attempt < hc.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(hc.interval)
	}

	return fmt.Errorf("health check timeout after %d attempts", hc.maxAttempts)
}

func (hc *HealthChecker) CheckTCP(ctx context.Context, host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)

	for attempt := 0; attempt < hc.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		time.Sleep(hc.interval)
	}

	return fmt.Errorf("TCP health check timeout for %s after %d attempts", address, hc.maxAttempts)
}

func (hc *HealthChecker) CheckGRPC(ctx context.Context, address string) error {
	for attempt := 0; attempt < hc.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		time.Sleep(hc.interval)
	}

	return fmt.Errorf("gRPC health check timeout for %s after %d attempts", address, hc.maxAttempts)
}

type ServiceHealth struct {
	Name      string
	Healthy   bool
	CheckedAt time.Time
	Duration  time.Duration
	Error     error
}

func (hc *HealthChecker) CheckServices(ctx context.Context, services []ServiceConfig) ([]ServiceHealth, error) {
	results := make([]ServiceHealth, len(services))
	errChan := make(chan error, len(services))

	for i, svc := range services {
		go func(idx int, service ServiceConfig) {
			startTime := time.Now()

			var err error
			switch service.HealthCheck.Type {
			case "http":
				err = hc.CheckHTTP(ctx, service.HealthCheck.URL)
			case "tcp":
				err = hc.CheckTCP(ctx, service.HealthCheck.Host, service.HealthCheck.Port)
			case "grpc":
				err = hc.CheckGRPC(ctx, service.HealthCheck.Address)
			default:
				err = fmt.Errorf("unknown health check type: %s", service.HealthCheck.Type)
			}

			duration := time.Since(startTime)

			results[idx] = ServiceHealth{
				Name:      service.Name,
				Healthy:   err == nil,
				CheckedAt: time.Now(),
				Duration:  duration,
				Error:     err,
			}

			errChan <- err
		}(i, svc)
	}

	var errors []error
	for i := 0; i < len(services); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("health checks failed: %v", errors)
	}

	return results, nil
}

type ServiceConfig struct {
	Name        string
	Image       string
	Ports       map[string]int
	Environment map[string]string
	Volumes     []string
	HealthCheck HealthCheckConfig
	DependsOn   []string
}

type HealthCheckConfig struct {
	Type    string
	URL     string
	Host    string
	Port    int
	Address string
}

func GenerateServiceConfigs(mockConfig map[string]interface{}) []ServiceConfig {
	configs := []ServiceConfig{
		{
			Name:  "simulation-engine",
			Image: "sentra/lab-engine:latest",
			Ports: map[string]int{
				"50051": 50051,
			},
			Environment: map[string]string{
				"LOG_LEVEL":       "info",
				"STORAGE_PATH":    "/data",
				"ENABLE_RECORDER": "true",
			},
			Volumes: []string{
				"./.sentra-lab:/data",
			},
			HealthCheck: HealthCheckConfig{
				Type:    "grpc",
				Address: "localhost:50051",
			},
		},
	}

	if openai, ok := mockConfig["openai"].(map[string]interface{}); ok {
		if enabled, ok := openai["enabled"].(bool); ok && enabled {
			port := 8080
			if p, ok := openai["port"].(int); ok {
				port = p
			}

			configs = append(configs, ServiceConfig{
				Name:  "mock-openai",
				Image: "sentra/mock-openai:latest",
				Ports: map[string]int{
					"8080": port,
				},
				Environment: map[string]string{
					"LATENCY_MS":  fmt.Sprintf("%v", openai["latency_ms"]),
					"RATE_LIMIT":  fmt.Sprintf("%v", openai["rate_limit"]),
					"ERROR_RATE":  fmt.Sprintf("%v", openai["error_rate"]),
				},
				Volumes: []string{
					"./fixtures:/fixtures:ro",
				},
				HealthCheck: HealthCheckConfig{
					Type: "http",
					URL:  fmt.Sprintf("http://localhost:%d/health", port),
				},
			})
		}
	}

	if stripe, ok := mockConfig["stripe"].(map[string]interface{}); ok {
		if enabled, ok := stripe["enabled"].(bool); ok && enabled {
			port := 8081
			if p, ok := stripe["port"].(int); ok {
				port = p
			}

			configs = append(configs, ServiceConfig{
				Name:  "mock-stripe",
				Image: "sentra/mock-stripe:latest",
				Ports: map[string]int{
					"8080": port,
				},
				Environment: map[string]string{
					"LATENCY_MS": fmt.Sprintf("%v", stripe["latency_ms"]),
				},
				Volumes: []string{
					"./fixtures:/fixtures:ro",
				},
				HealthCheck: HealthCheckConfig{
					Type: "http",
					URL:  fmt.Sprintf("http://localhost:%d/health", port),
				},
			})
		}
	}

	if coreledger, ok := mockConfig["coreledger"].(map[string]interface{}); ok {
		if enabled, ok := coreledger["enabled"].(bool); ok && enabled {
			port := 8082
			if p, ok := coreledger["port"].(int); ok {
				port = p
			}

			configs = append(configs, ServiceConfig{
				Name:  "mock-coreledger",
				Image: "sentra/mock-coreledger:latest",
				Ports: map[string]int{
					"8080": port,
				},
				Volumes: []string{
					"./fixtures:/fixtures:ro",
				},
				HealthCheck: HealthCheckConfig{
					Type: "http",
					URL:  fmt.Sprintf("http://localhost:%d/health", port),
				},
			})
		}
	}

	return configs
}
