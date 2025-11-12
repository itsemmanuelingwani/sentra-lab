// Package metrics provides observability tools.
// This file implements Prometheus metrics for monitoring the OpenAI mock server.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the OpenAI mock server.
// These metrics match production monitoring requirements.

var (
	// RequestsTotal counts total HTTP requests by model and status.
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "requests_total",
			Help:      "Total number of API requests",
		},
		[]string{"model", "endpoint", "status"},
	)

	// RequestDuration measures request latency (including simulated delay).
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openai_mock",
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds (including simulated latency)",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		},
		[]string{"model", "endpoint"},
	)

	// ProcessingDuration measures actual processing time (excluding simulated delay).
	ProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openai_mock",
			Name:      "processing_duration_seconds",
			Help:      "Actual processing duration in seconds (excluding simulated latency)",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 15), // 0.1ms to ~3.2s
		},
		[]string{"model", "endpoint"},
	)

	// TokensTotal counts total tokens processed.
	TokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "tokens_total",
			Help:      "Total tokens processed",
		},
		[]string{"model", "type"}, // type: input or output
	)

	// CostUSDTotal tracks cumulative cost in USD.
	CostUSDTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "cost_usd_total",
			Help:      "Total cost in USD (simulated)",
		},
		[]string{"model"},
	)

	// ErrorsTotal counts errors by type.
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"model", "error_type"},
	)

	// RateLimitHits counts rate limit hits.
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "rate_limit_hits_total",
			Help:      "Total number of rate limit hits",
		},
		[]string{"api_key", "limit_type"}, // limit_type: requests or tokens
	)

	// RateLimitRemaining tracks remaining rate limit quota.
	RateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "rate_limit_remaining",
			Help:      "Remaining rate limit quota",
		},
		[]string{"api_key", "limit_type"},
	)

	// CacheHits counts cache hits.
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_type"}, // cache_type: token, response, fixture
	)

	// CacheMisses counts cache misses.
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	// CacheSize tracks cache size.
	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "cache_size",
			Help:      "Current cache size (number of entries)",
		},
		[]string{"cache_type"},
	)

	// StreamingConnections tracks active streaming connections.
	StreamingConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "streaming_connections",
			Help:      "Current number of active streaming connections",
		},
	)

	// FixturesLoaded counts loaded fixtures.
	FixturesLoaded = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "fixtures_loaded",
			Help:      "Number of loaded fixtures",
		},
		[]string{"fixture_type"},
	)

	// SimulatedLatency tracks simulated latency distribution.
	SimulatedLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openai_mock",
			Name:      "simulated_latency_seconds",
			Help:      "Simulated latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.01, 1.5, 15), // 10ms to ~4s
		},
		[]string{"model"},
	)

	// StorageOperations counts storage operations.
	StorageOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openai_mock",
			Name:      "storage_operations_total",
			Help:      "Total number of storage operations",
		},
		[]string{"operation", "status"}, // operation: get, set, delete; status: success, error
	)

	// StorageLatency measures storage operation latency.
	StorageLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openai_mock",
			Name:      "storage_latency_seconds",
			Help:      "Storage operation latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.00001, 2, 15), // 10µs to ~320ms
		},
		[]string{"operation"},
	)

	// TokenizationLatency measures tokenization latency.
	TokenizationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openai_mock",
			Name:      "tokenization_latency_seconds",
			Help:      "Tokenization latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.00001, 2, 12), // 10µs to ~40ms
		},
		[]string{"model"},
	)

	// ActiveRequests tracks currently processing requests.
	ActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "active_requests",
			Help:      "Number of currently processing requests",
		},
		[]string{"endpoint"},
	)

	// RequestsPerSecond tracks request rate.
	RequestsPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "requests_per_second",
			Help:      "Current requests per second",
		},
		[]string{"endpoint"},
	)

	// MemoryUsage tracks memory usage.
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "memory_usage_bytes",
			Help:      "Memory usage in bytes",
		},
		[]string{"component"}, // component: cache, storage, total
	)

	// BuildInfo provides build information.
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "openai_mock",
			Name:      "build_info",
			Help:      "Build information",
		},
		[]string{"version", "go_version", "build_time"},
	)
)

// RecordRequest records a completed request.
func RecordRequest(model, endpoint string, statusCode int, duration float64, processingDuration float64) {
	status := "success"
	if statusCode >= 400 {
		status = "error"
	}

	RequestsTotal.WithLabelValues(model, endpoint, status).Inc()
	RequestDuration.WithLabelValues(model, endpoint).Observe(duration)
	ProcessingDuration.WithLabelValues(model, endpoint).Observe(processingDuration)
}

// RecordTokens records token usage.
func RecordTokens(model string, inputTokens, outputTokens int) {
	TokensTotal.WithLabelValues(model, "input").Add(float64(inputTokens))
	TokensTotal.WithLabelValues(model, "output").Add(float64(outputTokens))
}

// RecordCost records cost in USD.
func RecordCost(model string, cost float64) {
	CostUSDTotal.WithLabelValues(model).Add(cost)
}

// RecordError records an error.
func RecordError(model string, errorType string) {
	ErrorsTotal.WithLabelValues(model, errorType).Inc()
}

// RecordRateLimitHit records a rate limit hit.
func RecordRateLimitHit(apiKey string, limitType string, remaining int) {
	RateLimitHits.WithLabelValues(apiKey, limitType).Inc()
	RateLimitRemaining.WithLabelValues(apiKey, limitType).Set(float64(remaining))
}

// RecordCacheHit records a cache hit.
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss.
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}

// SetCacheSize sets the current cache size.
func SetCacheSize(cacheType string, size int) {
	CacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// IncrementStreamingConnections increments active streaming connections.
func IncrementStreamingConnections() {
	StreamingConnections.Inc()
}

// DecrementStreamingConnections decrements active streaming connections.
func DecrementStreamingConnections() {
	StreamingConnections.Dec()
}

// SetFixturesLoaded sets the number of loaded fixtures.
func SetFixturesLoaded(fixtureType string, count int) {
	FixturesLoaded.WithLabelValues(fixtureType).Set(float64(count))
}

// RecordSimulatedLatency records simulated latency.
func RecordSimulatedLatency(model string, latency float64) {
	SimulatedLatency.WithLabelValues(model).Observe(latency)
}

// RecordStorageOperation records a storage operation.
func RecordStorageOperation(operation string, success bool, latency float64) {
	status := "success"
	if !success {
		status = "error"
	}

	StorageOperations.WithLabelValues(operation, status).Inc()
	StorageLatency.WithLabelValues(operation).Observe(latency)
}

// RecordTokenization records tokenization metrics.
func RecordTokenization(model string, latency float64) {
	TokenizationLatency.WithLabelValues(model).Observe(latency)
}

// IncrementActiveRequests increments active request count.
func IncrementActiveRequests(endpoint string) {
	ActiveRequests.WithLabelValues(endpoint).Inc()
}

// DecrementActiveRequests decrements active request count.
func DecrementActiveRequests(endpoint string) {
	ActiveRequests.WithLabelValues(endpoint).Dec()
}

// SetRequestsPerSecond sets the current RPS.
func SetRequestsPerSecond(endpoint string, rps float64) {
	RequestsPerSecond.WithLabelValues(endpoint).Set(rps)
}

// SetMemoryUsage sets memory usage.
func SetMemoryUsage(component string, bytes int64) {
	MemoryUsage.WithLabelValues(component).Set(float64(bytes))
}

// SetBuildInfo sets build information (should be called once at startup).
func SetBuildInfo(version, goVersion, buildTime string) {
	BuildInfo.WithLabelValues(version, goVersion, buildTime).Set(1)
}