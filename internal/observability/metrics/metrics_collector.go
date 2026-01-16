// Package metrics provides metrics collection and exposition for OpenEAAP.
// It integrates Prometheus SDK to define and collect core business metrics
// including QPS, latency, error rates, cache hit rates, and more.
package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ============================================================================
// Metrics Collector
// ============================================================================

// MetricsCollector manages Prometheus metrics collection
type MetricsCollector struct {
	// Prometheus registry
	registry *prometheus.Registry

	// Namespace for metrics
	namespace string

	// Subsystem for metrics
	subsystem string

	// Registered metrics
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// CollectorConfig defines metrics collector configuration
type CollectorConfig struct {
	// Namespace for all metrics
	Namespace string

	// Subsystem for metrics grouping
	Subsystem string

	// Enable default Go metrics
	EnableGoMetrics bool

	// Enable process metrics
	EnableProcessMetrics bool

	// Custom registry (optional)
	Registry *prometheus.Registry
}

// ============================================================================
// Collector Initialization
// ============================================================================

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(cfg CollectorConfig) *MetricsCollector {
	registry := cfg.Registry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	// Register default collectors if enabled
	if cfg.EnableGoMetrics {
		registry.MustRegister(prometheus.NewGoCollector())
	}

	if cfg.EnableProcessMetrics {
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	collector := &MetricsCollector{
		registry:   registry,
		namespace:  cfg.Namespace,
		subsystem:  cfg.Subsystem,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
	}

	// Register core business metrics
	collector.registerCoreMetrics()

	return collector
}

// ============================================================================
// Core Business Metrics Registration
// ============================================================================

// registerCoreMetrics registers all core business metrics
func (c *MetricsCollector) registerCoreMetrics() {
	// HTTP metrics
	c.RegisterCounter("http_requests_total", "Total number of HTTP requests", []string{"method", "path", "status"})
	c.RegisterHistogram("http_request_duration_seconds", "HTTP request duration in seconds", []string{"method", "path"}, prometheus.DefBuckets)
	c.RegisterCounter("http_request_errors_total", "Total number of HTTP request errors", []string{"method", "path", "error_type"})

	// Agent metrics
	c.RegisterGauge("agent_active_count", "Number of active agents", []string{"type"})
	c.RegisterCounter("agent_executions_total", "Total number of agent executions", []string{"agent_id", "status"})
	c.RegisterHistogram("agent_execution_duration_seconds", "Agent execution duration in seconds", []string{"agent_id"}, prometheus.DefBuckets)
	c.RegisterCounter("agent_errors_total", "Total number of agent errors", []string{"agent_id", "error_type"})

	// Workflow metrics
	c.RegisterGauge("workflow_active_count", "Number of active workflows", []string{"status"})
	c.RegisterCounter("workflow_executions_total", "Total number of workflow executions", []string{"workflow_id", "status"})
	c.RegisterHistogram("workflow_execution_duration_seconds", "Workflow execution duration in seconds", []string{"workflow_id"}, prometheus.DefBuckets)
	c.RegisterCounter("workflow_step_executions_total", "Total workflow step executions", []string{"workflow_id", "step_id", "status"})

	// Database metrics
	c.RegisterGauge("db_connections_active", "Number of active database connections", []string{"pool"})
	c.RegisterGauge("db_connections_idle", "Number of idle database connections", []string{"pool"})
	c.RegisterCounter("db_queries_total", "Total number of database queries", []string{"operation", "table"})
	c.RegisterHistogram("db_query_duration_seconds", "Database query duration in seconds", []string{"operation", "table"}, prometheus.DefBuckets)
	c.RegisterCounter("db_query_errors_total", "Total database query errors", []string{"operation", "error_type"})

	// Cache metrics
	c.RegisterCounter("cache_requests_total", "Total number of cache requests", []string{"operation", "cache_name"})
	c.RegisterCounter("cache_hits_total", "Total number of cache hits", []string{"cache_name"})
	c.RegisterCounter("cache_misses_total", "Total number of cache misses", []string{"cache_name"})
	c.RegisterHistogram("cache_operation_duration_seconds", "Cache operation duration in seconds", []string{"operation", "cache_name"}, prometheus.DefBuckets)
	c.RegisterGauge("cache_items_count", "Number of items in cache", []string{"cache_name"})
	c.RegisterGauge("cache_memory_bytes", "Cache memory usage in bytes", []string{"cache_name"})

	// Message queue metrics
	c.RegisterCounter("mq_messages_published_total", "Total messages published", []string{"topic"})
	c.RegisterCounter("mq_messages_consumed_total", "Total messages consumed", []string{"topic", "consumer_group"})
	c.RegisterCounter("mq_messages_failed_total", "Total failed messages", []string{"topic", "error_type"})
	c.RegisterGauge("mq_consumer_lag", "Consumer lag in messages", []string{"topic", "consumer_group"})
	c.RegisterHistogram("mq_message_processing_duration_seconds", "Message processing duration", []string{"topic"}, prometheus.DefBuckets)

	// Vector database metrics
	c.RegisterCounter("vector_db_operations_total", "Total vector database operations", []string{"operation", "collection"})
	c.RegisterHistogram("vector_db_operation_duration_seconds", "Vector DB operation duration", []string{"operation", "collection"}, prometheus.DefBuckets)
	c.RegisterCounter("vector_db_search_requests_total", "Total vector search requests", []string{"collection"})
	c.RegisterHistogram("vector_db_search_latency_seconds", "Vector search latency", []string{"collection"}, prometheus.DefBuckets)

	// LLM metrics
	c.RegisterCounter("llm_requests_total", "Total LLM API requests", []string{"provider", "model"})
	c.RegisterCounter("llm_tokens_total", "Total tokens used", []string{"provider", "model", "type"})
	c.RegisterHistogram("llm_request_duration_seconds", "LLM request duration", []string{"provider", "model"}, prometheus.DefBuckets)
	c.RegisterCounter("llm_errors_total", "Total LLM errors", []string{"provider", "model", "error_type"})
	c.RegisterGauge("llm_rate_limit_remaining", "Remaining rate limit", []string{"provider", "model"})

	// System metrics
	c.RegisterGauge("system_goroutines_count", "Number of goroutines", nil)
	c.RegisterGauge("system_memory_alloc_bytes", "Allocated memory in bytes", nil)
	c.RegisterGauge("system_memory_sys_bytes", "System memory in bytes", nil)
	c.RegisterCounter("system_gc_runs_total", "Total garbage collection runs", nil)
}

// ============================================================================
// Counter Operations
// ============================================================================

// RegisterCounter registers a new counter metric
func (c *MetricsCollector) RegisterCounter(name, help string, labels []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.counters[name]; exists {
		return
	}

	counter := promauto.With(c.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	c.counters[name] = counter
}

// IncrementCounter increments a counter by 1
func (c *MetricsCollector) IncrementCounter(name string, labels prometheus.Labels) {
	c.mu.RLock()
	counter, exists := c.counters[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	counter.With(labels).Inc()
}

// AddCounter adds a value to a counter
func (c *MetricsCollector) AddCounter(name string, value float64, labels prometheus.Labels) {
	c.mu.RLock()
	counter, exists := c.counters[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	counter.With(labels).Add(value)
}

// ============================================================================
// Gauge Operations
// ============================================================================

// RegisterGauge registers a new gauge metric
func (c *MetricsCollector) RegisterGauge(name, help string, labels []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.gauges[name]; exists {
		return
	}

	gauge := promauto.With(c.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	c.gauges[name] = gauge
}

// SetGauge sets a gauge value
func (c *MetricsCollector) SetGauge(name string, value float64, labels prometheus.Labels) {
	c.mu.RLock()
	gauge, exists := c.gauges[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	gauge.With(labels).Set(value)
}

// IncrementGauge increments a gauge by 1
func (c *MetricsCollector) IncrementGauge(name string, labels prometheus.Labels) {
	c.mu.RLock()
	gauge, exists := c.gauges[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	gauge.With(labels).Inc()
}

// DecrementGauge decrements a gauge by 1
func (c *MetricsCollector) DecrementGauge(name string, labels prometheus.Labels) {
	c.mu.RLock()
	gauge, exists := c.gauges[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	gauge.With(labels).Dec()
}

// AddGauge adds a value to a gauge
func (c *MetricsCollector) AddGauge(name string, value float64, labels prometheus.Labels) {
	c.mu.RLock()
	gauge, exists := c.gauges[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	gauge.With(labels).Add(value)
}

// ============================================================================
// Histogram Operations
// ============================================================================

// RegisterHistogram registers a new histogram metric
func (c *MetricsCollector) RegisterHistogram(name, help string, labels []string, buckets []float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.histograms[name]; exists {
		return
	}

	histogram := promauto.With(c.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	c.histograms[name] = histogram
}

// ObserveHistogram records a value in histogram
func (c *MetricsCollector) ObserveHistogram(name string, value float64, labels prometheus.Labels) {
	c.mu.RLock()
	histogram, exists := c.histograms[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	histogram.With(labels).Observe(value)
}

// ObserveDuration records duration in histogram
func (c *MetricsCollector) ObserveDuration(name string, start time.Time, labels prometheus.Labels) {
	duration := time.Since(start).Seconds()
	c.ObserveHistogram(name, duration, labels)
}

// ============================================================================
// Summary Operations
// ============================================================================

// RegisterSummary registers a new summary metric
func (c *MetricsCollector) RegisterSummary(name, help string, labels []string, objectives map[float64]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.summaries[name]; exists {
		return
	}

	summary := promauto.With(c.registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  c.namespace,
			Subsystem:  c.subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)

	c.summaries[name] = summary
}

// ObserveSummary records a value in summary
func (c *MetricsCollector) ObserveSummary(name string, value float64, labels prometheus.Labels) {
	c.mu.RLock()
	summary, exists := c.summaries[name]
	c.mu.RUnlock()

	if !exists {
		return
	}

	summary.With(labels).Observe(value)
}

// ============================================================================
// HTTP Handler
// ============================================================================

// Handler returns HTTP handler for metrics exposition
func (c *MetricsCollector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// ServeHTTP implements http.Handler interface
func (c *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.Handler().ServeHTTP(w, r)
}

// ============================================================================
// Utility Methods
// ============================================================================

// RecordHTTPRequest records HTTP request metrics
func (c *MetricsCollector) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": fmt.Sprintf("%d", statusCode),
	}

	c.IncrementCounter("http_requests_total", labels)
	c.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), prometheus.Labels{
		"method": method,
		"path":   path,
	})

	if statusCode >= 400 {
		errorType := "client_error"
		if statusCode >= 500 {
			errorType = "server_error"
		}
		c.IncrementCounter("http_request_errors_total", prometheus.Labels{
			"method":     method,
			"path":       path,
			"error_type": errorType,
		})
	}
}

// RecordCacheHit records cache hit
func (c *MetricsCollector) RecordCacheHit(cacheName string) {
	c.IncrementCounter("cache_requests_total", prometheus.Labels{
		"operation":  "get",
		"cache_name": cacheName,
	})
	c.IncrementCounter("cache_hits_total", prometheus.Labels{
		"cache_name": cacheName,
	})
}

// RecordCacheMiss records cache miss
func (c *MetricsCollector) RecordCacheMiss(cacheName string) {
	c.IncrementCounter("cache_requests_total", prometheus.Labels{
		"operation":  "get",
		"cache_name": cacheName,
	})
	c.IncrementCounter("cache_misses_total", prometheus.Labels{
		"cache_name": cacheName,
	})
}

// CalculateCacheHitRate returns cache hit rate
func (c *MetricsCollector) CalculateCacheHitRate(cacheName string) float64 {
	c.mu.RLock()
	hitsCounter := c.counters["cache_hits_total"]
	missesCounter := c.counters["cache_misses_total"]
	c.mu.RUnlock()

	if hitsCounter == nil || missesCounter == nil {
		return 0
	}

	// labels := prometheus.Labels{"cache_name": cacheName}
	// Note: In production, you'd need to collect metrics data
	// This is a simplified example
	return 0
}

// RecordDBQuery records database query metrics
func (c *MetricsCollector) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	labels := prometheus.Labels{
		"operation": operation,
		"table":     table,
	}

	c.IncrementCounter("db_queries_total", labels)
	c.ObserveHistogram("db_query_duration_seconds", duration.Seconds(), labels)

	if err != nil {
		c.IncrementCounter("db_query_errors_total", prometheus.Labels{
			"operation":  operation,
			"error_type": "query_error",
		})
	}
}

// RecordLLMRequest records LLM API request metrics
func (c *MetricsCollector) RecordLLMRequest(provider, model string, tokens int, duration time.Duration, err error) {
	labels := prometheus.Labels{
		"provider": provider,
		"model":    model,
	}

	c.IncrementCounter("llm_requests_total", labels)
	c.ObserveHistogram("llm_request_duration_seconds", duration.Seconds(), labels)
	c.AddCounter("llm_tokens_total", float64(tokens), prometheus.Labels{
		"provider": provider,
		"model":    model,
		"type":     "total",
	})

	if err != nil {
		c.IncrementCounter("llm_errors_total", prometheus.Labels{
			"provider":   provider,
			"model":      model,
			"error_type": "api_error",
		})
	}
}

// ============================================================================
// Global Collector
// ============================================================================

var globalCollector *MetricsCollector
var once sync.Once

// GetGlobalCollector returns the global metrics collector
func GetGlobalCollector() *MetricsCollector {
	once.Do(func() {
		globalCollector = NewMetricsCollector(CollectorConfig{
			Namespace:            "openeaap",
			EnableGoMetrics:      true,
			EnableProcessMetrics: true,
		})
	})
	return globalCollector
}

// SetGlobalCollector sets the global metrics collector
func SetGlobalCollector(collector *MetricsCollector) {
	globalCollector = collector
}

//Personal.AI order the ending
