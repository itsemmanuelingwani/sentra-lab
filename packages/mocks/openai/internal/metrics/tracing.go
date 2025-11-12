// Package metrics provides observability tools.
// This file implements distributed tracing using OpenTelemetry.
package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is the global tracer instance.
var Tracer trace.Tracer

// TracingConfig contains tracing configuration.
type TracingConfig struct {
	// Enabled indicates if tracing is enabled
	Enabled bool

	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment (dev, staging, prod)
	Environment string

	// Endpoint is the OTLP collector endpoint
	Endpoint string

	// SampleRate is the sampling rate (0.0 to 1.0)
	// 1.0 = trace everything, 0.1 = trace 10%
	SampleRate float64
}

// DefaultTracingConfig returns default tracing configuration.
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:        false,
		ServiceName:    "openai-mock",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       "localhost:4317",
		SampleRate:     1.0,
	}
}

// InitTracing initializes distributed tracing.
func InitTracing(config TracingConfig) error {
	if !config.Enabled {
		// Use no-op tracer if tracing is disabled
		Tracer = otel.Tracer(config.ServiceName)
		return nil
	}

	// Create OTLP exporter
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(config.Endpoint),
			otlptracegrpc.WithInsecure(), // Use TLS in production
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Get tracer
	Tracer = tp.Tracer(config.ServiceName)

	return nil
}

// StartSpan starts a new span with the given name.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if Tracer == nil {
		// Initialize with default config if not initialized
		InitTracing(DefaultTracingConfig())
	}

	return Tracer.Start(ctx, name)
}

// StartSpanWithAttributes starts a span with attributes.
func StartSpanWithAttributes(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, name)
	span.SetAttributes(attrs...)
	return ctx, span
}

// EndSpan ends a span and records any error.
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// SpanFromContext returns the span from the context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span in context.
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the current span in context.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordSpanError records an error on the current span in context.
func RecordSpanError(ctx context.Context, err error) {
	span := SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Common attribute keys for consistency
var (
	AttrModel          = attribute.Key("model")
	AttrEndpoint       = attribute.Key("endpoint")
	AttrStatusCode     = attribute.Key("http.status_code")
	AttrMethod         = attribute.Key("http.method")
	AttrPath           = attribute.Key("http.path")
	AttrRequestID      = attribute.Key("request.id")
	AttrAPIKey         = attribute.Key("api.key")
	AttrInputTokens    = attribute.Key("tokens.input")
	AttrOutputTokens   = attribute.Key("tokens.output")
	AttrTotalTokens    = attribute.Key("tokens.total")
	AttrCost           = attribute.Key("cost.usd")
	AttrErrorType      = attribute.Key("error.type")
	AttrCacheType      = attribute.Key("cache.type")
	AttrCacheHit       = attribute.Key("cache.hit")
	AttrRateLimitType  = attribute.Key("rate_limit.type")
	AttrRateLimitHit   = attribute.Key("rate_limit.hit")
	AttrStreamingMode  = attribute.Key("streaming.mode")
	AttrSimulatedDelay = attribute.Key("simulated.delay_ms")
)

// TraceHTTPRequest creates a span for an HTTP request.
func TraceHTTPRequest(ctx context.Context, method, path string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "http.request",
		AttrMethod.String(method),
		AttrPath.String(path),
	)
}

// TraceChatCompletion creates a span for a chat completion.
func TraceChatCompletion(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "chat.completion",
		AttrModel.String(model),
		AttrEndpoint.String("/v1/chat/completions"),
	)
}

// TraceCompletion creates a span for a legacy completion.
func TraceCompletion(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "completion",
		AttrModel.String(model),
		AttrEndpoint.String("/v1/completions"),
	)
}

// TraceEmbedding creates a span for an embedding request.
func TraceEmbedding(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "embedding",
		AttrModel.String(model),
		AttrEndpoint.String("/v1/embeddings"),
	)
}

// TraceImageGeneration creates a span for image generation.
func TraceImageGeneration(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "image.generation",
		AttrModel.String(model),
		AttrEndpoint.String("/v1/images/generations"),
	)
}

// TraceTokenization creates a span for tokenization.
func TraceTokenization(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "tokenization",
		AttrModel.String(model),
	)
}

// TraceRateLimit creates a span for rate limit checking.
func TraceRateLimit(ctx context.Context, limitType string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "rate_limit.check",
		AttrRateLimitType.String(limitType),
	)
}

// TraceCacheOperation creates a span for a cache operation.
func TraceCacheOperation(ctx context.Context, operation, cacheType string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "cache."+operation,
		AttrCacheType.String(cacheType),
	)
}

// TraceStorageOperation creates a span for a storage operation.
func TraceStorageOperation(ctx context.Context, operation, key string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "storage."+operation,
		attribute.String("key", key),
	)
}

// TraceFixtureLoad creates a span for fixture loading.
func TraceFixtureLoad(ctx context.Context, fixtureType string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "fixture.load",
		attribute.String("fixture.type", fixtureType),
	)
}

// TraceResponseGeneration creates a span for response generation.
func TraceResponseGeneration(ctx context.Context, model string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "response.generation",
		AttrModel.String(model),
	)
}

// TraceLatencySimulation creates a span for latency simulation.
func TraceLatencySimulation(ctx context.Context, model string, delayMs int64) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, "latency.simulation",
		AttrModel.String(model),
		AttrSimulatedDelay.Int64(delayMs),
	)
}

// RecordTokenUsage records token usage on the current span.
func RecordTokenUsage(ctx context.Context, inputTokens, outputTokens int) {
	AddSpanAttributes(ctx,
		AttrInputTokens.Int(inputTokens),
		AttrOutputTokens.Int(outputTokens),
		AttrTotalTokens.Int(inputTokens+outputTokens),
	)
}

// RecordCost records cost on the current span.
func RecordCost(ctx context.Context, cost float64) {
	AddSpanAttributes(ctx,
		AttrCost.Float64(cost),
	)
}

// RecordCacheHit records a cache hit/miss on the current span.
func RecordCacheHit(ctx context.Context, hit bool) {
	AddSpanAttributes(ctx,
		AttrCacheHit.Bool(hit),
	)
}

// RecordRateLimitHit records a rate limit hit on the current span.
func RecordRateLimitHit(ctx context.Context, hit bool) {
	AddSpanAttributes(ctx,
		AttrRateLimitHit.Bool(hit),
	)
}

// RecordStreamingMode records streaming mode on the current span.
func RecordStreamingMode(ctx context.Context, streaming bool) {
	AddSpanAttributes(ctx,
		AttrStreamingMode.Bool(streaming),
	)
}