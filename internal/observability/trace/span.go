// Package trace provides span data model and operations for distributed tracing.
// It defines the Span structure with TraceID, SpanID, ParentSpanID, and related
// methods for adding tags, logs, and setting status.
package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================================
// Span Data Model
// ============================================================================

// Span represents a single unit of work in a distributed trace
type Span struct {
	// Trace identifier
	TraceID string `json:"trace_id"`

	// Span identifier
	SpanID string `json:"span_id"`

	// Parent span identifier
	ParentSpanID string `json:"parent_span_id,omitempty"`

	// Operation name
	Operation string `json:"operation"`

	// Service name
	ServiceName string `json:"service_name"`

	// Start time
	StartTime time.Time `json:"start_time"`

	// End time
	EndTime time.Time `json:"end_time,omitempty"`

	// Duration in microseconds
	Duration int64 `json:"duration_us,omitempty"`

	// Tags (key-value pairs)
	Tags map[string]interface{} `json:"tags"`

	// Logs (timestamped events)
	Logs []SpanLog `json:"logs"`

	// Status
	Status SpanStatus `json:"status"`

	// Kind (server, client, producer, consumer, internal)
	Kind SpanKind `json:"kind"`

	// Resource attributes
	Resource map[string]interface{} `json:"resource"`

	// Events
	Events []SpanEvent `json:"events"`

	// Links to other spans
	Links []SpanLink `json:"links"`

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// SpanLog represents a timestamped log entry in a span
type SpanLog struct {
	// Timestamp
	Timestamp time.Time `json:"timestamp"`

	// Log level
	Level string `json:"level"`

	// Message
	Message string `json:"message"`

	// Fields
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// SpanEvent represents a significant event during span execution
type SpanEvent struct {
	// Name of the event
	Name string `json:"name"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`

	// Attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanLink represents a link to another span
type SpanLink struct {
	// Trace ID of linked span
	TraceID string `json:"trace_id"`

	// Span ID of linked span
	SpanID string `json:"span_id"`

	// Link attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	// Status code (ok, error, unset)
	Code string `json:"code"`

	// Status message
	Message string `json:"message,omitempty"`
}

// SpanKind represents the kind of span
type SpanKind string

const (
	// SpanKindServer indicates a server span
	SpanKindServerType SpanKind = "server"

	// SpanKindClient indicates a client span
	SpanKindClientType SpanKind = "client"

	// SpanKindProducer indicates a producer span
	SpanKindProducerType SpanKind = "producer"

	// SpanKindConsumer indicates a consumer span
	SpanKindConsumerType SpanKind = "consumer"

	// SpanKindInternal indicates an internal span
	SpanKindInternalType SpanKind = "internal"
)

// ============================================================================
// Span Creation
// ============================================================================

// NewSpan creates a new span
func NewSpan(traceID, spanID, parentSpanID, operation, serviceName string) *Span {
	return &Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Operation:    operation,
		ServiceName:  serviceName,
		StartTime:    time.Now(),
		Tags:         make(map[string]interface{}),
		Logs:         make([]SpanLog, 0),
		Events:       make([]SpanEvent, 0),
		Links:        make([]SpanLink, 0),
		Resource:     make(map[string]interface{}),
		Status: SpanStatus{
			Code: "unset",
		},
		Kind: SpanKindInternalType,
	}
}

// NewSpanFromContext creates a new span from context
func NewSpanFromContext(ctx context.Context, operation, serviceName string) *Span {
	spanCtx := trace.SpanContextFromContext(ctx)

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	parentSpan := trace.SpanFromContext(ctx)
	parentSpanID := ""
	if parentSpan.SpanContext().IsValid() {
		parentSpanID = parentSpan.SpanContext().SpanID().String()
	}

	return NewSpan(traceID, spanID, parentSpanID, operation, serviceName)
}

// ============================================================================
// Tag Operations
// ============================================================================

// AddTag adds a tag to the span
func (s *Span) AddTag(key string, value interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tags[key] = value
	return s
}

// AddTags adds multiple tags to the span
func (s *Span) AddTags(tags map[string]interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range tags {
		s.Tags[k] = v
	}
	return s
}

// GetTag retrieves a tag value
func (s *Span) GetTag(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Tags[key]
	return val, ok
}

// RemoveTag removes a tag from the span
func (s *Span) RemoveTag(key string) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Tags, key)
	return s
}

// ============================================================================
// Log Operations
// ============================================================================

// AddLog adds a log entry to the span
func (s *Span) AddLog(level, message string, fields map[string]interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := SpanLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	s.Logs = append(s.Logs, log)
	return s
}

// AddDebugLog adds a debug log entry
func (s *Span) AddDebugLog(message string, fields map[string]interface{}) *Span {
	return s.AddLog("debug", message, fields)
}

// AddInfoLog adds an info log entry
func (s *Span) AddInfoLog(message string, fields map[string]interface{}) *Span {
	return s.AddLog("info", message, fields)
}

// AddWarnLog adds a warning log entry
func (s *Span) AddWarnLog(message string, fields map[string]interface{}) *Span {
	return s.AddLog("warn", message, fields)
}

// AddErrorLog adds an error log entry
func (s *Span) AddErrorLog(message string, fields map[string]interface{}) *Span {
	return s.AddLog("error", message, fields)
}

// GetLogs returns all logs
func (s *Span) GetLogs() []SpanLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	logs := make([]SpanLog, len(s.Logs))
	copy(logs, s.Logs)
	return logs
}

// ============================================================================
// Status Operations
// ============================================================================

// SetStatus sets the span status
func (s *Span) SetStatus(code, message string) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = SpanStatus{
		Code:    code,
		Message: message,
	}
	return s
}

// SetStatusOK sets the span status to OK
func (s *Span) SetStatusOK() *Span {
	return s.SetStatus("ok", "")
}

// SetStatusError sets the span status to error
func (s *Span) SetStatusError(message string) *Span {
	return s.SetStatus("error", message)
}

// GetStatus returns the span status
func (s *Span) GetStatus() SpanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// IsError checks if the span has an error status
func (s *Span) IsError() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status.Code == "error"
}

// ============================================================================
// Event Operations
// ============================================================================

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attributes map[string]interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	s.Events = append(s.Events, event)
	return s
}

// GetEvents returns all events
func (s *Span) GetEvents() []SpanEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]SpanEvent, len(s.Events))
	copy(events, s.Events)
	return events
}

// ============================================================================
// Link Operations
// ============================================================================

// AddLink adds a link to another span
func (s *Span) AddLink(traceID, spanID string, attributes map[string]interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()

	link := SpanLink{
		TraceID:    traceID,
		SpanID:     spanID,
		Attributes: attributes,
	}

	s.Links = append(s.Links, link)
	return s
}

// GetLinks returns all links
func (s *Span) GetLinks() []SpanLink {
	s.mu.RLock()
	defer s.mu.RUnlock()
	links := make([]SpanLink, len(s.Links))
	copy(links, s.Links)
	return links
}

// ============================================================================
// Kind Operations
// ============================================================================

// SetKind sets the span kind
func (s *Span) SetKind(kind SpanKind) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Kind = kind
	return s
}

// GetKind returns the span kind
func (s *Span) GetKind() SpanKind {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Kind
}

// ============================================================================
// Resource Operations
// ============================================================================

// SetResource sets a resource attribute
func (s *Span) SetResource(key string, value interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Resource[key] = value
	return s
}

// SetResources sets multiple resource attributes
func (s *Span) SetResources(resources map[string]interface{}) *Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range resources {
		s.Resource[k] = v
	}
	return s
}

// GetResource retrieves a resource attribute
func (s *Span) GetResource(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Resource[key]
	return val, ok
}

// ============================================================================
// Timing Operations
// ============================================================================

// End marks the span as ended and calculates duration
func (s *Span) End() *Span {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime).Microseconds()

	return s
}

// GetDuration returns the span duration
func (s *Span) GetDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}

	return s.EndTime.Sub(s.StartTime)
}

// IsFinished checks if the span has ended
func (s *Span) IsFinished() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.EndTime.IsZero()
}

// ============================================================================
// Conversion Operations
// ============================================================================

// ToJSON converts span to JSON
func (s *Span) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s)
}

// FromJSON creates span from JSON
func FromJSON(data []byte) (*Span, error) {
	var span Span
	if err := json.Unmarshal(data, &span); err != nil {
		return nil, fmt.Errorf("failed to unmarshal span: %w", err)
	}
	return &span, nil
}

// ToAttributes converts span to OpenTelemetry attributes
func (s *Span) ToAttributes() []attribute.KeyValue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	attrs := make([]attribute.KeyValue, 0, len(s.Tags))
	for k, v := range s.Tags {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int64(k, val))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		default:
			attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}

	return attrs
}

// ============================================================================
// Context Operations
// ============================================================================

// ToOtelStatus converts span status to OpenTelemetry status
func (s *Span) ToOtelStatus() (codes.Code, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.Status.Code {
	case "ok":
		return codes.Ok, s.Status.Message
	case "error":
		return codes.Error, s.Status.Message
	default:
		return codes.Unset, s.Status.Message
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

// Clone creates a deep copy of the span
func (s *Span) Clone() *Span {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tags := make(map[string]interface{})
	for k, v := range s.Tags {
		tags[k] = v
	}

	logs := make([]SpanLog, len(s.Logs))
	copy(logs, s.Logs)

	events := make([]SpanEvent, len(s.Events))
	copy(events, s.Events)

	links := make([]SpanLink, len(s.Links))
	copy(links, s.Links)

	resource := make(map[string]interface{})
	for k, v := range s.Resource {
		resource[k] = v
	}

	return &Span{
		TraceID:      s.TraceID,
		SpanID:       s.SpanID,
		ParentSpanID: s.ParentSpanID,
		Operation:    s.Operation,
		ServiceName:  s.ServiceName,
		StartTime:    s.StartTime,
		EndTime:      s.EndTime,
		Duration:     s.Duration,
		Tags:         tags,
		Logs:         logs,
		Events:       events,
		Links:        links,
		Status:       s.Status,
		Kind:         s.Kind,
		Resource:     resource,
	}
}

// String returns a string representation of the span
func (s *Span) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf(
		"Span{TraceID: %s, SpanID: %s, Operation: %s, Duration: %dμs, Status: %s}",
		s.TraceID,
		s.SpanID,
		s.Operation,
		s.Duration,
		s.Status.Code,
	)
}

// ============================================================================
// Validation
// ============================================================================

// Validate validates the span structure
func (s *Span) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TraceID == "" {
		return fmt.Errorf("trace ID is required")
	}

	if s.SpanID == "" {
		return fmt.Errorf("span ID is required")
	}

	if s.Operation == "" {
		return fmt.Errorf("operation name is required")
	}

	if s.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if s.StartTime.IsZero() {
		return fmt.Errorf("start time is required")
	}

	return nil
}

//Personal.AI order the ending
