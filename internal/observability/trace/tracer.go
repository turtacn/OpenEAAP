// Package trace provides distributed tracing capabilities for OpenEAAP.
// It integrates OpenTelemetry SDK to provide trace and span creation,
// management, and automatic TraceID/SpanID injection across request chains.
package trace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================================
// Tracer Interface
// ============================================================================

// Tracer defines the distributed tracing interface
type Tracer interface {
	// Start creates a new span
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)

	// StartWithParent creates a new span with parent
	StartWithParent(ctx context.Context, parentSpan trace.Span, spanName string) (context.Context, trace.Span)

	// GetTraceID returns trace ID from context
	GetTraceID(ctx context.Context) string

	// GetSpanID returns span ID from context
	GetSpanID(ctx context.Context) string

	// InjectContext injects trace context into carrier
	InjectContext(ctx context.Context, carrier propagation.TextMapCarrier)

	// ExtractContext extracts trace context from carrier
	ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context

	// Shutdown gracefully shuts down the tracer
	Shutdown(ctx context.Context) error
}

// ============================================================================
// OpenTelemetry Tracer Implementation
// ============================================================================

// OtelTracer wraps OpenTelemetry tracer
type OtelTracer struct {
	tracer         trace.Tracer
	provider       *sdktrace.TracerProvider
	propagator     propagation.TextMapPropagator
	serviceName    string
	serviceVersion string
}

// TracerConfig defines tracer configuration
type TracerConfig struct {
	// Service name
	ServiceName string

	// Service version
	ServiceVersion string

	// Environment (development, staging, production)
	Environment string

	// Provider (jaeger, zipkin, otlp)
	Provider string

	// Endpoint for exporter
	Endpoint string

	// Sampling rate (0.0 - 1.0)
	SamplingRate float64

	// Enable console exporter for debugging
	EnableConsole bool
}

// ============================================================================
// Tracer Initialization
// ============================================================================

// NewTracer creates a new OpenTelemetry tracer
func NewTracer(cfg TracerConfig) (Tracer, error) {
	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on provider
	var exporter sdktrace.SpanExporter
	switch cfg.Provider {
	case "jaeger":
		exporter, err = createJaegerExporter(cfg.Endpoint)
	case "zipkin":
		exporter, err = createZipkinExporter(cfg.Endpoint)
	case "otlp":
		exporter, err = createOTLPExporter(cfg.Endpoint)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SamplingRate),
	)

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Create propagator
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// Set global propagator
	otel.SetTextMapPropagator(propagator)

	// Create tracer
	tracer := tp.Tracer(
		cfg.ServiceName,
		trace.WithInstrumentationVersion(cfg.ServiceVersion),
	)

	return &OtelTracer{
		tracer:         tracer,
		provider:       tp,
		propagator:     propagator,
		serviceName:    cfg.ServiceName,
		serviceVersion: cfg.ServiceVersion,
	}, nil
}

// ============================================================================
// Exporter Creation
// ============================================================================

// createJaegerExporter creates a Jaeger exporter
func createJaegerExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(endpoint),
		),
	)
}

// createZipkinExporter creates a Zipkin exporter
func createZipkinExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return zipkin.New(endpoint)
}

// createOTLPExporter creates an OTLP exporter
func createOTLPExporter(endpoint string) (sdktrace.SpanExporter, error) {
	ctx := context.Background()
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	return otlptrace.New(ctx, client)
}

// ============================================================================
// Tracer Methods
// ============================================================================

// Start creates a new span
func (t *OtelTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartWithParent creates a new span with parent
func (t *OtelTracer) StartWithParent(ctx context.Context, parentSpan trace.Span, spanName string) (context.Context, trace.Span) {
	opts := []trace.SpanStartOption{
		trace.WithLinks(trace.Link{
			SpanContext: parentSpan.SpanContext(),
		}),
	}
	return t.tracer.Start(ctx, spanName, opts...)
}

// GetTraceID returns trace ID from context
func (t *OtelTracer) GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanID returns span ID from context
func (t *OtelTracer) GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// InjectContext injects trace context into carrier
func (t *OtelTracer) InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	t.propagator.Inject(ctx, carrier)
}

// ExtractContext extracts trace context from carrier
func (t *OtelTracer) ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return t.propagator.Extract(ctx, carrier)
}

// Shutdown gracefully shuts down the tracer
func (t *OtelTracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// ============================================================================
// Span Helpers
// ============================================================================

// SetSpanAttributes sets attributes on current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// SetSpanStatus sets status on current span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// RecordSpanError records an error on current span
func RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// AddSpanEvent adds an event to current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// EndSpan ends the current span
func EndSpan(span trace.Span) {
	span.End()
}

// ============================================================================
// Common Attribute Constructors
// ============================================================================

// StringAttr creates a string attribute
func StringAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// IntAttr creates an int attribute
func IntAttr(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// Int64Attr creates an int64 attribute
func Int64Attr(key string, value int64) attribute.KeyValue {
	return attribute.Int64(key, value)
}

// Float64Attr creates a float64 attribute
func Float64Attr(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

// BoolAttr creates a bool attribute
func BoolAttr(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// ============================================================================
// Pre-defined Attributes
// ============================================================================

// HTTP attributes
func HTTPMethodAttr(method string) attribute.KeyValue {
	return attribute.String("http.method", method)
}

func HTTPURLAttr(url string) attribute.KeyValue {
	return attribute.String("http.url", url)
}

func HTTPStatusCodeAttr(code int) attribute.KeyValue {
	return attribute.Int("http.status_code", code)
}

func HTTPRouteAttr(route string) attribute.KeyValue {
	return attribute.String("http.route", route)
}

// Database attributes
func DBSystemAttr(system string) attribute.KeyValue {
	return attribute.String("db.system", system)
}

func DBNameAttr(name string) attribute.KeyValue {
	return attribute.String("db.name", name)
}

func DBStatementAttr(statement string) attribute.KeyValue {
	return attribute.String("db.statement", statement)
}

func DBOperationAttr(operation string) attribute.KeyValue {
	return attribute.String("db.operation", operation)
}

// RPC attributes
func RPCSystemAttr(system string) attribute.KeyValue {
	return attribute.String("rpc.system", system)
}

func RPCServiceAttr(service string) attribute.KeyValue {
	return attribute.String("rpc.service", service)
}

func RPCMethodAttr(method string) attribute.KeyValue {
	return attribute.String("rpc.method", method)
}

// Message queue attributes
func MessagingSystemAttr(system string) attribute.KeyValue {
	return attribute.String("messaging.system", system)
}

func MessagingDestinationAttr(destination string) attribute.KeyValue {
	return attribute.String("messaging.destination", destination)
}

func MessagingOperationAttr(operation string) attribute.KeyValue {
	return attribute.String("messaging.operation", operation)
}

// Custom application attributes
func UserIDAttr(userID string) attribute.KeyValue {
	return attribute.String("user.id", userID)
}

func AgentIDAttr(agentID string) attribute.KeyValue {
	return attribute.String("agent.id", agentID)
}

func WorkflowIDAttr(workflowID string) attribute.KeyValue {
	return attribute.String("workflow.id", workflowID)
}

func RequestIDAttr(requestID string) attribute.KeyValue {
	return attribute.String("request.id", requestID)
}

// ============================================================================
// Span Kind Options
// ============================================================================

// SpanKindServer creates a server span
func SpanKindServer() trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKindServer)
}

// SpanKindClient creates a client span
func SpanKindClient() trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKindClient)
}

// SpanKindProducer creates a producer span
func SpanKindProducer() trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKindProducer)
}

// SpanKindConsumer creates a consumer span
func SpanKindConsumer() trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKindConsumer)
}

// SpanKindInternal creates an internal span
func SpanKindInternal() trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKindInternal)
}

// ============================================================================
// Utility Functions
// ============================================================================

// StartSpan is a convenience function to start a span
func StartSpan(ctx context.Context, tracer Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// TraceFunc wraps a function with tracing
func TraceFunc(ctx context.Context, tracer Tracer, name string, fn func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordSpanError(ctx, err)
	}

	return err
}

// TraceFuncWithResult wraps a function with tracing and returns result
func TraceFuncWithResult[T any](ctx context.Context, tracer Tracer, name string, fn func(context.Context) (T, error)) (T, error) {
ctx, span := tracer.Start(ctx, name)
defer span.End()

result, err := fn(ctx)
if err != nil {
RecordSpanError(ctx, err)
}

return result, err
}

// ============================================================================
// HTTP Propagation Carrier
// ============================================================================

// HTTPHeadersCarrier adapts http.Header to TextMapCarrier
type HTTPHeadersCarrier map[string][]string

// Get returns the value associated with the passed key
func (c HTTPHeadersCarrier) Get(key string) string {
	values := c[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set stores the key-value pair
func (c HTTPHeadersCarrier) Set(key, value string) {
	c[key] = []string{value}
}

// Keys lists the keys stored in this carrier
func (c HTTPHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// ============================================================================
// Global Tracer
// ============================================================================

var globalTracer Tracer

// SetGlobalTracer sets the global tracer instance
func SetGlobalTracer(tracer Tracer) {
	globalTracer = tracer
}

// GetGlobalTracer returns the global tracer instance
func GetGlobalTracer() Tracer {
	return globalTracer
}

// ============================================================================
// No-op Tracer
// ============================================================================

// NoopTracer is a tracer that does nothing
type NoopTracer struct{}

// NewNoopTracer creates a no-op tracer
func NewNoopTracer() Tracer {
	return &NoopTracer{}
}

func (t *NoopTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

func (t *NoopTracer) StartWithParent(ctx context.Context, parentSpan trace.Span, spanName string) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

func (t *NoopTracer) GetTraceID(ctx context.Context) string {
	return ""
}

func (t *NoopTracer) GetSpanID(ctx context.Context) string {
	return ""
}

func (t *NoopTracer) InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
}

func (t *NoopTracer) ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return ctx
}

func (t *NoopTracer) Shutdown(ctx context.Context) error {
	return nil
}

// ============================================================================
// Timing Utilities
// ============================================================================

// Timer helps measure operation duration
type Timer struct {
	start time.Time
	span  trace.Span
}

// StartTimer starts a new timer with span
func StartTimer(ctx context.Context, tracer Tracer, name string) (*Timer, context.Context) {
	ctx, span := tracer.Start(ctx, name)
	return &Timer{
		start: time.Now(),
		span:  span,
	}, ctx
}

// Stop stops the timer and records duration
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.start)
	t.span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	t.span.End()
	return duration
}

//Personal.AI order the ending
